.PHONY: run lint test build

run:
	air

build:
	go build -o tmp/main ./cmd/chirpy

lint:
	golangci-lint run ./...

test:
	go test -race ./...