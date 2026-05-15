# Student Boot.dev Repo: Learn HTTP Servers in Go

This repository contains my implementation of the example server defined in the [Boot.dev](https://www.boot.dev) course [Learn HTTP Servers in Go](https://www.boot.dev/courses/learn-http-servers-golang). 

Being my first server written in Golang, I thought I'd better dig a little deeper than the course called for. Here are some of the things this project has that the course didn't explicitly require, but I built anyway:

**Ephemeral E2E Testing** - On a host with Docker, integration tests will spin up Postgres in a `testcontainer`, run all the migrations with `goose up`, and then run a small suite of tests.

**Stateful Test Client** - Sometimes API calls need to happen in a sequence, like `Sign up` -> `Log in` -> `Post a message (authenticated)`. My internal test suites were getting _really_ long, so the test client offers functions structured like a DSL for this course's API, greatly reducing the length of complex test cases.

**Hex Architecture Guardrails** - I wrote a test supported by a JSON data model to remind me to stay within pre-defined guardrails for a [Hexagonal Architecture](https://en.wikipedia.org/wiki/Hexagonal_architecture_(software)).

None of these things was called for on Boot.dev, but I knew from long experience that I would appreciate them. Maybe not in the moment, but definitely over time. For a little background on what inspired these things, I wrote [a blog post](https://www.rjpw.ca/posts/go-go-go/) about encountering ghosts of DB operations past, and then doing something about it. 

The TL;DR is that I didn't want my code to keep anybody up at night, and so I used [Vernon's Implementing DDD](https://www.pearson.com/en-ca/subject-catalog/p/implementing-domain-driven-design/P200000009616/9780133039887) (I know: a book!) and my occasionally frustrating AI advisor Kiro (who in turn consults with one of many Claudes) to steer me toward a Domain-Driven Design. My guidance to Kiro is [here](docs/ai-guidance-notes.md).

## Quickstart for Developers

### Prerequisites

- Go 1.26+ (`go version`)
- Docker running (`docker info`)
- [golangci-lint](https://golangci-lint.run/) (optional but recommended)
- [Air](https://github.com/air-verse/air) for hot reload (optional)

```bash
# install golangci-lint
curl -sSfL https://golangci-lint.run/install.sh | \
  sh -s -- -b $(go env GOPATH)/bin v2.11.4

# install goose and sqlc
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# for automatic server restarts while developing (using `make run`)
go install github.com/air-verse/air@latest

# install dependencies
go mod tidy
```

### Environment

Copy the example environment file and adjust as needed:

```bash
cp .env.example .env
```

Required variables:

| Variable | Description |
|----------|-------------|
| `DB_URL` | Postgres connection string |
| `PLATFORM` | `dev` or `production` |
| `HS256_KEY` |  Symmetrical JWT creation/validation key |
| `POLKA_KEY` | Sample API Key for webhook lesson |

### Database

Run a PostgreSQL server in Docker:

```bash
cat << 'eof' > docker-compose.yaml
services:
  postgres:
    image: postgres:17.2-alpine3.21
    environment:
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
eof

docker compose up -d
```

Create the database and apply migrations:

```bash
export PGPASSWORD=postgres
psql "postgres://postgres:@localhost:5432" -c "CREATE DATABASE chirpy;"

make sql-migrate
```

### Running

```bash
# configure air (with any changes you see fit to make)
cp .air-example.toml .air.toml

# run in development mode
make run

# in another terminal, check health endpoint
curl localhost:8080/api/healthz
```

## Testing

Fast tests (no Docker required):

```bash
make test
```

Integration tests (requires Docker) run against a testcontainers-managed Postgres instance — no manual database setup needed:

```bash
make test-integration
```

Database-only integration tests for faster iteration on SQL and store logic:

```bash
make test-db
```

Integration tests are gated behind a `//go:build integration` tag. `make test` skips them automatically.

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make run` | Start with hot reload (air) |
| `make build` | Compile to `tmp/main` |
| `make test` | Run all tests with race detector |
| `make test-integration` | Run all tests including integration |
| `make test-db` | Run database/store integration tests only |
| `make test-hex-guardrail` | Run hex architecture boundary check only |
| `make lint` | Run golangci-lint |
| `make format` | Auto-format with golangci-lint |
| `make sql-status` | Show migration status |
| `make sql-migrate` | Apply pending migrations |
| `make sql-migrate-down` | Roll back last migration |
| `make sql-generate` | Regenerate sqlc code |
| `make sql-create` | Create a new timestamped migration |
| `make sql-fix` | Renumber migrations for release |
| `make start-db` | Start Postgres via Docker Compose |
| `make stop-db` | Stop Postgres |

## Migrations

In deployed environments, you won't typically have access to the source code or the `goose` command. The build process embeds the migrations within the `chirpy` binary. Note that starting `chirpy` without the migrate subcommand launches the web service.

To apply these migrations, use the built-in subcommand (no goose CLI required):

```bash
export DB_URL="postgres://user:pass@host:5432/chirpy?sslmode=disable"
./chirpy migrate up
```

Check migration status:

```bash
./chirpy migrate status
```

For development, you can also use the goose CLI directly via `make sql-migrate`.

## Architecture Guardrails

See [docs/architecture.md](docs/architecture.md) for a primer on the project's interface-driven architecture.

### Hex boundary enforcement

The project follows hexagonal architecture. Each package under `internal/` has a role — domain, driving adapter (`httpapi`), driven adapter (`memory`, `postgres`), or assembly (`config`). The dependency rule enforces an outside-in design. Adapters depend on the application layer and the domain, never on each other. The application and domain depend on nothing outside it.

These boundaries are enforced by an automated test (`hex_guardrail_test.go`) that runs as part of `make test`. The test uses `go list -json` to inspect the actual import graph of every internal package and checks each import against a set of rules.

The classifications and rules are data-driven, defined in two files:

- `testdata/hex_roles.json` — maps each package to its hex role
- `testdata/hex_rules.json` — defines which roles may import which other roles

When you add a new package under `internal/`, add an entry to `hex_roles.json` with its role. The rules apply automatically, though if a new module is introduced in `internal`, it must be classified in the JSON files. With these rules defined, if a package imports something its role doesn't allow, the test fails with a message like:

```bash
hex violation: internal/domain (domain) imports internal/metrics (application)
```

To run the boundary check in isolation:

```bash
make test-hex-guardrail
```

