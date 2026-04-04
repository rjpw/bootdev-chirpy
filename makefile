.PHONY: run lint test build

run:
	air

build:
	go build -o tmp/main ./cmd/chirpy

format:
	golangci-lint fmt ./...

lint:
	golangci-lint run ./...

test:
	go test -race ./...

sql-status:
	cd sql/schema && \
	goose postgres "postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable" status

sql-migrate:
	cd sql/schema && \
	goose postgres "postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable" up

sql-migrate-down:
	cd sql/schema && \
	goose postgres "postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable" down

sql-generate:
	sqlc generate
