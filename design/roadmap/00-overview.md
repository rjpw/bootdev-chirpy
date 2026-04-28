# Roadmap: Database Integration Testing

> **Note:** Docs 01–08 use the original package names (`store`, `UserStore`, `PostgresStore`). Doc 09 restructured the project to hexagonal/DDD conventions (`domain`, `UserRepository`, `Repository`). The diagrams below reflect the original layout; see doc 09 for the current structure.

This series of documents walks you through adding database integration tests to the Chirpy API server. By the end, you'll have:

- A shared test helper that spins up a real Postgres container and runs goose migrations
- Query-level tests that verify your sqlc-generated code against real SQL
- A store interface that decouples your handlers from the database
- A PostgresStore implementation with integration tests using snapshot/restore
- A clean developer workflow with build tags and makefile targets


## The test pyramid for this project

```
            ┌────────────────────┐
            │  API handler tests │  ← httptest, fake store, no DB
            │  (fast, isolated)  │
            ├────────────────────┤
            │  Store integration │  ← PostgresStore against real Postgres
            │  tests             │     snapshot/restore isolation
            │  (committed data)  │
            ├────────────────────┤
            │  sqlc query tests  │  ← database.Queries against real Postgres
            │  (tx rollback)     │     transaction rollback isolation
            └────────────────────┘
```

The base of the pyramid tests that your SQL is correct — that the queries sqlc generated actually work against a real Postgres with your real schema. These are fast because each test rolls back its transaction.

The middle layer tests your store implementation — the code that wraps sqlc queries, maps types, handles errors, and potentially coordinates multiple queries in a transaction. These use snapshot/restore so data is actually committed and visible across connections.

The top layer is what you already have: HTTP-level tests using `httptest` that don't touch a database at all. These stay fast by using a fake or nil store.


## How the tools fit together

```
┌──────────────────────────────────────────────────────┐
│  Your application code                               │
│  (handlers, business logic)                          │
│         │                                            │
│         ▼                                            │
│  store.UserStore interface                           │
│         │                                            │
│         ▼                                            │
│  PostgresStore struct                                │
│         │                                            │
│         ▼                                            │
│  sqlc-generated Queries struct                       │
│         │                                            │
│         ▼                                            │
│  Postgres (schema managed by goose)                  │
└──────────────────────────────────────────────────────┘

  testcontainers-go  →  spins up Postgres in Docker for tests
  goose              →  applies your internal/schema/migrations/*.sql migrations
  sqlc               →  generates Go code from sql/queries/*.sql
```

Each layer only knows about the one directly below it. sqlc and goose are implementation details of the bottom two layers. Your store interface sits above all of it.


## Reading order

| Doc | What you'll build | Key Go concepts |
|-----|-------------------|-----------------|
| [01-store-interface](01-store-interface.md) | `internal/store/store.go`, `errors.go` | Interface design, sentinel errors |
| [02-memory-store](02-memory-store.md) | `internal/store/memory/` — prove the interface | `sync.RWMutex`, interface satisfaction |
| [03-wire-and-integrate](03-wire-and-integrate.md) | Updated `config.go`, `main.go`, API tests | Dependency injection, composition root |
| [04-testdb-helper](04-testdb-helper.md) | `internal/testdb/` — shared test container + migrations | `embed.FS`, `sync.Once`, `t.Cleanup()` |
| [05-sqlc-query-tests](05-sqlc-query-tests.md) | `internal/database/queries_test.go` | `DBTX` interface, tx rollback, table-driven tests |
| [06-postgres-store](06-postgres-store.md) | `internal/store/postgres/` | Snapshot/restore, error mapping, composition |
| [07-developer-workflow](07-developer-workflow.md) | Build tags, makefile targets, README | `//go:build`, test organization |
| [08-migrate-subcommand](08-migrate-subcommand.md) | `./chirpy migrate up` — self-contained binary | `os.Args`, goose provider, `fs.Sub` |
| [09-hexagonal-restructure](09-hexagonal-restructure.md) | Reorganize to domain-driven hexagonal layout | Package structure, dependency rule, ports vs adapters |
| [10-scaling-the-store-layer](10-scaling-the-store-layer.md) | Patterns for adding entities without busy work | Interface-per-entity, file conventions, checklist |
| [11-openapi-codegen](11-openapi-codegen.md) | Contract-first API with OpenAPI spec and code generation | oapi-codegen, spec-driven design, generated routing |
| [12-schema-version-check](12-schema-version-check.md) | Startup gate + healthz for pending migrations | Goose provider status, fail-fast startup |
| [13-always-on-readiness](13-always-on-readiness.md) | SIGTERM, connection pooling, readiness/liveness probes | `syscall.SIGTERM`, `sql.DB` pool config, K8s health checks |

Work through them in order. Each doc builds on the previous one.

The first three docs (01, 02, 03) require no database and no Docker. You'll have a working server with a memory-backed store before touching Postgres. Docs 04, 05, and 06 introduce the database layer incrementally. 


## Prerequisites

Before starting:

- [ ] Docker is running (`docker info` should succeed)
- [ ] Your existing tests pass (`go test -race ./...`)
- [ ] Postgres container is not required to be running — testcontainers will manage its own
- [ ] You've read through `design/design-faq.md` and `design/design-basics.md`
- [ ] You're comfortable with `go test`, `go build`, and your existing makefile targets


## A note on red-green-refine-refactor

Each doc is structured to support your workflow:

1. **Red** — Write a test that fails (or doesn't compile). The doc tells you what to test before telling you how to implement it.
2. **Green** — Make it pass with the simplest thing that works. Hints show the shape of the solution without giving it away.
3. **Refine** — The "Verify" checkpoints confirm you're on track and suggest what to look at.
4. **Refactor** — The "Explore" prompts push you to understand why the code works and what happens when you change it.

Don't skip the explore prompts. They're where the muscle memory comes from.
