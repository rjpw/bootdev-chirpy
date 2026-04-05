# Scaling the Store Layer

This doc addresses what happens when the project grows beyond a single `UserStore`. The Boot.dev course will add chirps, tokens, refresh tokens, and potentially more. Without a plan, each new entity means copy-pasting boilerplate across four layers (SQL, sqlc, postgres store, memory store) and watching the `Config` struct accumulate fields.

The goal: adding a new entity should be mechanical, not creative. Low cognitive load, minimal busy work, no architectural decisions each time.


## Where the boilerplate lives today

Adding "chirps" to the current architecture means touching:

| Layer | Files | Work |
|-------|-------|------|
| Migration | `internal/schema/migrations/TIMESTAMP_add_chirps.sql` | Write DDL |
| SQL queries | `internal/schema/queries/chirps.sql` | Write queries |
| sqlc codegen | `internal/database/` | Run `make sql-generate` (free) |
| Domain type | `internal/store/store.go` | Add `Chirp` struct + `ChirpStore` interface |
| Sentinel errors | `internal/store/errors.go` | Usually nothing — `ErrNotFound`/`ErrConflict` are reusable |
| Postgres impl | `internal/store/postgres/chirps.go` | Implement `ChirpStore`, write `toStoreChirp` |
| Memory impl | `internal/store/memory/chirps.go` | Implement `ChirpStore` with map + mutex |
| Postgres tests | `internal/store/postgres/chirps_test.go` | Test against real DB |
| Memory tests | `internal/store/memory/chirps_test.go` | Test against in-memory |
| Query tests | `internal/database/chirps_queries_test.go` | Test raw sqlc queries |
| Config | `internal/config/config.go` | Add `Chirps store.ChirpStore` field |
| Wiring | `cmd/chirpy/main.go` | Wire the new store into config |
| API test helper | `internal/api/server_test.go` | Add memory chirp store to `newTestServer` |

That's 13 touch points. Most are small, but the list itself is the cognitive load problem. A developer adding their third entity shouldn't need to remember all 13 steps.


## Principle: one interface per entity

Don't combine entities into a single `Store` interface:

```go
// Don't do this
type Store interface {
    CreateUser(ctx context.Context, email string) (*User, error)
    GetUserByEmail(ctx context.Context, email string) (*User, error)
    CreateChirp(ctx context.Context, body string, userID uuid.UUID) (*Chirp, error)
    GetChirp(ctx context.Context, id uuid.UUID) (*Chirp, error)
    // ... grows forever
}
```

This forces every implementation (memory, postgres, future test doubles) to implement every method, even if a handler only needs chirps. It also makes the interface impossible to read at a glance.

Instead, keep interfaces small and focused:

```go
// internal/store/store.go

type UserStore interface {
    CreateUser(ctx context.Context, email string) (*User, error)
    GetUserByEmail(ctx context.Context, email string) (*User, error)
    // ...
}

type ChirpStore interface {
    CreateChirp(ctx context.Context, body string, userID uuid.UUID) (*Chirp, error)
    GetChirp(ctx context.Context, id uuid.UUID) (*Chirp, error)
    // ...
}
```

Each interface is a contract for one entity. Handlers declare which interfaces they need. A handler that only reads chirps only needs `ChirpStore`.


## Principle: one concrete type satisfies many interfaces

Both `memory.Store` and `postgres.Store` can implement all the entity interfaces. The compile-time checks make this explicit:

```go
// internal/store/postgres/users.go
var _ store.UserStore = (*Store)(nil)

// internal/store/postgres/chirps.go
var _ store.ChirpStore = (*Store)(nil)
```

This means you don't need separate `PostgresUserStore` and `PostgresChirpStore` types. One `Store` struct, one `*database.Queries` field, multiple interface satisfactions. The methods are split across files by entity — `users.go`, `chirps.go`, `tokens.go` — but they all hang off the same receiver.

Same for memory:

```go
// internal/store/memory/users.go
var _ store.UserStore = (*Store)(nil)

// internal/store/memory/chirps.go
var _ store.ChirpStore = (*Store)(nil)
```

One `Store` struct with multiple maps, one mutex, multiple interface satisfactions.


## Principle: Config grows, but predictably

As entities accumulate, a positional constructor like `NewConfig(platform, metrics, users, chirps, tokens)` becomes hard to read and easy to get wrong. Group the store interfaces into a `Stores` struct:

```go
// internal/config/config.go

type Stores struct {
    Users  store.UserStore
    Chirps store.ChirpStore
    Tokens store.TokenStore
}

type Config struct {
    Platform string
    Metrics  *metrics.ServerMetrics
    Stores   Stores
}
```

The wiring in `main.go` uses named fields, so each assignment is self-documenting:

```go
s := postgres.NewPostgresStore(database.New(db))
cfg := config.NewConfig(
    env.Platform,
    &metrics.ServerMetrics{},
    config.Stores{Users: s, Chirps: s, Tokens: s},
)
```

In tests, populate only what the test needs — unused stores stay nil:

```go
m := memory.NewMemoryStore()
cfg := &config.Config{
    Platform: "dev",
    Metrics:  &metrics.ServerMetrics{},
    Stores:   config.Stores{Users: m},
}
```

This is more honest than passing `nil, nil` in a positional constructor. A handler that only touches chirps gets a config where only `Stores.Chirps` is set, making the test's scope visible at a glance.


## Reducing the touch points

The 13-step list above is unavoidable in terms of what needs to exist. But you can reduce the cognitive load of executing it.

### File-per-entity convention

Every entity follows the same file layout:

```
internal/store/store.go          ← all domain types and interfaces
internal/store/errors.go         ← shared sentinel errors

internal/store/postgres/users.go
internal/store/postgres/chirps.go
internal/store/postgres/tokens.go

internal/store/memory/users.go
internal/store/memory/chirps.go
internal/store/memory/tokens.go
```

When you open `postgres/chirps.go`, you know exactly what's in it: the `ChirpStore` methods on `*Store`, a `toStoreChirp` mapper, and a compile-time check. No surprises.

### The `toStore*` mapper pattern

Every postgres entity file has one mapper:

```go
func toStoreChirp(row database.Chirp) *store.Chirp {
    return &store.Chirp{
        ID:        row.ID,
        CreatedAt: row.CreatedAt,
        UpdatedAt: row.UpdatedAt,
        Body:      row.Body,
        UserID:    row.UserID,
    }
}
```

This is the only place that knows about both `database.Chirp` and `store.Chirp`. When sqlc regenerates, you fix the mapper and nothing else changes.

### Checklist for adding an entity

Keep this somewhere you'll see it (this doc, a CONTRIBUTING.md, or a Makefile comment):

1. `make sql-create` → write the migration DDL
2. Write queries in `internal/schema/queries/<entity>.sql`
3. `make sql-generate`
4. Add domain type and interface to `internal/store/store.go`
5. Implement in `internal/store/postgres/<entity>.go` (with compile-time check)
6. Implement in `internal/store/memory/<entity>.go` (with compile-time check)
7. Add field to `Config`, update `NewConfig`
8. Wire in `main.go` and `newTestServer`
9. Write tests

Steps 1–3 are SQL. Steps 4–6 are Go store layer. Steps 7–8 are wiring. Step 9 is verification. The order is always the same.


## When to split `store.go`

With one or two entities, everything fits in `store.go`. Once you have four or five, consider splitting:

```
internal/store/user.go       ← User type + UserStore interface
internal/store/chirp.go      ← Chirp type + ChirpStore interface
internal/store/token.go      ← Token type + TokenStore interface
internal/store/errors.go     ← shared errors (unchanged)
```

The trigger is readability, not a rule. If you open `store.go` and can't find what you're looking for in a few seconds, split it.


## What this doesn't solve

- **Cross-entity transactions**: If creating a chirp and updating a user's chirp count need to be atomic, the store interfaces as designed don't help. You'd need a method that spans both, or a `UnitOfWork` pattern. Cross that bridge when you reach it.
- **Query complexity**: As queries get more complex (joins, pagination, filters), the store interface methods get more parameters. Consider parameter structs (like sqlc's `CreateUserParams`) when a method exceeds 3-4 arguments.
- **Migration coordination**: Two entities added in the same PR need two migrations. The migration discipline doc covers ordering.

These are real problems but they're future problems. The entity-per-file convention and small interfaces keep the day-to-day work mechanical.
