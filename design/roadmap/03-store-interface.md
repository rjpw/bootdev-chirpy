# 03 — Store Interface and Domain Types

In this step you'll define the `store.UserStore` interface and the domain types that your application will use. This is the boundary between your application logic and the database — everything above this line speaks in domain terms, everything below it speaks in SQL.

By the end of this doc, you'll have:
- `internal/store/store.go` — the `UserStore` interface and `User` type
- `internal/store/errors.go` — sentinel errors for common failure cases

No implementation yet. Just the contract.


## Why not just use sqlc's types directly?

You explored this in `design/design-faq.md`, but it's worth restating in the context of testing:

sqlc generates types shaped by your SQL, not by your application's needs. The generated `CreateUserParams` struct has fields for `ID`, `CreatedAt`, and `UpdatedAt` — but should a handler really be responsible for generating UUIDs and timestamps? That's a persistence concern.

Your store interface can have a simpler signature:

```go
CreateUser(ctx context.Context, email string) (User, error)
```

The store implementation generates the UUID and timestamps internally. The handler just passes the email. This is a better contract — it expresses what the application needs, not what the database requires.

It also means your API handler tests don't need to construct `database.CreateUserParams` with fake UUIDs and timestamps. They work with the simpler store interface, which can be faked trivially.


## Step 1: Define the User domain type

Create `internal/store/store.go`. Start with the domain type:

```go
package store

import (
    "context"
    "time"

    "github.com/google/uuid"
)

type User struct {
    ID        uuid.UUID
    CreatedAt time.Time
    UpdatedAt time.Time
    Email     string
}
```

This looks identical to `database.User` right now. That's fine. The point isn't that they're different today — it's that they *can* diverge without breaking each other. When you add password hashing, the database model might have a `HashedPassword` field that the domain type exposes differently (or not at all).

> **Go idiom: separate domain types from persistence types.** Even when they look the same, keeping them separate means your application code never imports the `database` package. Changes to your SQL schema (and therefore to sqlc's generated types) only affect the store implementation, not the handlers.


## Step 2: Define the UserStore interface

Add the interface to the same file:

```go
type UserStore interface {
    CreateUser(ctx context.Context, email string) (User, error)
}
```

Start small. You only have one query (`CreateUser`) right now. Add methods as you add queries — don't design the interface ahead of the implementation.

> **Go idiom: small interfaces.** Go favors small, focused interfaces. The standard library's `io.Reader` has one method. `io.Writer` has one method. Start with what you need now and grow the interface as requirements emerge. A one-method interface is perfectly normal in Go.

> **Go idiom: accept interfaces, return structs.** Your handlers will accept `UserStore` (an interface). Your `PostgresStore` constructor will return `*PostgresStore` (a concrete struct). This is the standard Go pattern — it keeps the interface at the consumer side, where it belongs.

Notice the signature: `CreateUser(ctx context.Context, email string) (User, error)`. Compare this to sqlc's generated method:

```go
func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
```

Your interface is simpler. The store implementation will bridge the gap — it takes an email, generates the UUID and timestamps, constructs `CreateUserParams`, calls the sqlc method, and maps the result back to `store.User`.


## Step 3: Define sentinel errors

Create `internal/store/errors.go`:

```go
package store

import "errors"

var (
    ErrNotFound = errors.New("not found")
    ErrConflict = errors.New("conflict")
)
```

These are the errors your application logic will check for. Handlers will map them to HTTP status codes:

- `ErrNotFound` → 404
- `ErrConflict` → 409

The store implementation is responsible for translating database-specific errors (like Postgres error code `23505` for unique violations) into these sentinel errors. Your handlers never see a `*pq.Error`.

> **Go idiom: sentinel errors.** A sentinel error is a package-level `var` created with `errors.New`. Callers check for it with `errors.Is(err, store.ErrConflict)`. This works through wrapped errors — if your store wraps the original database error with `fmt.Errorf("create user: %w", store.ErrConflict)`, `errors.Is` will still find it.

> **Go idiom: errors.Is vs errors.As.** Use `errors.Is` when you want to check if an error matches a known value (sentinel). Use `errors.As` when you want to extract a specific error type (like `*pq.Error`). Your store implementation will use `errors.As` to inspect database errors, then return sentinel errors that handlers check with `errors.Is`.

Keep the error set small. You can always add more later (`ErrForbidden`, `ErrInvalidInput`, etc.) as your application needs them. Two is enough to start.


## Step 4: Review the package structure

You should now have:

```
internal/store/
├── store.go    ← User type + UserStore interface
└── errors.go   ← ErrNotFound, ErrConflict
```

That's it. No implementation. The `memory/` directory that already exists is empty, and you haven't created `postgres/` yet. That's doc 04.


## Verify

- [ ] `go build ./internal/store/...` compiles without errors
- [ ] The `UserStore` interface has a `CreateUser` method with a clean signature
- [ ] The `User` type has the fields you need
- [ ] Sentinel errors are defined and exported
- [ ] No imports of `internal/database` in this package — the store package knows nothing about sqlc


## Explore

1. **Compare interfaces.** Open `internal/database/db.go` and look at sqlc's generated `DBTX` interface. Now look at your `UserStore`. How are they different in purpose? `DBTX` describes what the database connection can do. `UserStore` describes what your application needs. Which one would you want to mock in a handler test?

2. **Interface satisfaction.** Go interfaces are satisfied implicitly. Right now, nothing satisfies `UserStore`. That's fine — the compiler won't complain until something tries to use it. Try assigning a `*database.Queries` to a `UserStore` variable — what error do you get? This shows you exactly what methods are missing.

3. **Error wrapping.** Try this in a scratch file or test:
   ```go
   wrapped := fmt.Errorf("create user: %w", store.ErrConflict)
   fmt.Println(errors.Is(wrapped, store.ErrConflict)) // true
   ```
   The `%w` verb wraps the error so `errors.Is` can find it through the chain. This is how your store will return errors — wrapped with context but still matchable.

4. **Read the design docs again.** Now that you've defined the interface, re-read `design/design-faq.md` and `design/design-help-claude.md`. The mental model described there should feel concrete now. You've built the middle layer of the diagram.


## Reference

- [Effective Go: Interfaces](https://go.dev/doc/effective_go#interfaces)
- [Go blog: Error handling](https://go.dev/blog/go1.13-errors)
- [errors.Is](https://pkg.go.dev/errors#Is)
- [errors.As](https://pkg.go.dev/errors#As)
