# Roadmap Checklist

Progress tracker for the roadmap docs. Tick items as they're completed in code.


## Doc 01 — Store Interface

- [x] `internal/store/store.go` — `User` type, `UserStore` interface
- [x] `internal/store/errors.go` — `ErrNotFound`, `ErrConflict`
- [x] `go build ./internal/store/...` compiles


## Doc 02 — Memory Store

- [x] `internal/store/memory/memory.go` — `MemoryStore` implementing `UserStore`
- [x] Compile-time interface check: `var _ store.UserStore = (*MemoryStore)(nil)`
- [x] `internal/store/memory/memory_test.go` — external test package
- [x] `TestCreateUser` verifies ID, email, timestamps
- [x] `TestCreateUserDuplicateEmail` checks `ErrConflict`
- [x] Tests use `store.UserStore` as the variable type
- [x] `go test -race ./internal/store/memory/...` passes


## Doc 03 — Wire and Integrate

- [x] `config.Config` has `Users store.UserStore` field
- [x] `main.go` wires a store implementation into config
- [x] Server compiles and starts
- [x] API handler tests use a fake or memory store (not Postgres)


## Doc 04 — Testdb Helper

- [x] `internal/testdb/` package exists
- [x] `Setup` function starts a Postgres container
- [x] Migrations embedded via `embed.FS`
- [x] `sync.Once` ensures single container per test run
- [x] `t.Cleanup` tears down the container (skipped — Ryuk handles it)
- [x] Verification test passes


## Doc 05 — sqlc Query Tests

- [x] `internal/database/queries_test.go` exists
- [x] Tests use testdb helper
- [x] Transaction rollback isolation
- [x] `TestCreateUser` against real Postgres
- [x] `pq.Error` inspection for unique constraint


## Doc 06 — Postgres Store

- [x] `internal/store/postgres/store.go` — `Store` implementing `UserStore`
- [x] Compile-time interface check: `var _ store.UserStore = (*Store)(nil)`
- [x] `mapError` translates `sql.ErrNoRows` → `ErrNotFound`, pq `23505` → `ErrConflict`
- [x] Timestamps use `UTC().Truncate(time.Microsecond)`
- [x] Snapshot/restore integration tests
- [x] Tests run via testcontainers (not local Postgres)


## Doc 07 — Developer Workflow

- [x] Integration tests gated behind `//go:build integration`
- [x] `make test` runs only fast tests
- [x] `make test-integration` runs everything
- [x] `make sql-create` generates timestamped migrations
- [x] `make sql-fix` renumbers for release
- [x] README documents the testing approach


## Doc 08 — Migrate Subcommand

- [x] `./chirpy migrate up` applies pending migrations
- [x] `./chirpy migrate status` shows migration state
- [x] SQL files embedded in binary via `embed.FS`
- [x] `./chirpy` (no args) still starts the server
- [x] README documents the subcommand


## Doc 09 — Hexagonal Restructure

### Step 1: Extract `domain/` from `store/` and rename to Repository

- [x] `internal/domain/user.go` — `User` type, `UserRepository` interface
- [x] `internal/domain/errors.go` — `ErrNotFound`, `ErrConflict`
- [x] `UserStore` renamed to `UserRepository` throughout
- [x] `domain/` has zero imports from `internal/`
- [x] All imports updated (`store.User` → `domain.User`, etc.)
- [x] `make test` passes

### Step 2: Promote adapters and rename to Repository

- [x] `internal/memory/` (moved from `store/memory/`), struct renamed to `Repository`
- [x] `internal/postgres/` (moved from `store/postgres/`), struct renamed to `Repository`
- [x] `internal/store/` directory removed
- [x] All imports updated
- [x] `make test` passes

### Step 3: Nest Postgres infrastructure

- [x] `internal/postgres/database/` (moved from `internal/database/`)
- [x] `internal/postgres/schema/` (moved from `internal/schema/`)
- [x] `internal/postgres/testdb/` (moved from `internal/testdb/`)
- [x] `sqlc.yaml` paths updated
- [x] `Makefile` paths updated
- [x] `make sql-generate` works
- [x] `make test` passes

### Step 4: Rename `api/` to `httpapi/`

- [x] `internal/httpapi/` (renamed from `api/`)
- [x] Package declarations and imports updated
- [x] `make test` passes

### Step 5: Verify dependency rule

- [x] `domain/` imports nothing from `internal/`
- [x] `postgres/` and `memory/` import only `domain/` (not each other, not `httpapi/`)
- [x] `make test-integration` passes

### Step 6: Enforce hex boundaries in tests

Reference: [design/testing-at-hex-boundaries.md](../testing-at-hex-boundaries.md)

- [x] Delete `Server.CreateUser` — inline `s.cfg.Users.CreateUser` in the handler
- [x] Rewrite `users_test.go` to test through HTTP (POST, check status and body)
- [x] Rewrite `chirp_test.go` to use raw JSON instead of unexported `parameters` struct
- [x] Rewrite `metrics_test.go` to test through HTTP endpoints instead of `s.cfg`
- [x] All httpapi test files use `package httpapi_test`
- [x] `Server`'s exported API is only `NewServer` and `ServeHTTP`
- [x] `make test` passes


## Doc 10 — Scaling the Store Layer

### Structural refactor (before or with second entity)

- [ ] `Repositories` struct in `internal/config/` with named fields per entity
- [ ] `Config` / `NewConfig` updated to accept `Repositories`
- [ ] `main.go` wiring updated
- [ ] `newTestServer` in httpapi tests updated

### Per-entity checklist (repeat for each new entity via feature-development-loop.md)

Fast loop (no Docker):

- [ ] Governance checklist reviewed (endpoint shape, error cases, pagination, cascades)
- [ ] Domain type + interface in `internal/domain/`
- [ ] Memory repository implementation with compile-time check
- [ ] API handler + HTTP tests (red → green)
- [ ] Refactor: interface minimal, response shape deliberate, service method extracted if needed

Slow loop (Docker, real Postgres):

- [ ] Migration created (`make sql-create`)
- [ ] Queries in `internal/postgres/schema/queries/<entity>.sql`, then `make sql-generate`
- [ ] Postgres repository with `toUser`-style mapper and compile-time check
- [ ] Wired into `Repositories` and `main.go`
- [ ] Integration tests for relational behavior (cascades, FK violations, compound uniqueness)


---

## Doc 11 — OpenAPI Codegen

- [ ] `api/openapi.yaml` describes all existing endpoints
- [ ] `api/codegen.yaml` configures oapi-codegen
- [ ] `make api-generate` produces `internal/httpapi/openapi.gen.go`
- [ ] `Server` implements the generated `ServerInterface`
- [ ] Hand-written request/response structs removed
- [ ] Hand-written routing removed
- [ ] Hand-written validation replaced by spec constraints
- [ ] Tests updated — no longer testing generated behavior
- [ ] `make test` passes
- [ ] README documents the codegen workflow


---

## Milestone: Production Readiness

Docs 12–13 make the server safe for multi-replica, zero-downtime deployment.
Prerequisite: startup must fail fast if the database is unreachable.


## Doc 12 — Schema Version Check

- [ ] Migration checker function with tests
- [ ] Startup gate in `main.go` (exit if migrations pending)
- [ ] `/api/healthz` reports schema status
- [ ] Healthz tests updated
- [ ] README documents the behavior


## Doc 13 — Always-On Readiness

### Step 0: Startup DB ping (fail fast)

- [ ] `postgres.Open` pings the database after `sql.Open`
- [ ] Unreachable database at startup is a fatal error

### Step 1: SIGTERM and shutdown timeout

- [ ] `runUntilInterrupt` catches `syscall.SIGTERM` alongside `os.Interrupt`
- [ ] Shutdown uses a timeout context (not `context.Background()`)
- [ ] Logs which signal was received

### Step 2: Connection pool limits

- [ ] `SetMaxOpenConns` configured
- [ ] `SetMaxIdleConns` configured
- [ ] `SetConnMaxLifetime` configured

### Step 3: Readiness and liveness probes

- [ ] `/livez` returns 200 unconditionally (no dependency checks)
- [ ] Dependency check registry with private/shared classification
- [ ] `/readyz` returns 503 when a private check fails
- [ ] `/readyz` reports shared check failures in body without failing the probe
- [ ] Schema status registered as a private check
- [ ] Database ping registered as a shared check
- [ ] Tests for readiness (healthy, private failure, shared failure)
- [ ] Tests for liveness

### Step 4: Wire and verify

- [ ] Existing `/api/healthz` still works (backward compatibility)
- [ ] README documents the new health endpoints


---

## Doc 14 — Observability

- [ ] Extract metrics interface in `application/`
- [ ] Move `ServerMetrics` implementation to an adapter
- [ ] Assembly wires metrics adapter (production, no-op for tests)
- [ ] Define logging interface in `application/`
- [ ] Implement structured logging adapter (slog-based)
- [ ] Handlers and services log through interface, not `log`/`fmt`
- [ ] Health/readiness endpoints verify DB connectivity
- [ ] `make test` passes
