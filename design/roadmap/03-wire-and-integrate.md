# 03 — Wire the Store into the API Server

In this step you'll replace the direct `*database.Queries` dependency in your API server with the `store.UserStore` interface. This connects the architecture you've been building to the application code you already have.

By the end of this doc:
- `Config` holds a `store.UserStore` instead of `*database.Queries`
- `main.go` creates a `PostgresStore` and injects it
- All existing tests still pass
- Your API handler tests remain fast (no database)


## The composition root

Your `cmd/chirpy/main.go` is the composition root — the one place in the application that knows which concrete implementations are in use. Right now it creates a `*database.Queries` and puts it in `Config`. After this change, it will create a `PostgresStore` (which internally uses `*database.Queries`) and put that in `Config` instead.

Nothing else in the application changes. Handlers call methods on the interface. They don't know or care whether the implementation is backed by Postgres, an in-memory map, or a test fake.

> **Go idiom: composition root.** The composition root is where you wire together all your dependencies. In Go, this is almost always `main()`. It's the only function that imports concrete implementations. Everything else depends on interfaces. This keeps your dependency graph clean and makes testing easy.


## Step 1: Update Config

Open `internal/config/config.go`. Currently:

```go
type Config struct {
    Metrics *metrics.ServerMetrics
    Db      *database.Queries
}
```

Change `Db` to use the store interface:

```go
type Config struct {
    Metrics *metrics.ServerMetrics
    Users   store.UserStore
}
```

A few things to decide:

- **Field name:** `Db` was generic. `Users` is specific to the `UserStore` interface. As you add more store interfaces (e.g., `ChirpStore`), you'll add more fields. This keeps each dependency explicit.
- **Import path:** You'll need to import `github.com/rjpw/bootdev-chirpy/internal/store`. Note that `config` now depends on `store` (the interface package), not on `database` or `store/postgres`. This is the right dependency direction.

> **Go idiom: depend on interfaces, not implementations.** The `config` package imports `store` (which defines interfaces), not `store/postgres` (which implements them). Only `main.go` imports the implementation. This is the dependency inversion principle in action.

After this change, the compiler will tell you everywhere that referenced `cfg.Db`. Fix each one. If no handlers currently use the database, there may be nothing to fix beyond `main.go`.


## Step 2: Update main.go

Open `cmd/chirpy/main.go`. Currently:

```go
dbQueries := database.New(db)
cfg := &config.Config{Metrics: &metrics.ServerMetrics{}, Db: dbQueries}
```

Change it to create a `PostgresStore`:

```go
dbQueries := database.New(db)
userStore := postgres.NewPostgresStore(dbQueries)
cfg := &config.Config{Metrics: &metrics.ServerMetrics{}, Users: userStore}
```

You'll need to import `github.com/rjpw/bootdev-chirpy/internal/store/postgres`. This is the only place in the application that imports the concrete implementation.

> **Hint:** If the import alias collides with the testcontainers postgres package (it won't here, since `main.go` doesn't use testcontainers), use an alias like `storepostgres`.


## Step 3: Update the test helper

Open `internal/api/server_test.go` and look at `newTestServer()`:

```go
func newTestServer() *Server {
    cfg := &config.Config{Metrics: &metrics.ServerMetrics{}}
    return NewServer(cfg, "./testdata")
}
```

The `Db` field was implicitly nil. Now you need to set `Users` — but to what?

You have three options:

**Option A: nil.** If no handlers currently use `cfg.Users`, nil works. But it's a landmine — the first handler that calls a store method will panic with a nil pointer dereference.

**Option B: A minimal fake.** Create a simple struct that satisfies `UserStore` and returns canned data or errors. This is the long-term right answer.

**Option C: nil with a plan.** Use nil for now, but add a comment noting that this needs a fake when handlers start using the store.

For now, Option A or C is fine if no handlers use the store yet. When you add a handler that calls `CreateUser`, you'll need a fake. Here's what a minimal fake looks like for future reference:

```go
// internal/store/fake/fake.go
package fake

import (
    "context"

    "github.com/rjpw/bootdev-chirpy/internal/store"
)

type FakeUserStore struct {
    CreateUserFn func(ctx context.Context, email string) (store.User, error)
}

func (f *FakeUserStore) CreateUser(ctx context.Context, email string) (store.User, error) {
    return f.CreateUserFn(ctx, email)
}
```

This pattern lets each test configure the fake's behavior by setting the function field. It's more flexible than a mock library and more idiomatic in Go.

> **Go idiom: function-field fakes.** Instead of using a mock framework, Go developers often write fakes with function fields. Each test sets the function to return whatever that test needs. It's explicit, type-safe, and easy to read. No magic, no code generation.


## Step 4: Verify everything compiles and passes

Run the full test suite:

```bash
go test -race ./...
```

Every test should pass. The API handler tests don't touch the database. The sqlc query tests and store tests use testcontainers.

Also verify the server starts:

```bash
# Make sure your local Postgres is running (docker compose up -d)
# Make sure migrations are applied (make sql-migrate)
go run ./cmd/chirpy/
```

The server should start and serve requests as before.


## Step 5: Trace the dependency graph

After wiring, your dependency graph should look like this:

```
cmd/chirpy/main.go
    imports: internal/config
    imports: internal/store/postgres  ← only place that imports the implementation
    imports: internal/database        ← to create Queries for PostgresStore

internal/api/
    imports: internal/config
    imports: internal/store           ← interface only, not postgres

internal/config/
    imports: internal/store           ← interface only

internal/store/postgres/
    imports: internal/store           ← for domain types and errors
    imports: internal/database        ← for sqlc types

internal/store/
    imports: nothing from this project ← pure interface + types
```

The key property: no circular dependencies, and the interface package (`internal/store`) is a leaf — it depends on nothing else in your project. Everything points inward toward the interface.

> **Go idiom: dependency direction.** In Go, import cycles are a compile error, not a warning. This forces you to think about dependency direction. The rule of thumb: depend inward (toward interfaces and domain types), not outward (toward implementations and infrastructure).


## Verify

- [ ] `go build ./...` compiles without errors
- [ ] `go test -race ./...` — all tests pass (API, query, store)
- [ ] `Config` holds `store.UserStore`, not `*database.Queries`
- [ ] Only `main.go` imports `internal/store/postgres`
- [ ] The server starts and serves requests
- [ ] No import cycles


## Explore

1. **Try importing the wrong way.** In `internal/api/`, try importing `internal/store/postgres` instead of `internal/store`. Does it compile? It will — but it defeats the purpose. Your handlers would depend on the concrete implementation. Revert it and think about why the interface import is better.

2. **Add a handler that uses the store.** Write a `POST /api/users` handler that calls `cfg.Users.CreateUser(ctx, email)`. Write an API-level test for it using a fake store (no database). Then test it manually against the running server with `curl`. This is the payoff of the whole architecture — fast tests at the top, real database tests at the bottom.

3. **What if the store is nil?** In `newTestServer()`, leave `Users` as nil and write a handler that calls it. What error do you get? Is it a good error? This is why Option B (a fake) is better than Option A (nil) — it fails with a meaningful error instead of a panic.

4. **Inspect the binary.** Run `go build -o /dev/null ./cmd/chirpy/ 2>&1` and check for any warnings. Then run `go vet ./...`. Clean code should produce no output from either.


## Reference

- [Go blog: The Laws of Reflection (interfaces section)](https://go.dev/blog/laws-of-reflection)
- [Effective Go: Embedding](https://go.dev/doc/effective_go#embedding)
- [Go FAQ: Why doesn't Go have circular imports?](https://go.dev/doc/faq#no_goroutine_id)
