# Adapting Existing Course Work to the Roadmap

This was written after I had already completed Boot.dev lesson [CH5-L3: SQLC](https://www.boot.dev/lessons/e5bddf3d-d96b-487e-97e6-7a5aa06b1ee1), which had already added the goose migration and sqlc steps. This doc explains how to layer the roadmap's interface-driven architecture onto your existing code without breaking what works.


## Current state

- `sql/schema/001_users.sql` — sequential migration (course style)
- `sql/queries/users.sql` — sqlc query for `CreateUser`
- `internal/database/` — sqlc-generated code: `Queries`, `User`, `CreateUserParams`
- `config.Config` holds `*database.Queries` directly
- `main.go` opens Postgres, creates `Queries`, passes into config
- Handlers access the database through `cfg.Db`


## Target state

Handlers talk to a `store.UserStore` interface. The sqlc-generated code stays, but it's wrapped by a postgres adapter that implements the interface. A memory store proves the interface works without a database.


## Steps

### 1. Create the store interface (roadmap doc 01)

Create `internal/store/store.go` with:
- A `User` domain type (will look nearly identical to `database.User` — that's fine, the store layer owns its own types)
- A `UserStore` interface with `CreateUser(ctx, email) (User, error)`
- Sentinel errors `ErrNotFound` and `ErrConflict` in the same file or a separate `errors.go`

The interface signature differs from sqlc's generated `CreateUser` — it takes just `email`, not a params struct with ID and timestamps. The implementation decides how to generate those. That's the abstraction.

### 2. Build the memory store (roadmap doc 02)

Create `internal/store/memory/memory.go` implementing `UserStore` with a map and mutex. This proves the interface compiles and is sufficient. Write the two tests (`TestCreateUser`, `TestCreateUserDuplicateEmail`) against the interface type.

### 3. Add UserStore to config (roadmap doc 03)

Add a `Users store.UserStore` field to `Config` alongside the existing `Db *database.Queries`. Wire the memory store in `main.go`. You don't need to remove `Db` — both can coexist. Handlers migrate from `cfg.Db` to `cfg.Users` one at a time.

```go
type Config struct {
    Metrics *metrics.ServerMetrics
    Db      *database.Queries   // keep — course handlers still use it
    Users   store.UserStore     // new
}
```

### 4. Build the postgres adapter (roadmap doc 06)

Create `internal/store/postgres/postgres.go`. This is a thin wrapper around your existing `*database.Queries`:
- `CreateUser` generates the UUID and timestamps, calls `queries.CreateUser` with a `CreateUserParams`, translates `database.User` to `store.User`, and maps `*pq.Error` to sentinel errors
- The compile-time check: `var _ store.UserStore = (*PostgresStore)(nil)`

### 5. Swap in main.go

Replace the memory store with the postgres adapter:

```go
// before
userStore := memory.NewMemoryStore()

// after
userStore := postgres.NewPostgresStore(dbQueries)
```

Everything else stays the same. The interface held.

### 6. Migrate handlers gradually

As the course adds new endpoints, write them against `cfg.Users` instead of `cfg.Db`. Existing handlers that use `cfg.Db` directly can be migrated when convenient. There's no rush — both paths work simultaneously.


## Migration file convention

Your `001_users.sql` is already applied and works. Don't rename it. For all future migrations, use `goose create`:

```bash
goose -dir sql/schema create add_chirps sql
```

This generates a timestamped file. Goose handles mixed sequential + timestamp ordering. See `design/roadmap/devops/02-migration-discipline.md` for the full convention.


## What stays, what changes

| Component | Status |
|-----------|--------|
| `sql/schema/001_users.sql` | Stays as-is |
| `sql/queries/users.sql` | Stays as-is |
| `internal/database/` (sqlc generated) | Stays — postgres adapter wraps it |
| `config.Config.Db` | Stays until all handlers migrate |
| `config.Config.Users` | New — added in step 3 |
| `internal/store/store.go` | New — interface and domain types |
| `internal/store/memory/` | New — proves the interface, disposable |
| `internal/store/postgres/` | New — thin adapter over sqlc |
