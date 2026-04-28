# Scaling the Store Layer

Reference card for the mechanical patterns used when adding entities. For the development workflow and testing philosophy, see [design/feature-development-loop.md](../../design/feature-development-loop.md).


## Touch points for a new entity

| Layer | Files | Work |
|-------|-------|------|
| Domain type + interface | `internal/store/store.go` | Add struct + interface |
| Memory impl | `internal/store/memory/<entity>.go` | Implement interface, compile-time check |
| API handler + tests | `internal/api/` | Handler, route, HTTP tests |
| Migration | `internal/schema/migrations/` | `make sql-create`, write DDL |
| SQL queries | `internal/schema/queries/<entity>.sql` | Write queries, `make sql-generate` |
| Postgres impl | `internal/store/postgres/<entity>.go` | Implement interface, `toStore*` mapper, compile-time check |
| Config + wiring | `internal/config/config.go`, `cmd/chirpy/main.go`, `server_test.go` | Add to `Stores`, wire |
| Integration tests | `internal/store/postgres/<entity>_test.go` | Only for non-trivial translation |

Order matters — work top to bottom. See [design/feature-development-loop.md](../../design/feature-development-loop.md) for why.


## One interface per entity

```go
type UserStore interface { ... }
type ChirpStore interface { ... }
```

Not a combined `Store` interface. Each handler declares only the interfaces it needs.


## One concrete type, many interfaces

```go
// postgres/users.go
var _ store.UserStore = (*Store)(nil)

// postgres/chirps.go
var _ store.ChirpStore = (*Store)(nil)
```

Same for `memory.Store`. One struct, multiple interface satisfactions, methods split across files by entity.


## Stores struct

```go
type Stores struct {
    Users  store.UserStore
    Chirps store.ChirpStore
    Tokens store.TokenStore
}
```

Named fields in wiring. Tests populate only what they need — unused stores stay nil.


## File-per-entity convention

```
internal/store/store.go              ← all domain types and interfaces
internal/store/errors.go             ← shared sentinel errors

internal/store/postgres/users.go     ← UserStore methods, toStoreUser, compile-time check
internal/store/postgres/chirps.go
internal/store/memory/users.go
internal/store/memory/chirps.go
```

Every entity file has the same structure. No surprises.


## The `toStore*` mapper

```go
func toStoreChirp(row database.Chirp) *store.Chirp {
    return &store.Chirp{
        ID: row.ID, CreatedAt: row.CreatedAt, Body: row.Body, AuthorID: row.UserID,
    }
}
```

The only place that knows both `database.Chirp` and `store.Chirp`. When sqlc regenerates, fix the mapper and nothing else changes.


## When to split `store.go`

With four or five entities, split into per-entity files:

```
internal/store/user.go       ← User type + UserStore interface
internal/store/chirp.go      ← Chirp type + ChirpStore interface
internal/store/errors.go     ← shared errors (unchanged)
```

The trigger is readability, not a rule.


## What this doesn't solve

- **Cross-entity transactions**: Store interfaces don't span entities. Use a `UnitOfWork` pattern or a purpose-built method when needed.
- **Query complexity**: Use parameter structs when a method exceeds 3–4 arguments.
- **Migration coordination**: Two entities in one PR need two migrations. See [devops/02-migration-discipline.md](devops/02-migration-discipline.md).
