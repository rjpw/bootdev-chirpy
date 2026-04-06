# SQL vs Go Migrations

Goose supports two migration formats: plain SQL files and Go files that register migration functions. This doc explains the tradeoffs and why this project uses SQL.


## SQL migrations

Plain `.sql` files with `-- +goose Up` and `-- +goose Down` sections.

**Pros:**
- Portable. Any DBA, ops person, or developer can read them, run them with `psql`, or use any migration tool. Zero Go knowledge required.
- Auditable. Security review can inspect exactly what runs against production without reading Go code.
- Tool-agnostic. If you outgrow goose, the `.sql` files work with Flyway, Liquibase, Atlas, or raw scripts. The tool is just a runner.
- Diffable. Git diffs on SQL are meaningful.

**Cons:**
- Limited to what SQL can express. Complex data transformations (backfilling from an API, encrypting existing values) get awkward or impossible.
- No type safety. A typo in a column name is a runtime error.


## Go migrations

`.go` files that register functions with goose. They run arbitrary Go code.

**Pros:**
- Full language power for data transformations, API calls, conditional logic.
- Embedded in the binary — no separate files to ship.

**Cons:**
- Lock-in to Go. A DBA can't read them without Go knowledge. Can't run them with `psql`. Can't switch to a non-Go migration tool without rewriting.
- Lock-in to goose. SQL migrations are portable between tools. Go migrations are not.
- Harder to audit. A Go migration could make HTTP calls, read environment variables, write to disk. SQL can only do what SQL does.
- Binary coupling. A migration bug requires shipping a new binary. With SQL, you fix the file and re-run.


## The embedded binary argument

Ops teams like self-contained binaries. `./chirpy migrate up` with no external tools is appealing. But you get that benefit without Go migrations:

```go
//go:embed migrations/*.sql
var migrations embed.FS

func runMigrations(db *sql.DB) error {
    provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations)
    // ...
}
```

SQL files are embedded at compile time. The binary is self-contained. No goose CLI needed on the target machine. But the migrations themselves remain plain SQL — portable, auditable, tool-agnostic.

This is what the project's testdb helper already does.


## The lock-in spectrum

```
most portable                                    most locked-in
     │                                                  │
     ▼                                                  ▼
plain .sql files  →  embedded .sql in Go binary  →  Go migrations
(run with any tool)  (run with goose-as-library)    (run with goose only)
```

This project sits at the middle position: self-contained binary for ops, portable SQL for everyone else.


## When Go migrations are worth it

Rarely, and only for data migrations — not schema migrations.

- Schema changes (add table, add column, add index) are always expressible in SQL. Use SQL.
- Data migrations (backfill by hashing values, split a name column, encrypt plaintext) sometimes need Go.

Goose supports mixing SQL and Go migrations in the same sequence. Use SQL by default. Reach for Go when SQL genuinely can't do the job.


## This project's decision

SQL migrations, embedded in the binary via `embed.FS`, run by goose-as-library. Portability, self-contained deployment, and an escape hatch to Go for the rare case that needs it.
