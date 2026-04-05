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

- [x] `internal/store/postgres/postgres.go` — `PostgresStore` implementing `UserStore`
- [x] Compile-time interface check: `var _ store.UserStore = (*PostgresStore)(nil)`
- [x] `mapError` translates `sql.ErrNoRows` → `ErrNotFound`, pq `23505` → `ErrConflict`
- [x] Timestamps use `UTC().Truncate(time.Microsecond)`
- [ ] Snapshot/restore integration tests
- [ ] Tests run via testcontainers (not local Postgres)


## Doc 07 — Developer Workflow

- [ ] Integration tests gated behind `//go:build integration`
- [ ] `make test` runs only fast tests
- [ ] `make test-integration` runs everything
- [ ] `make sql-create` generates timestamped migrations
- [ ] `make sql-fix` renumbers for release
- [ ] README documents the testing approach


## Doc 08 — Migrate Subcommand

- [ ] `./chirpy migrate up` applies pending migrations
- [ ] `./chirpy migrate status` shows migration state
- [ ] SQL files embedded in binary via `embed.FS`
- [ ] `./chirpy` (no args) still starts the server
- [ ] README documents the subcommand
