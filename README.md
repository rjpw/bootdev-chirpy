# Student Boot.dev Repo: Learn HTTP Servers in Go

This repository contains my implementation of the example server defined in the [Boot.dev](https://www.boot.dev) course [Learn HTTP Servers in Go](https://www.boot.dev/courses/learn-http-servers-golang).

## Progress

This is early days. I won't be pushing commits for every lesson, just when I have important milestones.

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

# install dependencies
go mod tidy
```

### Environment

Copy the example environment file and adjust as needed:

```bash
cp .env.example .env
```

Required variables:

| Variable | Description | Example |
|----------|-------------|---------|
| `DB_URL` | Postgres connection string | `postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable` |
| `PLATFORM` | `dev` or `production` | `dev` |

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
| `make lint` | Run golangci-lint |
| `make format` | Auto-format with golangci-lint |
| `make sql-status` | Show migration status |
| `make sql-migrate` | Apply pending migrations |
| `make sql-migrate-down` | Roll back last migration |
| `make sql-generate` | Regenerate sqlc code |

## Architecture

See `design/` for roadmap docs, design decisions, and FAQs about the project's interface-driven architecture.
