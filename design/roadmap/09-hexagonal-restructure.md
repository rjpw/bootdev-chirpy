# Hexagonal Restructure

Reorganize the project to reflect domain-driven design and hexagonal architecture. The domain becomes the center of the dependency graph. Adapters depend on the domain; the domain depends on nothing internal. Infrastructure packages nest under the adapter they serve.

This is a pure refactor — no behavior changes, no new features. Every test must pass after each step.


## Why now

Doc 10 (scaling the store layer) adds a second entity and the `Repositories` struct. Doc 12 (always-on readiness) adds health check infrastructure. Both are easier to build correctly if the package boundaries already reflect the architecture. Moving packages later means rewriting more imports and touching more files.


## Current layout

```
internal/
  api/              ← HTTP handlers (driving adapter, unnamed)
  config/           ← composition root
  database/         ← sqlc-generated code (top-level, but only postgres uses it)
  metrics/          ← server metrics
  schema/           ← embedded migrations + query SQL (top-level, but only postgres/testdb use it)
  store/            ← domain types AND infrastructure concern
    store.go        ← User, UserStore — domain lives here
    errors.go       ← ErrNotFound, ErrConflict
    memory/         ← adapter, nested under the domain it implements
    postgres/       ← adapter, nested under the domain it implements
  testdb/           ← test helper for postgres (top-level, but only postgres tests use it)
```

Problems:
- Domain types live inside an infrastructure package (`store`)
- Adapters are children of the port they implement — the dependency arrow is ambiguous
- `api` doesn't name its transport; nothing signals it's an adapter
- `database/`, `schema/`, and `testdb/` are implementation details of the Postgres adapter but sit at the same level as the domain
- `internal/` has nine top-level entries; five of them are Postgres plumbing


## Target layout

```
internal/
  domain/                ← the core: types, ports (interfaces), domain errors
    user.go              ← User type, UserRepository interface
    errors.go            ← ErrNotFound, ErrConflict
  httpapi/               ← driving adapter: HTTP handlers
  postgres/              ← driven adapter: implements domain ports
    repository.go        ← Repository struct, NewRepository
    users.go             ← UserRepository methods, toUser mapper
    db.go                ← Open, connection setup
    database/            ← sqlc-generated code (private to this adapter)
    schema/              ← migrations + query SQL + embed.FS
    testdb/              ← testcontainers helper (only used by tests in this tree)
  memory/                ← driven adapter: implements domain ports (tests)
    repository.go
    users.go
  config/                ← composition root
  metrics/               ← unchanged
```

Key properties:
- `domain/` imports nothing from `internal/`. This is the hex rule.
- `postgres/` and `memory/` import `domain/`. They are siblings, not children of the domain.
- `httpapi/` imports `domain/`. Named for the transport it adapts.
- `config/` imports `domain/` (for the port types in its struct).
- `cmd/chirpy/main.go` imports `config/`, `postgres/`, `httpapi/`, and `postgres/schema` — it's the composition root.
- `database/`, `schema/`, and `testdb/` are nested under `postgres/` — invisible from `internal/`'s top level.
- `UserStore` → `UserRepository` throughout, reflecting DDD vocabulary.


## Steps

Work one step at a time. Run `make test` after each. Commit after each green.


### Step 1: Extract `domain/` from `store/` and rename to Repository

Create `internal/domain/` and move the domain types out of `store/`:

- `store/store.go` → `domain/user.go`
  - Change package declaration to `domain`
  - Rename `UserStore` → `UserRepository`
- `store/errors.go` → `domain/errors.go`
  - Change package declaration to `domain`

Update all imports and type references:
- `config/config.go` — `store.UserStore` → `domain.UserRepository`
- `api/users.go`, `api/users_test.go` — `store.User` / `store.UserStore` → `domain.User` / `domain.UserRepository`
- `store/memory/store.go`, `store/memory/users.go`, `store/memory/memory_test.go`
- `store/postgres/store.go`, `store/postgres/users.go`, `store/postgres/users_test.go`

The `store/` directory still exists after this step (it contains `memory/` and `postgres/`).


### Step 2: Promote adapters and rename to Repository

Move the adapter packages out from under `store/`:

- `store/memory/` → `memory/` (under `internal/`)
- `store/postgres/` → `postgres/` (under `internal/`)

In each adapter, rename the struct and constructor:
- `Store` → `Repository`
- `NewPostgresStore` → `NewRepository` (or similar)
- `NewPostgresStoreFromURL` → `NewRepositoryFromURL`
- Compile-time checks: `var _ domain.UserRepository = (*Repository)(nil)`

Update imports:
- `cmd/chirpy/main.go`: `internal/store/postgres` → `internal/postgres`
- `api/server_test.go`: `internal/store/memory` → `internal/memory`
- Test files in each adapter package

Delete the now-empty `store/` directory.


### Step 3: Nest Postgres infrastructure

Move the Postgres-specific packages under `postgres/`:

- `internal/database/` → `internal/postgres/database/`
- `internal/schema/` → `internal/postgres/schema/`
- `internal/testdb/` → `internal/postgres/testdb/`

Update imports:
- `postgres/repository.go`, `postgres/users.go`, `postgres/db.go`: `internal/database` → `internal/postgres/database`
- `postgres/testdb/testdb.go`: `internal/schema` → `internal/postgres/schema`
- `cmd/chirpy/main.go`: `internal/schema` → `internal/postgres/schema`
- `database/queries_test.go`: `internal/testdb` → `internal/postgres/testdb`, `internal/database` → `internal/postgres/database`

Update `sqlc.yaml` paths:
```yaml
schema: "internal/postgres/schema/migrations"
queries: "internal/postgres/schema/queries"
out: "internal/postgres/database"
```

Update `Makefile` paths:
- `make test-db` target: `./internal/database/...` → `./internal/postgres/database/...`, etc.
- Any goose commands that reference `internal/schema/migrations`

Update the `embed.FS` in `schema.go` — the `//go:embed` directive uses a path relative to the file, so if `schema.go` moves with its `migrations/` directory, the directive stays the same.

Run `make sql-generate` to verify sqlc still works with the new paths.


### Step 4: Rename `api/` to `httpapi/`

Rename the package to name the transport:

- `api/` → `httpapi/` (rename directory)
- All files: `package api` → `package httpapi`
- Test files: `package api_test` → `package httpapi_test`

Update imports:
- `cmd/chirpy/main.go`: `internal/api` → `internal/httpapi`


### Step 5: Verify the dependency rule

Confirm the dependency graph is correct:

```bash
# domain/ imports nothing from internal/
grep -r 'bootdev-chirpy/internal/' internal/domain/
# should return nothing

# postgres/ and memory/ import only domain/ (not each other, not httpapi/)
grep -r 'internal/httpapi\|internal/memory' internal/postgres/
grep -r 'internal/httpapi\|internal/postgres' internal/memory/
# should return nothing

# httpapi/ does not import adapters
grep -r 'internal/postgres\|internal/memory' internal/httpapi/
# should return nothing
```

Run the full suite:
```bash
make test
make test-integration
```


## Keeping tests green

Each step is a mechanical rename. The process for each:

1. **Move the directory** (`git mv`).
2. **Update `package` declarations** in every `.go` file in the moved package.
3. **Find-and-replace the import path** across the project. `grep -r 'old/path' --include='*.go'` finds them all.
4. **Update non-Go config** — `sqlc.yaml`, `Makefile`, `.air.toml` if it references paths.
5. **`go build ./...`** — if it compiles, the imports are correct.
6. **`make test`** — if tests pass, behavior is unchanged.

The riskiest step is step 3 (nesting Postgres infrastructure) because it touches `sqlc.yaml`, the `Makefile`, and the `embed.FS`. Run `make sql-generate` after updating `sqlc.yaml` to confirm the code generation still works. The embed directive is relative to the file, so it should survive the move unchanged as long as `schema.go` and `migrations/` move together.

Test files that use external test packages (`package memory_test`, `package api_test`) need their package declarations updated too.


## Checklist

- [ ] `internal/domain/` exists with `user.go` and `errors.go`
- [ ] `UserStore` renamed to `UserRepository` throughout
- [ ] `domain/` has zero imports from `internal/`
- [ ] `internal/store/` directory removed
- [ ] `internal/memory/` exists with `Repository` struct
- [ ] `internal/postgres/` exists with `Repository` struct
- [ ] `internal/postgres/database/` — sqlc-generated code (moved from `internal/database/`)
- [ ] `internal/postgres/schema/` — migrations + queries (moved from `internal/schema/`)
- [ ] `internal/postgres/testdb/` — test helper (moved from `internal/testdb/`)
- [ ] `sqlc.yaml` paths updated
- [ ] `Makefile` paths updated
- [ ] `make sql-generate` works
- [ ] `internal/httpapi/` exists (renamed from `api/`)
- [ ] Dependency rule verified (step 5 grep checks pass)
- [ ] `make test` passes
- [ ] `make test-integration` passes
- [ ] Commit after each step
