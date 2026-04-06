# Schema Version Check

## Problem

The server can start and serve requests against a database whose schema is behind what the code expects. A missing table or column produces cryptic SQL errors at runtime instead of a clear failure at startup.

## Decision

Check for pending migrations at two points:

1. **Startup** — refuse to start if any embedded migrations are pending. Fail fast with a message telling the operator to run `./chirpy migrate up`.
2. **Health endpoint** — `/api/healthz` reports unhealthy if migrations are pending. This supports orchestrators (load balancers, container platforms) that check health before routing traffic.

## Why global, not granular

A granular approach — each feature declaring which migrations it depends on — adds a registry that must be maintained alongside every migration. In a monolith where one binary owns the entire schema, this bookkeeping cost exceeds the benefit. The global check ("all known migrations must be applied") is simple, correct, and requires no per-feature annotation.

## Why check, not auto-migrate

Running `provider.Up(ctx)` on startup is tempting but problematic:

- Multiple replicas starting simultaneously can race on migrations
- A bad migration takes down application startup, not just a migration step
- Migration and deployment should be separate, auditable steps (see [devops/03-release-process.md](roadmap/devops/03-release-process.md))

The check is read-only. It inspects `goose_db_version` and the embedded migration files, then reports the gap.

## Interaction with hybrid versioning

Dev databases migrated with timestamp versions (`make sql-migrate`) and release binaries with sequential versions (post-`goose fix`) use different version numbers for the same migrations. The startup check would see all migrations as pending even though the schema is correct.

This is expected and reinforces existing conventions:
- `goose fix` only happens on release branches
- Dev databases are disposable (`docker compose down -v`)
- Production is only migrated by the release binary

## Implementation

See [roadmap/10-schema-version-check.md](roadmap/10-schema-version-check.md).
