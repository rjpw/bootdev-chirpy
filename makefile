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