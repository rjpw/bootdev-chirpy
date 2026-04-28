-include .env

.PHONY: run build format lint test test-db test-integration start-db stop-db sql-create sql-fix sql-status sql-migrate sql-migrate-down sql-generate

run:
	air

start-db:
	docker compose up -d

stop-db:
	docker compose down

build:
	go build -o tmp/main ./cmd/chirpy

format:
	golangci-lint fmt ./...

lint:
	golangci-lint run ./...

test:
	go test -race ./...

test-db:
	go test -race -tags integration -count=1 ./internal/postgres/database/... ./internal/postgres/testdb/...

test-integration:
	go test -race -tags integration -count=1 ./...

sql-create:
	@read -p "Migration name: " name; \
	goose create $$name sql

sql-fix:
	goose fix

sql-status:
	goose status

sql-migrate:
	goose up

sql-migrate-down:
	goose down

sql-generate:
	sqlc generate
