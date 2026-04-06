# Schema Version Check

Build a startup gate and health check that prevent the server from running against a stale schema.

Design rationale: [design/schema-version-check.md](../../design/schema-version-check.md)


## What you'll build

- A function that checks for pending migrations using the goose provider
- A startup check in `main.go` that exits if migrations are pending
- An updated `/api/healthz` that reports schema status


## Step 1: The migration checker

You need a function that takes a goose provider and returns whether the schema is current. Think about what it should return — a boolean? An error? A list of pending migration names?

Hints:
- `provider.Status(ctx)` returns `[]*goose.MigrationStatus`
- Each status has a `State` field — compare against `goose.StatePending`
- The caller needs to know two things: is the schema current, and if not, what's missing

### Red

Write a test in a new package (or in an existing one that makes sense). The test should verify that:
- A fully migrated database reports no pending migrations
- A database with unapplied migrations reports them

You already have `testdb.Setup(t)` which gives you a fully migrated database. For the "pending" case, think about how to create a provider that knows about a migration the database hasn't seen.

### Green

Implement the checker. Keep it small — this is a query against goose's own bookkeeping, not your application tables.

### Explore

- What happens if the database has migrations that the binary doesn't know about? (e.g., a newer binary was rolled back to an older one) Should the checker care?
- What does `provider.Status` return for a brand-new database with no `goose_db_version` table?


## Step 2: Startup gate

Wire the checker into `main.go`, after opening the database connection but before starting the HTTP server.

### Red

There's no good way to unit test `main.go` directly. Instead, verify by running the binary against a database with pending migrations and confirming it exits with a clear error message. This is a manual verification step.

### Green

- Build the goose provider the same way `runMigrate` does (embedded `fs.Sub`, `goose.NewProvider`)
- Call your checker
- If pending: `log.Fatalf` with the list of pending migrations and a hint to run `./chirpy migrate up`
- If current: proceed to start the server

Think about where the provider gets created. Right now `runMigrate` builds its own provider. The startup check needs one too. Consider whether they should share construction logic.

### Refactor

Look at `main.go` after this change. You now have three places that open a database connection or build a goose provider: `main` (for the server), `runMigrate`, and the new check. Is there duplication worth extracting?


## Step 3: Health endpoint

Update `/api/healthz` to include schema status.

### Red

Write a test: when the server's schema status is "pending", `GET /api/healthz` should return a non-200 status. When current, it returns 200.

Think about how the handler knows the schema status. The handler runs in the API layer, which doesn't know about goose or migrations. You need to pass the status in — through `Config`, through a field on `Server`, or through a function the handler can call.

Consider: should healthz check the schema on every request (live check), or should it use the result from startup (cached check)?

- Live check: always accurate, but adds a database query to every health check
- Cached check: fast, but won't detect if someone rolls back a migration while the server is running

For this project, the cached check is fine. The startup gate already prevents starting with a stale schema, and mid-flight migration rollbacks are an operational emergency, not something healthz needs to detect in real time.

### Green

- Add a schema status field to `Config` or `Server` (a boolean is enough)
- Set it during startup, after the migration check passes
- Update `handleHealthz` to check the field
- When unhealthy: return 503 Service Unavailable with a body that says why

### Explore

- What should the response body look like? Plain text "OK" is fine for a healthy server, but an unhealthy response should be informative. Consider JSON: `{"status": "unhealthy", "reason": "pending migrations: 00003_add_chirps.sql"}`.
- Should a healthy response also be JSON for consistency? Or is plain text fine since existing clients expect it?
- If you go with JSON, this is a good time to update the existing healthz tests.


## Checklist

- [ ] Migration checker function with tests
- [ ] Startup gate in `main.go`
- [ ] `/api/healthz` reports schema status
- [ ] Healthz tests updated
- [ ] README documents the behavior (what happens when migrations are pending)
