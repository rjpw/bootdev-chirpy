# 07 — Embedded Migrate Subcommand

In this step you'll add a `migrate` subcommand to the chirpy binary so that the compiled application can run its own database migrations without the goose CLI or raw SQL files on disk.

By the end of this doc:
- `./chirpy migrate up` applies all pending migrations
- `./chirpy migrate status` shows which migrations have been applied
- The SQL files are embedded in the binary at compile time
- Customers deploy a single binary — no external tools required


## Why this matters

Your testdb helper already embeds migrations and runs them with goose-as-library. That proves the pattern works. But it's test-only infrastructure. A customer deploying your binary currently needs:

1. The goose CLI installed
2. The raw SQL files on disk
3. Knowledge of the `goose` command syntax

With an embedded migrate subcommand, they need:

1. The binary

That's it. `./chirpy migrate up` against their connection string. The SQL is baked in. See [design/sql-vs-go-migrations.md](../design/sql-vs-go-migrations.md) for the rationale.


## Step 1: Reuse the embedded migrations

You already have (or will have) an embed package from doc 01. The same `embed.FS` that testdb uses is what the migrate subcommand will use. No duplication — one embed declaration, two consumers.

If your embed lives in `internal/schema/`:

```go
// internal/schema/schema.go
package schema

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
```

Both `internal/testdb/` and `cmd/chirpy/` import `schema.Migrations`. Single source of truth.


## Step 2: Parse subcommands in main

Go's standard library doesn't have a subcommand framework, but you don't need one. A simple `os.Args` check is enough:

```go
func main() {
    if len(os.Args) > 1 && os.Args[1] == "migrate" {
        runMigrate(os.Args[2:])
        return
    }

    // ... existing server startup
}
```

This keeps the server as the default behavior. `./chirpy` starts the server. `./chirpy migrate <command>` runs migrations and exits.


## Step 3: Implement the migrate command

Create a helper in `cmd/chirpy/` (not in `internal/` — this is entry-point plumbing):

```go
func runMigrate(args []string) {
    if len(args) == 0 {
        fmt.Fprintf(os.Stderr, "usage: chirpy migrate [up|status]\n")
        os.Exit(1)
    }

    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        fmt.Fprintf(os.Stderr, "DATABASE_URL is required\n")
        os.Exit(1)
    }

    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        fmt.Fprintf(os.Stderr, "open db: %v\n", err)
        os.Exit(1)
    }
    defer db.Close()

    migrationsFS, err := fs.Sub(schema.Migrations, "migrations")
    if err != nil {
        fmt.Fprintf(os.Stderr, "migrations fs: %v\n", err)
        os.Exit(1)
    }

    provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
    if err != nil {
        fmt.Fprintf(os.Stderr, "goose provider: %v\n", err)
        os.Exit(1)
    }

    ctx := context.Background()

    switch args[0] {
    case "up":
        results, err := provider.Up(ctx)
        if err != nil {
            fmt.Fprintf(os.Stderr, "migrate up: %v\n", err)
            os.Exit(1)
        }
        for _, r := range results {
            fmt.Printf("applied: %s (%s)\n", r.Source.Path, r.Duration)
        }
    case "status":
        results, err := provider.Status(ctx)
        if err != nil {
            fmt.Fprintf(os.Stderr, "migrate status: %v\n", err)
            os.Exit(1)
        }
        for _, r := range results {
            fmt.Printf("%-5s %s\n", r.State, r.Source.Path)
        }
    default:
        fmt.Fprintf(os.Stderr, "unknown migrate command: %s\n", args[0])
        fmt.Fprintf(os.Stderr, "usage: chirpy migrate [up|status]\n")
        os.Exit(1)
    }
}
```

Notice:
- Connection string comes from `DATABASE_URL` environment variable — same convention most deployment tools expect
- `fs.Sub` strips the subdirectory prefix, same as in testdb
- The function prints results and exits — it doesn't start the server


## Step 4: Consider what NOT to expose

You might be tempted to add `down`, `down-to`, `redo`, etc. Resist this for now. Destructive migration commands in a production binary are dangerous. A customer accidentally running `./chirpy migrate down` against production is a bad day.

Start with `up` and `status`. If you add `down` later, consider requiring a confirmation flag:

```bash
./chirpy migrate down --confirm
```

For development, the goose CLI is still available and has the full command set. The embedded subcommand is for deployment, where the surface area should be small.


## Step 5: Test it

You can't easily unit test `runMigrate` (it calls `os.Exit`), but you can integration test the goose provider logic by extracting it:

```go
func applyMigrations(ctx context.Context, db *sql.DB, migrationsFS fs.FS) ([]*goose.MigrationResult, error) {
    provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
    if err != nil {
        return nil, err
    }
    return provider.Up(ctx)
}
```

This is already tested implicitly by your testdb helper — it does the same thing. But if you want an explicit test, tag it with `//go:build integration` and use testcontainers.

For a manual smoke test:

```bash
# build the binary
make build

# run migrations (assumes DATABASE_URL is set)
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable"
./tmp/main migrate status
./tmp/main migrate up
./tmp/main migrate status
```


## Step 6: Document it

Add to the README:

```markdown
## Migrations

Apply migrations using the built-in subcommand:

    export DATABASE_URL="postgres://user:pass@host:5432/chirpy?sslmode=disable"
    ./chirpy migrate up

Check migration status:

    ./chirpy migrate status

For development, you can also use the goose CLI directly:

    goose -dir sql/schema postgres "$DATABASE_URL" up
```

Update the release notes template in [devops/03-release-process.md](devops/03-release-process.md) to reference the subcommand:

```markdown
**Upgrade:**

    export DATABASE_URL="postgres://..."
    ./chirpy migrate up
```


## Verify

- [ ] `./chirpy` (no args) starts the server as before
- [ ] `./chirpy migrate up` applies pending migrations
- [ ] `./chirpy migrate status` shows applied/pending state
- [ ] `./chirpy migrate` (no subcommand) prints usage
- [ ] `./chirpy migrate nonsense` prints usage
- [ ] The binary contains the embedded SQL — no files needed on disk
- [ ] `make build` still works


## Explore

1. **Inspect the binary.** Run `strings tmp/main | grep "CREATE TABLE"`. Can you see your migration SQL embedded in the binary? This confirms the embed is working.

2. **Compare to testdb.** Open `internal/testdb/testdb.go` and `cmd/chirpy/migrate.go` side by side. How much code is shared? Could you extract a common `internal/migrate/` package that both use? Is that worth it, or is the duplication small enough to tolerate?

3. **What about `down`?** Think about what would happen if a customer ran `./chirpy migrate down` against production. What safeguards would you want? A `--confirm` flag? A `--dry-run` that shows what would be rolled back? How does this compare to the goose CLI's behavior?


## Reference

- [goose Provider API](https://pressly.github.io/goose/blog/2023/goose-provider/)
- [Go embed package](https://pkg.go.dev/embed)
- [design/sql-vs-go-migrations.md](../design/sql-vs-go-migrations.md)
