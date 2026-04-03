# Student Boot.dev Repo: Learn HTTP Servers in Go

This repository contains my implementation of the example server defined in the [Boot.dev](https://www.boot.dev) course [Learn HTTP Servers in Go](https://www.boot.dev/courses/learn-http-servers-golang).

## Progress

This is early days. I won't be pushing commits for every lesson, just when I have important milestones.

## Quickstart for Developers

I use Air while working on features. This provides hot reload of the back end, and saves you from having to stop and start the server all the time.

I'm also using `make` to orchestrate static analysis and testing, with [golangci-lint](https://golangci-lint.run/) to keep me honest.


```bash
# install golangci-lint (optional but a good idea)
curl -sSfL https://golangci-lint.run/install.sh | \
  sh -s -- -b $(go env GOPATH)/bin v2.11.4

# configure air (with any changes you see fit to make)
cp .air-example.toml .air.toml

# run in development mode
make run

# in another terminal, check health endpoint
curl localhost:8080/api/healthz
```

### Database

Run a PostgreSQL server in docker:

```bash
# heredoc to create the docker compose file
cat << 'eof' > docker-compose.yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
eof

# launch the postgres server
docker compose up -d
```

Create your `chirpy` database:

```bash
# provide password by environment variable
export PGPASSWORD=postgres

# access the Postgres shell in your container
psql "postgres://postgres:@localhost:5432"

# at prompt `postgres=#`
CREATE DATABASE chirpy;

# connect to gator
\c chirpy

# at prompt `chirpy=#`
ALTER USER postgres PASSWORD 'postgres';

# back in bash, install dependencies
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

go get github.com/google/uuid
go get github.com/lib/pq
go get github.com/joho/godotenv
go mod tidy

# bring up the database schema
make sql-migrate

# optionally (re)generate the go database code
make sql-generate
```

