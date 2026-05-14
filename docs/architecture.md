# Architecture

## Hexagonal Layers

The project follows hexagonal architecture with automated boundary enforcement.
Each package under `internal/` has a role; the dependency rule is enforced by
`hex_guardrail_test.go` using data from `testdata/hex_roles.json` and `testdata/hex_rules.json`.

### Domain (`internal/domain`)

Pure business types and logic. No dependencies on other internal packages.

| File | Contents |
|------|----------|
| `domain.go` | `ShortID` helper |
| `user.go` | `User`, `UserCredentials`, `UserSession` structs |
| `chirp.go` | `Chirp` struct, `FilterChirp` (profanity filter + truncation) |
| `errors.go` | Sentinel errors: `ErrNotFound`, `ErrConflict`, `ErrUnauthorized` |

### Application (`internal/application`)

Interfaces, environment, and cross-cutting contracts. Depends only on `domain`.

| File | Contents |
|------|----------|
| `application.go` | `Runnable` interface, `Environment` struct, `Repositories` aggregate, `UserRepository` / `ChirpRepository` / `UserSessionRepository` interfaces, `LoadEnvironment()` |
| `metrics.go` | `ServerMetrics` (concrete, atomic counter + middleware) |

### Application — Auth (`internal/auth`)

Cryptographic primitives: password hashing (argon2id), JWT creation/validation,
refresh token generation, header parsing. Classified as `application` in hex roles.

### Adapters

#### HTTP (`internal/httpapi`)

Driving adapter. Translates HTTP into repository calls.

| File | Contents |
|------|----------|
| `router.go` | `ChirpyAPIRouter` struct (wraps `http.ServeMux`, satisfies `http.Handler`), route registration, `respondWithJSON`, `respondWithMessage`, `handleReset`, `handleHealthz`, `handleMetrics` |
| `user_handlers.go` | `handleCreateUser`, `handleUpdateUser`, `handleLogin`, `handleSessionRefresh`, `handleSessionRevoke` |
| `chirp_handlers.go` | `handleCreateChirp`, `handleGetChirps`, `handleGetChirp`, `handleDeleteChirp` |
| `middleware.go` | `withValidBody[T]` generic validation middleware, `Validatable` interface, `validBody[T]` context extractor |

#### Memory (`internal/memory`)

In-memory driven adapter for fast tests. Single `Repository` struct satisfies all three repository interfaces.

| File | Contents |
|------|----------|
| `repository.go` | `Repository` struct (mutex + maps), `NewMemoryRepository()` |
| `users.go` | `UserRepository` implementation |
| `chirps.go` | `ChirpRepository` implementation |
| `usersessions.go` | `UserSessionRepository` implementation |

#### Postgres (`internal/postgres`)

Production driven adapter. Single `Repository` struct satisfies all three repository interfaces.

| File | Contents |
|------|----------|
| `db.go` | `Open(url)`, `NewRepositoryFromURL(url)` — connection factories |
| `repository.go` | `Repository` struct (wraps sqlc `Queries`), `mapError` (PG error codes → domain errors) |
| `users.go` | `UserRepository` implementation |
| `chirps.go` | `ChirpRepository` implementation |
| `usersessions.go` | `UserSessionRepository` implementation |
| `database/` | sqlc-generated query code |
| `schema/` | Embedded migration files |
| `testdb/` | Testcontainer helper (see Testing below) |

#### Operations (`internal/operations`)

Operational adapter for CLI subcommands.

| File | Contents |
|------|----------|
| `migrator.go` | `Migrator` struct (satisfies `Runnable`), runs goose up/status |

### Assembly (`internal/config`)

Composition root. Wires adapters to application interfaces. Only package that imports both adapters and application.

| File | Contents |
|------|----------|
| `config.go` | `Service` struct (satisfies `Runnable`, wraps `http.Server`), `NewRunner(env, staticPath)` factory, `NewMigrator(env, command)` factory |

### Binary (`cmd/chirpy`)

| File | Contents |
|------|----------|
| `main.go` | Loads environment, selects `Runnable` (server or migrator), runs with signal context. ~40 lines. |

---

## Dependency Rules

```
domain      → (nothing)
application → domain
adapter     → application, domain
assembly    → application, adapter
```

Enforced at build time by `hex_guardrail_test.go`. Run with `make test-hex-guardrail`.

---

## Testing

### Fast Tests (`make test`)

No external dependencies. Run with `go test -race ./...`.

| Package | What's tested | Backend |
|---------|---------------|---------|
| `internal/domain` | `FilterChirp` logic | Pure functions |
| `internal/auth` | JWT, password hashing, token parsing | Pure functions |
| `internal/memory` | Repository contract | In-memory |
| `internal/httpapi` | Full HTTP workflows (create user → login → chirp → refresh → revoke) | Memory repos via `ChirpyAPIRouter` |
| root | Hex guardrail | `go list -json` import graph |

### Integration Tests (`make test-integration`)

Require Docker. Run with `go test -race -tags integration -count=1 ./...`.

| Package | What's tested | Backend |
|---------|---------------|---------|
| `internal/postgres` | Repository contract against real Postgres | Testcontainer |
| `internal/postgres/database` | sqlc-generated queries | Testcontainer |
| `internal/postgres/testdb` | Testcontainer helper itself | Testcontainer |
| `cmd/chirpy` | Full-stack workflows through `config.NewRunner` | Testcontainer |

### Test Infrastructure

#### `internal/postgres/testdb`

Manages a single Postgres testcontainer per test process (via `sync.Once`).
Applies migrations, snapshots the container for fast restore between tests.

- `SetupTestHelperDB(t)` — for use in individual test functions
- `SetupTestsuiteDB()` — for use in `TestMain` (no `*testing.T` required)

#### `internal/testutil`

Shared stateful API client for workflow tests. Decoupled from concrete router type — takes `http.Handler`.

- `InternalAPIClient` — holds access token, session ID, email
- Happy-path methods (`CreateUser`, `Login`, `Chirp`) assert success internally
- `Try*` variants return the response without asserting — for error-path testing
- `IssueRequest`, `IssueAuthorizedRequest` — low-level helpers
- `AssertStatus`, `Decode`, `Marshal` — assertion/serialization utilities

Used by both `internal/httpapi` tests (against memory) and `cmd/chirpy` integration tests (against Postgres).

---

## Runtime Topology

```
cmd/chirpy/main.go
  → application.LoadEnvironment()
  → config.NewRunner(env, staticPath)
      → postgres.NewRepositoryFromURL(url)  → *postgres.Repository, *sql.DB
      → application.Repositories{...}
      → httpapi.NewRouter(env, metrics, repos, staticPath)  → *ChirpyAPIRouter
      → http.Server{Handler: router}
      → config.Service{httpServer, close: db.Close}
  → service.Run(ctx)  // ListenAndServe + graceful shutdown on signal
```

Migrator path:
```
cmd/chirpy/main.go (with "migrate" arg)
  → config.NewMigrator(env, command)
      → postgres.Open(url)
      → schema.Migrations (embedded FS)
      → operations.NewMigrator(db, fs, command)
  → migrator.Run(ctx)  // goose up or status
```
