# 02 — sqlc Query-Level Integration Tests

In this step you'll write tests that exercise your sqlc-generated `Queries` methods against a real Postgres database. These are the base of your test pyramid — they verify that the SQL you wrote actually works.

By the end of this doc, you'll have `internal/database/queries_test.go` with tests for `CreateUser` that run against a real Postgres container using transaction rollback for isolation.


## Why test at this layer

sqlc generates Go code from your SQL. That generation can go wrong in subtle ways:

- A `UUID` column that you expected to be `uuid.UUID` might become `pgtype.UUID` depending on your sqlc config
- A `NOT NULL` column with a default might still require an explicit value in the generated params struct
- Timestamp precision might not round-trip the way you expect
- Unique constraints produce database errors that your code needs to handle correctly

These are all things that only show up when you run the query against a real Postgres. A mock won't catch them.

The goal here is not to test Postgres itself — it's to test that **your SQL and sqlc's generated Go code agree with your schema**.


## The DBTX interface

Open `internal/database/db.go` and look at what sqlc generated:

```go
type DBTX interface {
    ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
    PrepareContext(context.Context, string) (*sql.Stmt, error)
    QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
    QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
    return &Queries{db: db}
}
```

Both `*sql.DB` and `*sql.Tx` satisfy this interface. That's the key insight: you can pass a transaction to `New()`, run your queries inside it, and then roll back. The queries never know the difference.

> **Go idiom: interface satisfaction.** Go interfaces are satisfied implicitly — there's no `implements` keyword. If a type has all the methods an interface requires, it satisfies it. `*sql.Tx` has `ExecContext`, `PrepareContext`, `QueryContext`, and `QueryRowContext`, so it satisfies `DBTX` without any declaration. This is why sqlc's design works so cleanly for testing.


## The transaction rollback pattern

The pattern for each test:

```go
func TestSomething(t *testing.T) {
    db := testdb.Setup(t)

    tx, err := db.BeginTx(context.Background(), nil)
    if err != nil {
        t.Fatalf("begin tx: %v", err)
    }
    defer tx.Rollback()

    queries := database.New(tx)

    // ... use queries, make assertions ...
}
```

After the test function returns, `defer tx.Rollback()` undoes everything. The next test starts with a clean database. No data leaks between tests, and it's fast because there's no I/O to disk (Postgres keeps uncommitted data in memory).

The one limitation: if you need to test behavior that depends on committed data being visible to other connections (e.g., testing that a unique constraint works across concurrent transactions), you'll need a different isolation strategy. That's what snapshot/restore is for in doc 06. For query-level tests, transaction rollback is the right tool.


## Step 1: Create the test file

Create `internal/database/queries_test.go`. This file will contain your query-level integration tests.

Start with the package declaration and imports:

```go
package database_test
```

Notice: `database_test`, not `database`. This is an external test package — it can only access exported symbols from the `database` package, just like any other consumer of your code. This is intentional: you're testing the public API of the generated code.

> **Go idiom: external test packages.** Using `package foo_test` instead of `package foo` means your tests can only use the exported API. This catches accidentally relying on unexported internals. For generated code like sqlc output, this is especially valuable — you want to test the interface that your application code will actually use.

You'll need these imports:

```go
import (
    "context"
    "testing"
    "time"

    "github.com/google/uuid"
    "github.com/rjpw/bootdev-chirpy/internal/database"
    "github.com/rjpw/bootdev-chirpy/internal/testdb"
)
```


## Step 2: Write a helper to get a transactional Queries instance

You'll repeat the "begin tx, create queries, defer rollback" pattern in every test. Extract it into a helper:

```go
func setupTx(t *testing.T) *database.Queries {
    t.Helper()
    db := testdb.Setup(t)
    tx, err := db.BeginTx(context.Background(), nil)
    if err != nil {
        t.Fatalf("begin tx: %v", err)
    }
    t.Cleanup(func() {
        tx.Rollback()
    })
    return database.New(tx)
}
```

Notice the use of `t.Cleanup` instead of `defer`. Inside a helper function, `defer` would run when the helper returns, not when the test finishes. `t.Cleanup` registers a function that runs after the test completes, which is what you want.

> **Go idiom: t.Cleanup vs defer.** In test helpers, always use `t.Cleanup()` for teardown. `defer` is scoped to the function it's in, but `t.Cleanup` is scoped to the test. This is a common gotcha when extracting test setup into helper functions.


## Step 3: Test CreateUser — happy path

Write a test that creates a user and verifies the returned fields:

```go
func TestCreateUser(t *testing.T) {
    queries := setupTx(t)
    ctx := context.Background()

    now := time.Now().UTC().Truncate(time.Microsecond)
    params := database.CreateUserParams{
        ID:        uuid.New(),
        CreatedAt: now,
        UpdatedAt: now,
        Email:     "alice@example.com",
    }

    user, err := queries.CreateUser(ctx, params)
    if err != nil {
        t.Fatalf("CreateUser: %v", err)
    }

    // Verify all fields round-trip correctly
    if user.ID != params.ID {
        t.Errorf("ID: got %v, want %v", user.ID, params.ID)
    }
    if user.Email != params.Email {
        t.Errorf("Email: got %q, want %q", user.Email, params.Email)
    }
    // ... check timestamps too
}
```

A few things to notice:

- `time.Now().UTC().Truncate(time.Microsecond)` — Postgres `TIMESTAMP` has microsecond precision. Go's `time.Time` has nanosecond precision. If you don't truncate, the round-trip comparison will fail because Postgres silently drops the extra precision. This is exactly the kind of thing these tests catch.
- `uuid.New()` generates the ID on the Go side. Your schema has `id UUID PRIMARY KEY` with no default, so the caller must provide it.

> **Go idiom: time truncation for database tests.** Always truncate to the database's precision before comparing. For Postgres `TIMESTAMP`, that's microseconds. For MySQL `DATETIME`, it's seconds. This is a classic source of flaky tests.


## Step 4: Test CreateUser — duplicate email

Your schema has `email TEXT NOT NULL UNIQUE`. What happens when you try to insert a duplicate?

```go
func TestCreateUserDuplicateEmail(t *testing.T) {
    queries := setupTx(t)
    ctx := context.Background()

    now := time.Now().UTC().Truncate(time.Microsecond)
    email := "duplicate@example.com"

    // First insert should succeed
    _, err := queries.CreateUser(ctx, database.CreateUserParams{
        ID: uuid.New(), CreatedAt: now, UpdatedAt: now, Email: email,
    })
    if err != nil {
        t.Fatalf("first CreateUser: %v", err)
    }

    // Second insert with same email should fail
    _, err = queries.CreateUser(ctx, database.CreateUserParams{
        ID: uuid.New(), CreatedAt: now, UpdatedAt: now, Email: email,
    })
    if err == nil {
        t.Fatal("expected error for duplicate email, got nil")
    }

    // Verify it's the right kind of error
    // ... see below
}
```

To check that it's specifically a unique violation (not some other error), you need to inspect the Postgres error code. The `lib/pq` driver returns errors of type `*pq.Error`:

```go
import "github.com/lib/pq"

var pgErr *pq.Error
if errors.As(err, &pgErr) {
    if pgErr.Code != "23505" { // unique_violation
        t.Errorf("expected unique_violation (23505), got %s: %s", pgErr.Code, pgErr.Message)
    }
} else {
    t.Errorf("expected *pq.Error, got %T: %v", err, err)
}
```

> **Go idiom: errors.As for type assertion on errors.** `errors.As` unwraps the error chain and checks if any error in the chain matches the target type. It's the error-handling equivalent of a type assertion, but it works through wrapped errors. Always prefer `errors.As` over direct type assertions on errors.

The Postgres error code `23505` is the standard code for unique constraint violations. You can find the full list in the [PostgreSQL error codes documentation](https://www.postgresql.org/docs/current/errcodes-appendix.html). Knowing how to inspect these codes is important — you'll use this in doc 06 when mapping database errors to store-level sentinel errors.


## Step 5: Consider table-driven tests (optional)

For `CreateUser` you might not need table-driven tests — there are only a couple of cases and they have different setup requirements. But as you add more queries (e.g., `GetUserByEmail`, `ListUsers`), table-driven tests become valuable.

The shape:

```go
func TestGetUserByEmail(t *testing.T) {
    cases := []struct {
        name    string
        setup   func(t *testing.T, q *database.Queries) // seed data
        email   string
        wantErr bool
    }{
        {
            name:  "existing user",
            setup: func(t *testing.T, q *database.Queries) { /* insert a user */ },
            email: "exists@example.com",
        },
        {
            name:    "nonexistent user",
            email:   "ghost@example.com",
            wantErr: true,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            queries := setupTx(t)
            if tc.setup != nil {
                tc.setup(t, queries)
            }
            // ... run query, check results
        })
    }
}
```

Each subtest gets its own transaction (via `setupTx`), so they're fully isolated.

> **Go idiom: table-driven tests with subtests.** The `cases` slice + `t.Run` loop is the standard Go testing pattern. Each entry in the slice is a test case with a name, inputs, and expected outputs. `t.Run` creates a subtest that shows up individually in test output and can be run in isolation with `-run TestGetUserByEmail/existing_user`.


## Step 6: Run your tests

```bash
go test -v ./internal/database/...
```

You should see:
- Testcontainer startup (first time only, shared with any other packages that use `testdb.Setup`)
- Each test running and passing
- No leftover data between tests (thanks to rollback)

Run it twice to confirm idempotency.


## Verify

- [ ] `go test -v ./internal/database/...` passes
- [ ] `TestCreateUser` verifies all fields including timestamps
- [ ] `TestCreateUserDuplicateEmail` checks for the specific Postgres error code
- [ ] Running tests twice produces identical results (rollback isolation works)
- [ ] Tests use `database_test` as the package name (external test package)


## Explore

1. **What happens without rollback?** Comment out the `t.Cleanup` in `setupTx`. Run the tests. Do they still pass? Run them again. What happens on the second run? Why?

2. **Cross-test visibility.** Write two tests: one that inserts a user, and one that queries for that user. Does the second test find the user? Why or why not? (Think about what transaction isolation means.)

3. **Error messages.** Make a test fail intentionally (e.g., assert the wrong email). Is the error message clear enough to debug? If not, improve it. Good test failure messages are an underrated skill.

4. **Inspect the generated SQL.** Open `internal/database/users.sql.go` and read the `createUser` constant. Compare it to `sql/queries/users.sql`. What did sqlc add? (Look at the `RETURNING` clause.)

5. **Add a query.** Add a new query to `sql/queries/users.sql` (e.g., `GetUserByEmail`), run `sqlc generate`, and write a test for it. This is the red-green cycle: write the test first, then the SQL, then generate, then make it pass.


## Reference

- [database/sql.Tx](https://pkg.go.dev/database/sql#Tx)
- [errors.As](https://pkg.go.dev/errors#As)
- [lib/pq error handling](https://pkg.go.dev/github.com/lib/pq#Error)
- [PostgreSQL error codes](https://www.postgresql.org/docs/current/errcodes-appendix.html)
- [Go testing package](https://pkg.go.dev/testing)
