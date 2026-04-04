# 04 — PostgresStore Implementation and Snapshot/Restore Tests

In doc 02 you built a memory store that satisfies `UserStore`. Now you'll replace it with a real Postgres-backed implementation. The interface doesn't change. The tests you wrote against the contract should pass against this new backend without modification. That's the payoff.

In this step you'll build the concrete `PostgresStore` that satisfies the `UserStore` interface, backed by sqlc-generated queries. Then you'll test it using testcontainers' snapshot/restore feature, which gives you committed-data isolation between tests.

By the end of this doc, you'll have:
- `internal/store/postgres/postgres.go` — the `PostgresStore` implementation
- `internal/store/postgres/store_test.go` — integration tests with snapshot/restore


## Why a different isolation strategy here

In doc 05, you tested sqlc queries with transaction rollback. That was appropriate because you were testing individual SQL statements — does this query work, does it return the right types, does it handle constraints?

At the store layer, you're testing higher-level behavior:
- Does `CreateUser` generate a valid UUID and timestamps?
- Does a duplicate email correctly map to `store.ErrConflict`?
- When you add multi-step operations later (create user + assign role in a transaction), does the whole thing commit or roll back correctly?

Some of these require committed data. A transaction-rollback test can't verify that data is actually visible to other connections after a commit. Snapshot/restore gives you real commits with a clean slate between tests.

The testcontainers Postgres module has built-in support for this:
1. After migrations, take a snapshot: `ctr.Snapshot(ctx)`
2. Each test commits data for real
3. After each test, restore: `ctr.Restore(ctx)` — this drops and recreates the database from the snapshot

It's slower than transaction rollback (tens of milliseconds vs sub-millisecond) but still fast enough for a focused set of store-level tests.


## Step 1: Extend testdb to support snapshots

Your `internal/testdb` package currently returns a `*sql.DB`. For snapshot/restore, tests also need access to the container itself (to call `Snapshot` and `Restore`).

You have a design choice here. A few options:

**Option A:** Have `Setup` return both the `*sql.DB` and the `*postgres.PostgresContainer`.

**Option B:** Add a separate function like `SetupWithSnapshot(t *testing.T)` that returns both and takes the snapshot automatically.

**Option C:** Return a struct that holds both.

Think about what the caller needs. For sqlc query tests (doc 05), they only need `*sql.DB`. For store tests, they need both. Option A or B keeps things simple. Option C is more extensible.

Here's the shape for Option A:

```go
func Setup(t *testing.T) (*sql.DB, *postgres.PostgresContainer) {
    // ... same as before, but also return the container
}
```

The snapshot itself should be taken once, after migrations, inside `sync.Once`. That way every test restores to the same post-migration state.

```go
once.Do(func() {
    // ... start container, run migrations ...
    setupErr = container.Snapshot(ctx)
})
```

> **Hint:** You'll need to update doc 05's `setupTx` helper if you change the `Setup` signature. The simplest fix: `setupTx` calls `Setup` and ignores the container return value.


## Step 2: Create the PostgresStore

Create `internal/store/postgres/postgres.go`:

```go
package postgres

import (
    "github.com/rjpw/bootdev-chirpy/internal/database"
)

type PostgresStore struct {
    queries *database.Queries
}

func NewPostgresStore(queries *database.Queries) *PostgresStore {
    return &PostgresStore{queries: queries}
}
```

This is composition, not embedding. `PostgresStore` holds a `*database.Queries` as a private field and delegates to it explicitly. This is intentional — you don't want sqlc's methods to leak through to the store's public API.

> **Go idiom: composition over embedding.** Go supports struct embedding, which promotes all methods of the embedded type. That's useful sometimes, but here it would expose every sqlc method on your store. By using a named field instead, you control exactly which methods the store exposes — only those defined by the `UserStore` interface.


## Step 3: Implement CreateUser

This is where the store earns its keep. The method signature is simpler than sqlc's — it takes an email and handles the rest:

```go
func (s *PostgresStore) CreateUser(ctx context.Context, email string) (store.User, error) {
    now := time.Now().UTC().Truncate(time.Microsecond)
    dbUser, err := s.queries.CreateUser(ctx, database.CreateUserParams{
        ID:        uuid.New(),
        CreatedAt: now,
        UpdatedAt: now,
        Email:     email,
    })
    if err != nil {
        return store.User{}, mapError(err)
    }
    return toStoreUser(dbUser), nil
}
```

You need two helper functions:

**`toStoreUser`** — maps `database.User` → `store.User`. Right now the fields are identical, so it's a straightforward field-by-field copy. It exists so that when the types diverge, you only change this one function.

```go
func toStoreUser(u database.User) store.User {
    return store.User{
        // ... map fields
    }
}
```

**`mapError`** — translates database errors to store sentinel errors. This is where you use the `*pq.Error` inspection from doc 05:

```go
func mapError(err error) error {
    var pgErr *pq.Error
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "23505": // unique_violation
            return store.ErrConflict
        }
    }
    return err
}
```

> **Design decision: wrap or replace?** The `mapError` function above replaces the original error with the sentinel. An alternative is to wrap it: `return fmt.Errorf("%w: %v", store.ErrConflict, err)`. Wrapping preserves the original error for debugging while still allowing `errors.Is(err, store.ErrConflict)` to work. Consider which approach serves your debugging needs better.


## Step 4: Verify interface satisfaction

Add a compile-time check at the top of your file:

```go
var _ store.UserStore = (*PostgresStore)(nil)
```

This line declares a variable of type `store.UserStore` and assigns a nil `*PostgresStore` to it. If `PostgresStore` doesn't satisfy the interface, the compiler will tell you exactly which methods are missing. The variable is blank (`_`) so it's discarded — it exists only for the compile check.

> **Go idiom: compile-time interface check.** This is a common pattern in Go codebases. It catches interface drift early — if you add a method to `UserStore` but forget to implement it on `PostgresStore`, you'll get a compile error immediately, not a runtime error later.


## Step 5: Write store tests with snapshot/restore

Create `internal/store/postgres/store_test.go`:

```go
package postgres_test

import (
    "context"
    "testing"

    "github.com/rjpw/bootdev-chirpy/internal/database"
    "github.com/rjpw/bootdev-chirpy/internal/store"
    storepostgres "github.com/rjpw/bootdev-chirpy/internal/store/postgres"
    "github.com/rjpw/bootdev-chirpy/internal/testdb"
)
```

Notice the import alias `storepostgres` — this avoids a collision with the testcontainers `postgres` package if you need both.

Write a helper that creates a `PostgresStore` and registers snapshot restore for cleanup:

```go
func setupStore(t *testing.T) store.UserStore {
    t.Helper()
    db, ctr := testdb.Setup(t)
    t.Cleanup(func() {
        if err := ctr.Restore(context.Background()); err != nil {
            t.Fatalf("restore snapshot: %v", err)
        }
    })
    queries := database.New(db)
    return storepostgres.NewPostgresStore(queries)
}
```

Key detail: `t.Cleanup` runs after the test finishes, restoring the database to its post-migration snapshot state. Every test starts clean.

> **Go idiom: t.Cleanup ordering.** Cleanup functions run in LIFO order (last registered, first run). If you register multiple cleanups, they unwind like a stack. This matters if you have dependencies between cleanup steps.

Now write the tests:

```go
func TestCreateUser(t *testing.T) {
    s := setupStore(t)
    ctx := context.Background()

    user, err := s.CreateUser(ctx, "alice@example.com")
    if err != nil {
        t.Fatalf("CreateUser: %v", err)
    }

    // The store should have generated a UUID
    if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
        t.Error("expected non-zero UUID")
    }

    // Timestamps should be recent
    // ... check that CreatedAt and UpdatedAt are within the last few seconds

    if user.Email != "alice@example.com" {
        t.Errorf("Email: got %q, want %q", user.Email, "alice@example.com")
    }
}

func TestCreateUserDuplicateEmail(t *testing.T) {
    s := setupStore(t)
    ctx := context.Background()

    _, err := s.CreateUser(ctx, "dupe@example.com")
    if err != nil {
        t.Fatalf("first CreateUser: %v", err)
    }

    _, err = s.CreateUser(ctx, "dupe@example.com")
    if !errors.Is(err, store.ErrConflict) {
        t.Errorf("expected ErrConflict, got: %v", err)
    }
}
```

Notice the difference from doc 05's tests:
- You're testing through the store interface, not sqlc directly
- You don't construct `CreateUserParams` — the store handles that
- You check for `store.ErrConflict`, not `*pq.Error` — the error mapping is part of what you're testing
- Data is committed (no transaction rollback), and snapshot/restore cleans up


## Step 6: Run the tests

```bash
go test -v ./internal/store/postgres/...
```

You should see:
- Container startup (shared with other test packages)
- Snapshot restore between tests
- All tests passing

Run twice to confirm idempotency.


## Verify

- [ ] `go build ./internal/store/postgres/...` compiles
- [ ] The compile-time interface check (`var _ store.UserStore = ...`) is present
- [ ] `go test -v ./internal/store/postgres/...` passes
- [ ] `TestCreateUser` verifies UUID generation, timestamps, and email
- [ ] `TestCreateUserDuplicateEmail` checks for `store.ErrConflict` (not `*pq.Error`)
- [ ] Running tests twice produces identical results (snapshot/restore works)
- [ ] The store package imports `internal/database` but the `store` package (interface) does not


## Explore

1. **Remove the Restore call.** Comment out the `ctr.Restore` in `setupStore`'s cleanup. Run the tests. Do they still pass? Run them again. What happens? This demonstrates why snapshot/restore matters for committed-data tests.

2. **Compare speed.** Time your sqlc query tests (doc 05) vs your store tests. How much slower is snapshot/restore compared to transaction rollback? Is the difference acceptable for the number of tests you have?

3. **Cross-connection visibility.** In a store test, after `CreateUser`, open a second `*sql.DB` connection to the same container and query for the user. Can you find it? (You should be able to — the data is committed.) Try the same thing in a doc 05 transaction-rollback test. Can you find it? (You shouldn't — it's uncommitted.)

4. **Error wrapping.** If you chose to wrap errors in `mapError` (using `%w`), verify that `errors.Is(err, store.ErrConflict)` still works. If you chose to replace, try wrapping instead and see how it changes your test assertions and error messages.

5. **Add a method.** Add `GetUserByEmail` to the `UserStore` interface, implement it in `PostgresStore`, and write a test. You'll need to add the SQL query first (`sql/queries/users.sql`), run `sqlc generate`, then implement the store method. This is the full cycle: SQL → sqlc → store → test.


## Reference

- [testcontainers-go Snapshot/Restore](https://golang.testcontainers.org/modules/postgres/#using-snapshots)
- [Go blog: Errors are values](https://go.dev/blog/errors-are-values)
- [fmt.Errorf %w verb](https://pkg.go.dev/fmt#Errorf)
