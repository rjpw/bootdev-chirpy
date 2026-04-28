# Domain-First Development

How to add a feature to Chirpy using a red-green-refactor loop that starts from the domain and works toward the system.

For the principles behind this workflow — why domain over system, how to think about FRs and NFRs, what makes an API contract — see [design-governance.md](design-governance.md).


## The two loops

```
  ┌─────────────────────────────────────────────────────────┐
  │  FAST LOOP (no Docker, sub-second tests)                │
  │                                                         │
  │  Requirement → API test (red) → Interface →             │
  │  Memory store → Handler (green) → Refactor              │
  └──────────────────────────┬──────────────────────────────┘
                             │ design settled
                             ▼
  ┌─────────────────────────────────────────────────────────┐
  │  SLOW LOOP (Docker, real Postgres)                      │
  │                                                         │
  │  Migration → Queries → sqlc → Postgres store →          │
  │  Integration tests (relational behavior only)           │
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


## Step 1: Write the API test (red)

Start from the requirement. Express it as an HTTP test:

```go
func TestCreateChirp(t *testing.T) {
    srv := newTestServer("dev")

    body := `{"body": "hello world"}`
    r := httptest.NewRequest("POST", "/api/chirps", strings.NewReader(body))
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, r)

    if w.Code != http.StatusCreated {
        t.Errorf("want 201, got %d", w.Code)
    }
}
```

This won't compile. Good. The test tells you what to build.

Write tests for the error cases too — validation failures, conflicts, not-found. These are part of the API contract and should be locked in from the start.


## Step 2: Define the interface

What does the handler need from the store? Write the minimum interface:

```go
type ChirpStore interface {
    CreateChirp(ctx context.Context, body string, authorID uuid.UUID) (*Chirp, error)
}
```

Design rules (from [governance](design-governance.md)):
- Method names use domain verbs: `Create`, `List`, `Get` — not `Insert`, `Select`
- Parameters are what the caller knows, not what the database needs
- The interface should make sense to someone who's never seen the schema
- If the governance checklist identified pagination, include it now: `ListChirps(ctx, cursor) ([]Chirp, NextCursor, error)`

Define the domain type:

```go
type Chirp struct {
    ID        uuid.UUID `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    Body      string    `json:"body"`
    AuthorID  uuid.UUID `json:"author_id"`
}
```

This is what the application thinks about. It may not match the database columns.


## Step 3: Implement the memory store (green)

Make the test pass with the simplest thing that works:

```go
func (s *Store) CreateChirp(_ context.Context, body string, authorID uuid.UUID) (*store.Chirp, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    chirp := store.Chirp{
        ID:        uuid.New(),
        CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
        Body:      body,
        AuthorID:  authorID,
    }
    s.chirps[chirp.ID] = chirp
    return &chirp, nil
}
```

Write the handler. Wire it into the router. Run the test. Green.

The memory store should be natural to write. If it's not — if you're simulating joins, faking pagination cursors, or enforcing foreign keys — the interface is leaking system concerns. Fix the interface, not the memory store.


## Step 4: Refactor

The feature works against the memory store. Now pressure-test the design:

- Does the interface have methods the handler doesn't call? Remove them.
- Does the domain type have fields the handler doesn't use? Remove them.
- Does the response shape match what you committed to in the governance checklist?
- If you identified an NFR (pagination, uniqueness), is it reflected in the interface?
- Does the handler orchestrate more than one concern? If it calls a store method *and* does something else (hashes a password, issues a token, sends a notification), extract a service method. The handler should delegate to a single call, not coordinate a multi-step workflow. See below.

This is the cheapest moment to change the design. No migration to undo, no sqlc to regenerate, no integration tests to update.


### When to extract a service method

A handler that parses a request, calls one store method, and writes a response is fine as-is. No service layer needed.

The signal is when the handler orchestrates across concerns:

```go
// This handler is doing too much
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
    // parse request
    // look up user by email        ← store concern
    // compare password hash         ← auth concern
    // issue JWT                     ← auth concern
    // write response
}
```

Steps 2–4 are a unit of business logic that doesn't belong in the HTTP layer. Extract it:

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

The handler becomes thin again:

```go
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
    // parse request
    token, err := s.svc.Login(r.Context(), email, password)
    // map err to status code, write response
}
```

The service struct holds the dependencies that cross-concern operations need:

```go
// internal/service/service.go
type Service struct {
    users     store.UserStore
    jwtSecret string
}
```

One `service` package, one struct, methods split by file (`users.go`, `chirps.go`). Don't split into separate service packages until methods have completely disjoint dependencies — for a project this size, that's unlikely.

The rule: if a handler calls two packages to fulfill one request, introduce a service method. If it calls one, leave it alone.


## Step 5: Add the database layer

The behavior is proven. Make it durable:

1. `make sql-create` — write the migration DDL
2. Write queries in `internal/schema/queries/<entity>.sql`
3. `make sql-generate`
4. Implement `postgres/<entity>.go` — `toStore*` mapper, interface methods, compile-time check
5. Wire into `Stores` and `main.go`

This is translation, not design. The interface already exists. The domain type already exists. You're mapping between sqlc's generated types and your domain types.

The `toStore*` mapper is the seam between the two worlds. It's the only code that knows about both `database.Chirp` and `store.Chirp`. When sqlc regenerates, fix the mapper and nothing else changes.


## Step 6: Integration tests

Not every store method needs an integration test. The API tests already proved the domain behavior. Integration tests verify two things:

### Translation correctness

Does the postgres store correctly bridge domain operations and SQL?

- Error mapping: unique violation → `ErrConflict`, no rows → `ErrNotFound`
- Field mapping: does `toStoreChirp` round-trip all fields correctly?
- Multi-step operations: does a method that coordinates multiple queries work with committed data?

Skip these for trivial pass-throughs (one sqlc call, one mapper).

### Relational behavior

Some behaviors only exist in the database. The memory store can't express them and shouldn't try:

- **Cascades**: delete a user → their chirps disappear (or the delete is rejected)
- **Referential integrity**: create a chirp for a nonexistent user → FK violation
- **Compound uniqueness**: one vote per user per chirp
- **Joins**: chirps returned with author info attached

Write these when you add a migration that creates a relationship. They test the DDL, not the Go code. The rule of thumb: if the behavior depends on a SQL clause (`ON DELETE CASCADE`, `REFERENCES`, `UNIQUE(a, b)`), it needs an integration test.


## The memory store is not scaffolding

It serves three ongoing purposes:

1. **Fast API tests** — `newTestServer` uses it. Sub-millisecond, no Docker, no cleanup.
2. **Interface design feedback** — if it's awkward to implement, the interface is wrong. The memory store is the first customer for every interface, and the most honest one.
3. **Test isolation** — each test gets a fresh store. No shared state, no snapshot/restore, no rollback. Tests are independent and parallelizable.

The postgres store is the production implementation. The memory store is the development implementation. Both are first-class.
