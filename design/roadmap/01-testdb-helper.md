# 01 — Shared Test Database Helper

In this step you'll build `internal/testdb/`, a package that any test in the project can import to get a connection to a real, migrated Postgres database running in a Docker container.

By the end of this doc, you'll have a `Setup` function that:
- Starts a Postgres container (once, shared across all tests in a run)
- Runs your goose migrations against it
- Returns a `*sql.DB` ready for queries
- Cleans up the container when tests finish


## Why a shared helper

Every test package that needs a real database will need the same boilerplate: start a container, get a connection string, run migrations. If you duplicate that, you'll have multiple containers starting up and migrations running in parallel. Worse, when the setup changes (new migration, different Postgres version), you'd have to update every test file.

A single `internal/testdb` package solves this. Import it, call `Setup`, get a database.


## Step 1: Add dependencies

You need two new modules:

```bash
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
go get github.com/pressly/goose/v3
```

You already have `github.com/lib/pq` in your `go.mod` — that's the database driver, and it stays.

> **Go idiom: module management.** `go get` adds the dependency to `go.mod` and downloads it. After adding dependencies, run `go mod tidy` to clean up anything unused. Get in the habit of checking `go.mod` after adding dependencies to see what changed.


## Step 2: Creating migrations with goose

Your migrations live in `sql/schema/`. When you need a new migration, always use `goose create` rather than creating the file by hand:

```bash
goose -dir sql/schema create add_users sql
# creates sql/schema/20260404120000_add_users.sql
```

This generates a timestamped filename. Timestamps matter when multiple developers are creating migrations concurrently — two developers will never collide on a version number. If you name files manually with sequential numbers (`000001_`, `000002_`), concurrent branches will conflict.

Before a release, `goose fix` renumbers timestamps into clean sequential order. See [devops/02-migration-discipline.md](devops/02-migration-discipline.md) for the full workflow.

> **Convention:** Always use `goose create`. Never name migration files by hand.


## Step 3: Understand embed.FS

Your migrations live in `sql/schema/`. Tests run from the package directory (e.g., `internal/testdb/`), so a relative path like `../../sql/schema` is fragile and won't work if you run tests from the project root.

Go's `embed` package solves this. You declare a variable with a `//go:embed` directive, and the compiler bakes the files into the binary at build time.

The catch: `//go:embed` can only embed files relative to the file containing the directive, and it cannot use `..` to go up directories. So you can't embed `../../sql/schema` from `internal/testdb/`.

There are two clean solutions:

**Option A: Embed at the project root and pass it in.** Create a file at the project root (or in `sql/`) that embeds the schema directory, then have your test helper accept an `fs.FS` parameter.

**Option B: Embed in a dedicated package.** Create a small package like `internal/schema/` whose only job is to embed and export the migration files.

Option B is cleaner because it keeps the embed declaration close to the files and any test package can import it without passing things around.

Here's the shape:

```go
// internal/schema/schema.go
package schema

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
```

But wait — the SQL files live in `sql/schema/`, not `internal/schema/migrations/`. You have a choice: move the files, or symlink, or adjust the embed path. The simplest approach is to keep the files where they are and create the embed package in a location that can reach them.

Think about where to put the embed declaration so that the `//go:embed` path can reach `sql/schema/*.sql`. Remember: the path is relative to the Go source file.

> **Hint:** A file at the project root can embed `sql/schema/*.sql`. A file in `sql/` can embed `schema/*.sql`. Either works. The embed package just needs to be importable.

> **Go idiom: embed.FS.** The embedded filesystem implements `fs.FS`, which is the standard interface for read-only file trees. Both `goose.NewProvider` and `os.DirFS` work with `fs.FS`. The difference is that `embed.FS` is baked into the binary — no filesystem access needed at runtime.


## Step 4: Run goose migrations programmatically

You've been running migrations via the CLI (`goose postgres "..." up`). In tests, you'll use goose as a library.

The modern goose API uses a `Provider`:

```go
provider, err := goose.NewProvider(
    goose.DialectPostgres,
    db,           // *sql.DB
    migrationsFS, // fs.FS containing your .sql files
)
// ...
results, err := provider.Up(ctx)
```

One subtlety: if your `embed.FS` embeds files under a subdirectory (e.g., `//go:embed sql/schema/*.sql` produces paths like `sql/schema/001_users.sql`), you need to use `fs.Sub` to strip the prefix before passing it to goose. Goose expects the `.sql` files to be at the root of the `fs.FS`.

```go
subFS, err := fs.Sub(rawFS, "sql/schema")
```

> **Go idiom: fs.Sub.** This is a standard library function that returns a new `fs.FS` rooted at a subdirectory. It's the `fs.FS` equivalent of `cd`-ing into a directory.


## Step 5: The sync.Once pattern

You want one container for the entire test run, not one per test or per package. Go's `sync.Once` guarantees a function runs exactly once, even if called from multiple goroutines.

The shape:

```go
var (
    once     sync.Once
    setupErr error
    testDB   *sql.DB
    // ... container reference for cleanup
)

func Setup(t *testing.T) *sql.DB {
    t.Helper()
    once.Do(func() {
        // start container
        // get connection string
        // open *sql.DB
        // run migrations
        // store in package-level vars
    })
    if setupErr != nil {
        t.Fatalf("testdb setup: %v", setupErr)
    }
    return testDB
}
```

A few things to notice:

- `t.Helper()` marks this function as a test helper, so when a test fails, the error points to the caller, not to this function.
- `once.Do` captures errors in a package-level variable because you can't call `t.Fatal` inside `once.Do` (it might not be the first caller's `t`).
- The container stays alive for the duration of the test process. Go's test runner will exit and Docker will clean up the container via testcontainers' resource reaper.

> **Go idiom: sync.Once.** This is the standard way to do lazy, thread-safe initialization in Go. It's simpler than `init()` because it only runs when actually needed, and it's safe for concurrent access. You'll see it used for database connections, config loading, and expensive setup.


## Step 6: Start the testcontainer

The testcontainers postgres module gives you a high-level API:

```go
ctr, err := postgres.Run(ctx,
    "postgres:16-alpine",
    postgres.WithDatabase("chirpy_test"),
    postgres.WithUsername("test"),
    postgres.WithPassword("test"),
    postgres.BasicWaitStrategies(),
)
```

After the container starts, get the connection string:

```go
connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
```

Then open a `*sql.DB` with your existing `lib/pq` driver:

```go
db, err := sql.Open("postgres", connStr)
```

Don't forget to import `_ "github.com/lib/pq"` for the driver side effect.

> **Go idiom: blank import for drivers.** The `_ "github.com/lib/pq"` import runs the package's `init()` function, which registers the "postgres" driver with `database/sql`. You never call `pq` directly — you use it through the `database/sql` interface. This is Go's standard pattern for database drivers.


## Step 7: Put it all together

Create `internal/testdb/testdb.go` (or whatever file name you prefer). It should:

1. Import your embedded migrations
2. Provide a `Setup(t *testing.T) *sql.DB` function
3. Use `sync.Once` to start the container and run migrations exactly once
4. Return the shared `*sql.DB`

The function body inside `once.Do` should:
1. Start the postgres container
2. Get the connection string
3. Open a `*sql.DB`
4. Ping it to verify connectivity
5. Create a goose provider with your embedded migrations
6. Run `provider.Up(ctx)`
7. Store the `*sql.DB` in the package-level variable

Keep error handling explicit — if any step fails, store the error and return early. Every subsequent call to `Setup` will see the error and fail the test.


## Step 8: Write a verification test

Create `internal/testdb/testdb_test.go`. Write a test that:

1. Calls `Setup(t)` to get a `*sql.DB`
2. Pings the database
3. Queries `information_schema.tables` to verify the `users` table exists

```go
func TestSetup(t *testing.T) {
    db := Setup(t)

    // Can we reach the database?
    if err := db.Ping(); err != nil {
        t.Fatalf("ping: %v", err)
    }

    // Did migrations run?
    var tableName string
    err := db.QueryRow(
        "SELECT table_name FROM information_schema.tables WHERE table_name = $1",
        "users",
    ).Scan(&tableName)
    if err != nil {
        t.Fatalf("users table not found: %v", err)
    }
}
```

Run it:

```bash
go test -v ./internal/testdb/...
```

You should see testcontainers log output showing the container starting, then your test passing. The first run will be slow (pulling the Docker image). Subsequent runs will be faster.


## Verify

- [ ] `go test -v ./internal/testdb/...` passes
- [ ] You see container startup logs in the output
- [ ] The test confirms the `users` table exists
- [ ] Running the test a second time still passes (idempotent)
- [ ] `go mod tidy` leaves no unused dependencies


## Explore

1. **What happens without `sync.Once`?** Remove it temporarily and call `Setup` from two tests in the same file. Do you get two containers? What errors do you see?

2. **Container logs.** Add `testcontainers.WithLogger(...)` or just watch the test output with `-v`. What does the container startup sequence look like? How long does it take?

3. **What if migrations fail?** Temporarily break a migration file (add invalid SQL). What error do you get? Is it clear enough to debug?

4. **Inspect the running container.** While a test is running (add a `time.Sleep` if needed), run `docker ps` in another terminal. Can you see the testcontainer? Can you `docker exec` into it and run `psql`?

5. **Read the testcontainers-go docs.** Look at the [Postgres module reference](https://golang.testcontainers.org/modules/postgres/) — especially the Snapshot/Restore section. You'll use that in doc 04.


## Reference

- [testcontainers-go Postgres module](https://golang.testcontainers.org/modules/postgres/)
- [goose Provider API](https://pressly.github.io/goose/blog/2023/goose-provider/)
- [Go embed package](https://pkg.go.dev/embed)
- [fs.Sub](https://pkg.go.dev/io/fs#Sub)
- [sync.Once](https://pkg.go.dev/sync#Once)
