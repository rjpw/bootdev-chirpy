# 06 — Developer Workflow

In this step you'll set up the day-to-day workflow for running tests: build tags to separate fast tests from slow container tests, makefile targets for convenience, and documentation so future-you (or a teammate) knows how it all works.

By the end of this doc:
- Container-based tests are gated behind a `//go:build integration` tag
- `make test` runs only fast tests (no Docker required)
- `make test-integration` runs everything including container tests
- The README documents the testing approach


## Why build tags

Right now, running `go test ./...` starts a Docker container. That's fine when you want it, but it's slow for the red-green-refine-refactor loop where you're iterating on handler logic that doesn't touch the database.

Go's build tags let you conditionally include files in a build. By tagging your container-based test files with `//go:build integration`, they'll be skipped by default and only included when you explicitly pass `-tags integration`.

This gives you two speeds:
- **Fast (default):** `go test ./...` runs API handler tests, unit tests, anything that doesn't need Docker. Sub-second.
- **Full:** `go test -tags integration ./...` runs everything, including container tests. A few seconds for container startup, then fast.

> **Go idiom: build tags for test tiers.** This is a common pattern in Go projects. Other popular tag names include `e2e`, `db`, and `slow`. The tag name is arbitrary — pick one and be consistent. `integration` is the most widely used.


## Step 1: Add build tags to container test files

Every test file that imports `testdb` (or directly uses testcontainers) needs the build tag. Add this as the very first line of the file, before the package declaration:

```go
//go:build integration

package database_test
```

The files that need this tag:
- `internal/testdb/testdb_test.go`
- `internal/database/queries_test.go`
- `internal/store/postgres/store_test.go`

The files that do NOT need this tag:
- `internal/api/*_test.go` — these use `httptest` and don't touch a database
- `internal/store/store.go`, `errors.go` — these aren't test files

> **Go idiom: build tag syntax.** The `//go:build` line must be the first line of the file (before package), with a blank line after it. The older syntax `// +build integration` still works but `//go:build` is preferred since Go 1.17. The tag is a boolean expression — you can use `&&`, `||`, and `!`. For example, `//go:build integration && !race` would skip the file when running with `-race`.

> **Important:** The `//go:build` directive controls whether the file is compiled at all. Without the tag, `go test ./...` won't even see the test functions in that file. This is different from `testing.Short()` which skips at runtime — build tags skip at compile time, which is faster and cleaner.


## Step 2: Verify the split

Run the fast tests:

```bash
go test ./...
```

You should see your API handler tests run, but no container startup. The integration test files are invisible.

Now run with the tag:

```bash
go test -tags integration ./...
```

You should see container startup and all tests (API + query + store) running.

If you see container tests running without the tag, check that the `//go:build` line is the very first line of the file with no leading whitespace or blank lines above it.


## Step 3: Update the Makefile

Open your `makefile` and update it:

```makefile
test:
	go test -race ./...

test-integration:
	go test -race -tags integration ./...
```

The existing `test` target already does `go test -race ./...`, so it automatically becomes the "fast" target once you add build tags. The new `test-integration` target adds `-tags integration`.

Also add migration targets if you haven't already:

```makefile
sql-create:
	@read -p "Migration name: " name; \
	goose -dir sql/schema create $$name sql

sql-fix:
	goose -dir sql/schema fix
```

`sql-create` prompts for a name and generates a timestamped migration file. `sql-fix` renumbers timestamps into sequential order before a release. See [devops/02-migration-discipline.md](devops/02-migration-discipline.md) for when to use each.

> **Hint:** Consider adding `-count=1` to `test-integration` to disable test caching. Cached integration tests can mask issues when the database schema changes but the Go code hasn't:
> ```makefile
> test-integration:
> 	go test -race -tags integration -count=1 ./...
> ```

You might also want a target that runs only the integration tests (not the fast ones too):

```makefile
test-db:
	go test -race -tags integration -count=1 ./internal/database/... ./internal/store/... ./internal/testdb/...
```

This is useful when you're iterating on SQL or store logic and don't want to wait for unrelated tests.


## Step 4: Consider your workflow

Here's the workflow that emerges:

| Activity | Command | Speed | When |
|----------|---------|-------|------|
| Iterating on handler logic | `make test` | < 1s | Every save (via `air` or manual) |
| Iterating on SQL/store logic | `make test-db` | ~3-5s | After changing SQL or store code |
| Pre-commit check | `make test-integration` | ~5-10s | Before committing |
| Full CI pipeline | `make lint test-integration` | ~15-20s | On push |

The key insight: you don't need the database for most of your development loop. The store interface means handler tests are fast. You only pay the container cost when you're working on the database layer or doing a final check.

> **Go idiom: fast feedback loops.** Go's test runner is already fast. Build tags keep it fast by excluding slow tests from the default path. The goal is that `go test ./...` always completes in under a second for the packages you're actively working on.


## Step 5: Update the README

Add a testing section to your `README.md`. Here's the shape — adapt it to your voice:

```markdown
## Testing

### Quick tests (no Docker required)

```bash
make test
```

Runs API handler tests and any unit tests. No database or Docker needed.

### Integration tests (requires Docker)

```bash
make test-integration
```

Spins up a Postgres container via testcontainers-go, runs goose migrations,
and executes database integration tests. The container is created automatically
and cleaned up after tests finish.

### Test architecture

Tests are organized in three tiers:

- **API handler tests** (`internal/api/*_test.go`) — HTTP-level tests using
  httptest. No database. Use fake store implementations.
- **Store integration tests** (`internal/store/postgres/*_test.go`) — Test the
  PostgresStore against a real Postgres. Use snapshot/restore for isolation.
- **sqlc query tests** (`internal/database/*_test.go`) — Test generated query
  methods against a real Postgres. Use transaction rollback for isolation.

Integration tests are gated behind the `integration` build tag and only run
with `make test-integration`.
```

Adjust the details to match what you actually built. The README should be accurate, not aspirational.


## Step 6: Verify the full workflow

Run through the complete sequence:

```bash
# Fast tests — should pass, no Docker
make test

# Integration tests — should start container, run everything
make test-integration

# Lint — should be clean
make lint

# Server starts — should connect to your local Postgres
make run
```

Everything should be green.


## Verify

- [ ] `make test` runs fast (< 1s) with no container startup
- [ ] `make test-integration` starts a container and runs all tests
- [ ] Build tags are on the correct files (and only those files)
- [ ] `//go:build integration` is the first line of each tagged file
- [ ] README documents the testing approach
- [ ] `make lint` is clean


## Explore

1. **Parallel integration tests.** Try adding `-parallel 2` to `test-integration`. Do the snapshot/restore tests still work? Think about what would happen if two tests tried to restore the database at the same time. (Spoiler: it depends on whether tests in the same package run in parallel by default — they don't unless you call `t.Parallel()`.)

2. **Test caching.** Run `make test-integration` twice. Is the second run faster? Go caches test results. Now change a migration file and run again. Does the cache invalidate? This is why `-count=1` can be useful for integration tests.

3. **CI considerations.** If you were to run these tests in CI (GitHub Actions, etc.), what would you need? Docker-in-Docker or a Docker socket. Most CI environments provide this. Think about what your `.github/workflows/test.yml` would look like.

4. **Build tag boolean expressions.** Try `//go:build integration && !short`. What does this mean? When would the file be included? Read the [build constraints documentation](https://pkg.go.dev/cmd/go#hdr-Build_constraints) for the full syntax.

5. **Test coverage.** Run `go test -tags integration -coverprofile=coverage.out ./...` and then `go tool cover -html=coverage.out`. What's your coverage like for the store and database packages? Is coverage a useful metric for integration tests?


## Reference

- [Go build constraints](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [Go test flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags)
- [Go test caching](https://pkg.go.dev/cmd/go#hdr-Testing_flags) (search for "caching")
