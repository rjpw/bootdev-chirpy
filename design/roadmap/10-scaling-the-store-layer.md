# Scaling the Repository Layer

Reference card for the mechanical patterns used when adding entities. For the development workflow and testing philosophy, see [design/feature-development-loop.md](../../design/feature-development-loop.md).


## Touch points for a new entity

| Layer | Files | Work |
|-------|-------|------|
| Domain type + interface | `internal/domain/<entity>.go` | Add struct + interface |
| Memory impl | `internal/memory/<entity>.go` | Implement interface, compile-time check |
| HTTP handler + tests | `internal/httpapi/` | Handler, route, HTTP tests |
| Migration | `internal/postgres/schema/migrations/` | `make sql-create`, write DDL |
| SQL queries | `internal/postgres/schema/queries/<entity>.sql` | Write queries, `make sql-generate` |
| Postgres impl | `internal/postgres/<entity>.go` | Implement interface, `toDomain*` mapper, compile-time check |
| Config + wiring | `internal/config/config.go`, `cmd/chirpy/main.go`, `server_test.go` | Add to `Repositories`, wire |
| Integration tests | `internal/postgres/<entity>_test.go` | Only for non-trivial translation |

Order matters — work top to bottom. See [design/feature-development-loop.md](../../design/feature-development-loop.md) for why.


## One interface per entity

```go
type UserRepository interface { ... }
type ChirpRepository interface { ... }
```

Not a combined `Repository` interface. Each handler declares only the interfaces it needs.


## One concrete type, many interfaces

```go
// postgres/users.go
var _ domain.UserRepository = (*Repository)(nil)

// postgres/chirps.go
var _ domain.ChirpRepository = (*Repository)(nil)
```

Same for `memory.Repository`. One struct, multiple interface satisfactions, methods split across files by entity.


## Repositories struct

```go
type Repositories struct {
    Users  domain.UserRepository
    Chirps domain.ChirpRepository
    Tokens domain.TokenRepository
}
```

Named fields in wiring. Tests populate only what they need — unused repositories stay nil.


## File-per-entity convention

```
internal/domain/user.go              ← User type + UserRepository interface
internal/domain/chirp.go             ← Chirp type + ChirpRepository interface
internal/domain/errors.go            ← shared sentinel errors

internal/postgres/users.go           ← UserRepository methods, toUser mapper, compile-time check
internal/postgres/chirps.go
internal/memory/users.go
internal/memory/chirps.go
```

Every entity file has the same structure. No surprises.


## The `toDomain*` mapper

```go
func toChirp(row database.Chirp) *domain.Chirp {
    return &domain.Chirp{
        ID: row.ID, CreatedAt: row.CreatedAt, Body: row.Body, AuthorID: row.UserID,
    }
}
```

The only place that knows both `database.Chirp` and `domain.Chirp`. When sqlc regenerates, fix the mapper and nothing else changes.


## When to split `domain/`

The domain package already uses per-entity files (`user.go`, `chirp.go`, `errors.go`). This scales naturally. No split needed — just add a new file per entity.


## What this doesn't solve

- **Cross-entity transactions**: Repository interfaces don't span entities. Use a `UnitOfWork` pattern or a purpose-built method when needed.
- **Query complexity**: Use parameter structs when a method exceeds 3–4 arguments.
- **Migration coordination**: Two entities in one PR need two migrations. See [devops/02-migration-discipline.md](devops/02-migration-discipline.md).
