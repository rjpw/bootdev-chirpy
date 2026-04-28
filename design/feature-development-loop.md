# Test-Driven Development at Hexagonal Boundaries

How to add a feature to Chirpy using TDD, where the tests enforce the hexagonal architecture. Each test lives at a boundary and verifies the translation that boundary performs. The tests are the architecture.

For the principles behind this workflow — why domain over system, how to think about FRs and NFRs, what makes an API contract — see [design-governance.md](design-governance.md).

For the testing philosophy in detail — why each layer tests only its own translation — see [testing-at-hex-boundaries.md](testing-at-hex-boundaries.md).

Previous version: [archive/feature-development-loop.md](archive/feature-development-loop.md)


## The architecture the tests enforce

```
  ┌─────────────────────────────────────────────────────────┐
  │  httpapi_test        (package httpapi_test)              │
  │                                                         │
  │  Speaks: HTTP in, HTTP out                              │
  │  Trusts: the domain port contract                       │
  │  Catches: JSON parsing, status codes, error mapping,    │
  │           routing, content types, response shape         │
  ├─────────────────────────────────────────────────────────┤
  │  httpapi             (the driving adapter)               │
  │  Exported surface: NewServer, ServeHTTP — nothing else   │
  └──────────────────────────┬──────────────────────────────┘
                             │ calls domain ports
                             ▼
  ┌─────────────────────────────────────────────────────────┐
  │  domain              (types, ports, errors, rules)       │
  │                                                         │
  │  domain_test: business rules, validation, computation    │
  │  No adapters. No HTTP. No SQL.                          │
  └──────────────────────────┬──────────────────────────────┘
                             │ implemented by
              ┌──────────────┴──────────────┐
              ▼                             ▼
  ┌─────────────────────┐      ┌─────────────────────┐
  │  memory_test         │      │  postgres_test       │
  │                     │      │  (integration)       │
  │  Speaks: domain     │      │  Speaks: domain      │
  │  Catches: contract  │      │  Catches: contract,  │
  │  compliance         │      │  SQL, error mapping,  │
  │                     │      │  relational behavior  │
  └─────────────────────┘      └─────────────────────┘
```

Every test package uses the external test package convention (`_test` suffix). This is a compiler-enforced guardrail: if a test can't access unexported symbols, it can't cross a boundary.


## The two loops

```
  ┌─────────────────────────────────────────────────────────┐
  │  FAST LOOP (no Docker, sub-second)                      │
  │                                                         │
  │  1. Domain type + port (interface)                      │
  │  2. Memory repository (port contract test → green)      │
  │  3. HTTP handler (HTTP boundary test → green)           │
  │  4. Refactor                                            │
  └──────────────────────────┬──────────────────────────────┘
                             │ design settled
                             ▼
  ┌─────────────────────────────────────────────────────────┐
  │  SLOW LOOP (Docker, real Postgres)                      │
  │                                                         │
  │  5. Migration + queries + sqlc                          │
  │  6. Postgres repository (port contract test → green)    │
  │  7. Relational behavior tests (cascades, FKs)           │
  └─────────────────────────────────────────────────────────┘
```

Stay in the fast loop until the behavior is right. Drop to the slow loop to make it durable.


## Before you start: the governance checklist

Before writing any code, work through the checklist in [design-governance.md](design-governance.md). You should be able to answer:

- What domain operations does this entity support?
- What does the endpoint look like?
- What error cases exist?
- Does this list endpoint need pagination? (Decide now, not later.)
- What happens when a related entity is deleted?
- Is the response shape deliberate?

These answers inform every step below. Skipping them means discovering constraints mid-implementation, when changes are expensive.


## Step 1: Define the domain type and port

Create the type and interface in `internal/domain/`:

```go
// internal/domain/chirp.go
type Chirp struct {
    ID        uuid.UUID
    CreatedAt time.Time
    Body      string
    AuthorID  uuid.UUID
}

type ChirpRepository interface {
    CreateChirp(ctx context.Context, body string, authorID uuid.UUID) (*Chirp, error)
}
```

Design rules:
- Method names use domain verbs: `Create`, `List`, `Get` — not `Insert`, `Select`
- Parameters are what the caller knows, not what the database needs
- The interface should make sense to someone who's never seen the schema
- If the governance checklist identified pagination, include it now

This step produces no tests. The type and interface are the specification.

**Guardrail check:** `domain/` imports nothing from `internal/`. If it does, the dependency rule is broken.


## Step 2: Implement the memory repository (red → green)

Write the port contract test first, in `package memory_test`:

```go
// internal/memory/chirps_test.go
package memory_test

func TestCreateChirp(t *testing.T) {
    repo := memory.NewRepository()
    chirp, err := repo.CreateChirp(ctx, "hello world", userID)
    // assert: no error, chirp.Body == "hello world", chirp.ID is set, etc.
}
```

This test speaks the domain language. It calls a domain method, gets a domain type, checks domain errors. No HTTP. No SQL.

Then implement the memory repository to make it green:

```go
// internal/memory/chirps.go
func (r *Repository) CreateChirp(_ context.Context, body string, authorID uuid.UUID) (*domain.Chirp, error) {
    // ...
}
```

**Guardrail check:** The test is in `package memory_test`. It can only use exported symbols. The repository satisfies `domain.ChirpRepository` (compile-time check).


## Step 3: Write the HTTP boundary test (red → green)

Write the HTTP test first, in `package httpapi_test`:

```go
// internal/httpapi/chirps_test.go
package httpapi_test

func TestCreateChirp(t *testing.T) {
    srv := httpapi.NewServer(cfg, "./testdata")
    body := `{"body": "hello world", "author_id": "..."}`
    r := httptest.NewRequest("POST", "/api/chirps", strings.NewReader(body))
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, r)

    if w.Code != http.StatusCreated {
        t.Errorf("want 201, got %d", w.Code)
    }
    // decode w.Body, check JSON shape
}

func TestCreateChirpBadJSON(t *testing.T) {
    // POST garbage → 400
}

func TestCreateChirpValidation(t *testing.T) {
    // POST body > 140 chars → 400
}
```

This test speaks HTTP. It sends a request, checks the status code and response body. It does not call domain methods. It does not inspect domain types. It does not check `domain.ErrConflict` — it checks for HTTP 409.

The test uses a memory repository under the hood (via `newTestServer`), but it doesn't know or care. The repository is an implementation detail of the test fixture.

Then implement the handler:

```go
// internal/httpapi/chirps.go
func (s *Server) handleCreateChirp(w http.ResponseWriter, r *http.Request) {
    // parse JSON body
    // call s.cfg.Chirps.CreateChirp(r.Context(), ...)
    // map domain errors to HTTP status codes
    // serialize response as JSON
}
```

Register the route. Run the test. Green.

**Guardrail checks:**
- The test is in `package httpapi_test`. It can only call `NewServer` and `ServeHTTP`.
- The handler does not expose domain methods on `Server`. No `Server.CreateChirp`.
- The handler calls the repository through the config, not through a forwarding method.


## Step 4: Refactor

The feature works against the memory repository, tested at both boundaries. Pressure-test the design:

- Does the domain interface have methods the handler doesn't call? Remove them.
- Does the domain type have fields the handler doesn't use? Remove them.
- Does the response shape match the governance checklist?
- Does the handler orchestrate more than one concern? If it calls a repository *and* does something else (hashes a password, issues a token), extract a service method in `domain/` or `internal/service/`. The handler should delegate, not coordinate.

This is the cheapest moment to change the design. No migration to undo, no sqlc to regenerate, no integration tests to update.

### When to extract a service method

A handler that parses a request, calls one repository method, and writes a response is fine as-is.

The signal is when the handler orchestrates across concerns:

```go
// This handler is doing too much
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
    // parse request
    // look up user by email        ← repository concern
    // compare password hash         ← auth concern
    // issue JWT                     ← auth concern
    // write response
}
```

Extract the business logic:

```go
// internal/service/users.go
func (svc *Service) Login(ctx context.Context, email, password string) (string, error) {
    user, err := svc.users.GetUserByEmail(ctx, email)
    if err != nil { return "", err }
    if err := auth.ComparePassword(user.PasswordHash, password); err != nil {
        return "", ErrUnauthorized
    }
    return auth.IssueToken(user.ID, svc.jwtSecret)
}
```

The service struct depends on domain ports, not adapters:

```go
type Service struct {
    users     domain.UserRepository
    jwtSecret string
}
```

The handler becomes thin:

```go
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
    // parse request
    token, err := s.svc.Login(r.Context(), email, password)
    // map err to status code, write response
}
```

The rule: if a handler calls two packages to fulfill one request, introduce a service method. If it calls one, leave it alone.


## Step 5: Add the database layer

The behavior is proven. Make it durable:

1. `make sql-create` — write the migration DDL
2. Write queries in `internal/postgres/schema/queries/<entity>.sql`
3. `make sql-generate`
4. Implement `postgres/<entity>.go` — `toDomain*` mapper, interface methods, compile-time check
5. Wire into `Repositories` and `main.go`

This is translation, not design. The interface already exists. The domain type already exists. You're mapping between sqlc's generated types and your domain types.


## Step 6: Port contract test for Postgres (red → green)

Write the same contract tests against the real database:

```go
// internal/postgres/chirps_test.go
package postgres_test

func TestCreateChirp(t *testing.T) {
    db := testdb.Setup(t)
    repo := postgres.NewRepository(database.New(db.DB))
    chirp, err := repo.CreateChirp(ctx, "hello world", userID)
    // same assertions as the memory test
}
```

The test speaks the same domain language as the memory test. It calls the same methods, checks the same types, expects the same errors. The only difference is the adapter under test.

**Guardrail check:** The test is in `package postgres_test`. It uses `testdb.Setup` for the container. It verifies the same contract the memory tests verify.

### Shared contract tests (optional refinement)

If you want to guarantee both implementations satisfy the same contract:

```go
// internal/domain/repositorytest/chirp.go
func TestChirpRepository(t *testing.T, repo domain.ChirpRepository) {
    t.Run("CreateChirp", func(t *testing.T) { ... })
    t.Run("CreateChirpDuplicate", func(t *testing.T) { ... })
}
```

Called from both:

```go
// memory/chirps_test.go
func TestChirpContract(t *testing.T) {
    repositorytest.TestChirpRepository(t, memory.NewRepository())
}

// postgres/chirps_test.go
func TestChirpContract(t *testing.T) {
    db := testdb.Setup(t)
    repositorytest.TestChirpRepository(t, postgres.NewRepository(database.New(db.DB)))
}
```

This is not required for every entity. Use it when the contract has enough cases that duplicating them feels wrong.


## Step 7: Relational behavior tests

Some behaviors only exist in the database. The memory repository can't express them and shouldn't try:

- **Cascades**: delete a user → their chirps disappear
- **Referential integrity**: create a chirp for a nonexistent user → FK violation
- **Compound uniqueness**: one vote per user per chirp

These are Postgres-only tests. They test the DDL, not the Go code. Write them when you add a migration that creates a relationship.


## Summary: what each test package proves

| Test package | Proves | Speaks | Trusts |
|---|---|---|---|
| `domain_test` | Business rules are correct | Domain types | Nothing |
| `memory_test` | Memory adapter satisfies the port | Domain methods | Domain types |
| `postgres_test` | Postgres adapter satisfies the port; SQL is correct | Domain methods | Domain types, real DB |
| `httpapi_test` | HTTP translation is correct | HTTP requests/responses | Domain port contract |

Each row tests one boundary. No row tests two. The `_test` suffix on every package makes this enforceable by the compiler.


## The memory repository is not scaffolding

It serves three ongoing purposes:

1. **Fast HTTP tests** — `newTestServer` uses it. Sub-millisecond, no Docker, no cleanup.
2. **Interface design feedback** — if it's awkward to implement, the interface is wrong. The memory repository is the first customer for every interface, and the most honest one.
3. **Test isolation** — each test gets a fresh repository. No shared state, no snapshot/restore, no rollback. Tests are independent and parallelizable.

The postgres repository is the production implementation. The memory repository is the development implementation. Both are first-class.
