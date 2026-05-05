# Doc 10 — Remaining Issues

Tests pass. The hex guardrail enforces correct dependency direction within `internal/`. These issues address coupling that currently lives outside the guardrail's reach.

## 1. Assembly lives in `main.go`, not in `internal/config/`

`main` directly imports `application`, `httpapi`, `postgres`, and `postgres/schema`. It instantiates adapters, wires them to the server, and passes dependencies. This is assembly's job. The guardrail can't see `cmd/` — so the coupling is outside the fence.

**Fix:** Move wiring into `internal/config/` (assembly). `main` should call `config.Run()` or similar — a single entry point that returns an error. `main` handles only process-level concerns: exit codes, signal setup, and calling assembly.

## 2. `postgres.NewPostgresRepositoryFromURL` may construct an application-layer type

If this function returns an application-layer type (e.g., a `Repositories` struct defined in `application/`), then the postgres adapter is constructing a type from an inner ring. The guardrail catches this within `internal/`, but `main` acts as a laundering intermediary — it receives the type from postgres and passes it to httpapi without the guardrail seeing the flow.

**Fix:** `postgres.NewPostgresRepositoryFromURL` returns `*postgres.Repository`. Assembly constructs the application-layer `Repositories` struct from the adapter's return value.

## 3. `&application.ServerMetrics{}` is a concrete type, not an interface

`main` instantiates `application.ServerMetrics` directly. The application layer owns both the contract and the implementation. There's no way to swap metrics (e.g., for testing, or for a Prometheus adapter) without changing `application/`.

**Fix:** Define a metrics interface in `application/`. Move the implementation to an adapter (or keep it in `application/` if it's truly stateless and has no external dependencies). Assembly instantiates and injects it.

## 4. `internal/config/` may be vestigial

`main.go` doesn't import `config`. If `config` has no callers, it's dead code. If it still has a role, `main` is bypassing it.

**Fix:** After issue 1 is resolved, `config` becomes the sole assembly point. Verify it has a caller (`main`) and a clear responsibility.

## 5. Migrate subcommand duplicates DB setup

`runMigrate` opens its own DB connection and reads `DB_URL` from env directly, bypassing `createEnvironment`. Two paths to the database means two places to maintain.

**Fix:** Unify environment loading. Both the serve path and the migrate path should use the same mechanism to obtain the DB connection string. This can be as simple as both calling `createEnvironment()`, or assembly exposing a `MigrateCommand` that reuses the same config.
