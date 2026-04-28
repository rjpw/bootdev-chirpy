## The Goal

> **Note:** This doc was written at the start of the project and uses the original package names (`store`, `UserStore`, `api`). The project has since been restructured (doc 09) to use hexagonal/DDD conventions: `domain/` for the core, `UserRepository` for interfaces, `httpapi/` for the HTTP adapter, and `postgres/`/`memory/` as top-level adapter packages. See [feature-development-loop.md](feature-development-loop.md) and [roadmap/10-scaling-the-store-layer.md](roadmap/10-scaling-the-store-layer.md) for the current patterns.

Build the course project in a way that each concern — routing, business logic, persistence, auth — lives in its own package with clear interfaces, so that when the course asks you to add a feature, you're adding it to the right layer, not piling it into main.go.


## 1. Project Layout

Adopt the standard Go project layout now, before you have anything to move:

```
bootdev-chirpy/
├── cmd/
│   └── chirpy/
│       └── main.go          ← thin: wire dependencies, start server
├── internal/
│   ├── api/                 ← HTTP handlers, middleware, routing
│   ├── chirp/               ← business logic (the "chirp" domain)
│   ├── user/                ← business logic (the "user" domain)
│   ├── auth/                ← JWT, password hashing, token logic
│   └── store/               ← persistence interfaces + implementations
│       ├── memory/          ← in-memory impl (use this first)
│       └── postgres/        ← postgres impl (swap in later)
├── root/                    ← static assets (keep as-is)
├── go.mod
└── go.sum
```

Key decisions here:

- internal/ prevents external packages from importing your code — appropriate for an app, not a library
- store/memory and store/postgres both implement the same interface, so swapping them is a one-line change in main.go
- Domain packages (chirp/, user/) contain no HTTP or DB knowledge — just types and logic


## 2. The Store Interface Pattern

This is the most important architectural move. Define interfaces before you write implementations:

```go
// internal/store/store.go
package store

type ChirpStore interface {
    CreateChirp(ctx context.Context, body string, userID uuid.UUID) (Chirp, error)
    GetChirp(ctx context.Context, id uuid.UUID) (Chirp, error)
    ListChirps(ctx context.Context) ([]Chirp, error)
    DeleteChirp(ctx context.Context, id uuid.UUID) error
}

type UserStore interface {
    CreateUser(ctx context.Context, email, hashedPassword string) (User, error)
    GetUserByEmail(ctx context.Context, email string) (User, error)
}
```

Your handlers receive these interfaces. When the course introduces Postgres, you implement the same interface against a real DB and swap it in main.go. Nothing else changes.

## 3. Dependency Injection in main.go

`main.go` is the composition root — the only place that knows which concrete 
implementations are in use:

```go
// cmd/chirpy/main.go
func main() {
    db := store.NewMemoryStore()   // later: store.NewPostgresStore(connStr)
    
    srv := api.NewServer(api.Config{
        ChirpStore: db,
        UserStore:  db,
        // AuthSecret: os.Getenv("JWT_SECRET"),
    })
    srv.Run(":8080")
}
```

## 4. Tooling

Add these now so they become habits:

Linting — golangci-lint
```bash
# install
go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest
```
# create .golangci.yml at project root


Minimal .golangci.yml to start:
```yaml
linters:
  enable:
    - errcheck      # catch ignored errors
    - staticcheck   # broad static analysis
    - govet         # catches common mistakes
    - gofmt         # formatting
    - revive        # style
```

Testing
```bash
go test ./...
```
No external test framework needed. Use testing from stdlib. Write tests alongside each package as you build it — internal/chirp/chirp_test.go, etc.

Makefile — a single place for all dev commands:
```makefile
.PHONY: run lint test build

run:
        air

build:
        go build -o tmp/main ./cmd/chirpy

lint:
        golangci-lint run ./...

test:
        go test -race ./...
```

sqlc (for when Postgres arrives)
When the course introduces SQL, use sqlc instead of writing boilerplate scan code by hand. You write SQL, it generates type-safe Go. Add it to the Makefile when the time comes.


## 5. Error Handling Convention

Pick one pattern now and stick to it. For an HTTP server, sentinel errors work well:

```go
// internal/store/errors.go
var (
    ErrNotFound   = errors.New("not found")
    ErrConflict   = errors.New("conflict")
    ErrForbidden  = errors.New("forbidden")
)
```

Handlers check for these and map them to HTTP status codes. Business logic returns them. Neither layer knows about the other's concerns.

## 6. Immediate Next Steps

In order:

1. Move main.go → cmd/chirpy/main.go, update go.mod if needed
2. Create internal/api/server.go — move the mux and server setup there
3. Create internal/store/store.go — define the interfaces (even if empty)
4. Create internal/store/memory/memory.go — stub implementation
5. Add .golangci.yml and run golangci-lint run ./... — fix what it finds
6. Add Makefile

That's it for the scaffold. Everything the course asks you to add from here has a home before you write a line of it.

The payoff: when the course says "add a Postgres store," you create `internal/store/postgres/`, implement the interface, and change one line in main.go. When it says "add JWT auth," it goes in internal/auth/ and gets injected into the handlers that need it. The course's monolith becomes your layered system, built incrementally.