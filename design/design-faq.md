> Please explain why you recommend interfaces for storage.

A few reasons, each one practical:

Testability is the biggest one. When your handlers depend on an interface rather than a concrete type, you can pass a fake implementation in tests — one that returns canned data or simulates errors — without touching a database. Tests run fast, run anywhere, and don't need a Postgres instance standing by.

The swap cost drops to near zero. You said it yourself: the course starts in-memory and moves to Postgres. If your handlers call methods on a concrete memory repository, switching to Postgres means touching every handler. If they call methods on a ChirpRepository interface, you write the Postgres implementation once and change one line in main.go. The handlers never know anything changed.

It enforces the boundary you actually want. An interface is a contract: "I need something that can do these operations." It prevents persistence details from leaking into business logic. If you find yourself wanting to call a Postgres-specific method from a handler, the interface stops you — which is the right outcome.

It documents intent. Reading domain.ChirpRepository tells you exactly what the application needs from persistence, nothing more. That's more useful than reading a struct with 40 methods and guessing which ones matter.

The cost is low — you're writing method signatures you'd write anyway — and the benefit compounds as the project grows. It's one of those Go patterns that feels like overhead until the first time you need to swap an implementation or write a test without a real DB, and then it feels obvious.

> explain how the repository interface survives the use of sqlc and goose

Good question, because sqlc in particular generates its own interface-like layer, and it can look like a collision.

__What sqlc generates__

sqlc reads your SQL queries and produces a Queries struct with one method per query — CreateChirp, GetChirp, etc. It also generates a Querier interface for that struct. So you end up with generated code that already looks like what you'd write by hand.

__The relationship__

Your domain.ChirpRepository interface is your application's contract. The sqlc-generated Queries struct is an implementation detail. The Postgres repository wraps it:

```go
// internal/postgres/chirps.go
type Repository struct {
    queries *database.Queries  // sqlc-generated
}

func (r *Repository) CreateChirp(ctx context.Context, body string, userID uuid.UUID) (*domain.Chirp, error) {
    row, err := r.queries.CreateChirp(ctx, database.CreateChirpParams{
        Body:   body,
        UserID: userID,
    })
    // map database.Chirp → domain.Chirp, return
}
```

Repository satisfies domain.ChirpRepository. The rest of your app never sees sqlc types.

__Why not just use sqlc's Querier directly?__

Two reasons:

1. sqlc's Querier is shaped by your SQL, not by your application's needs. It has one method per query, including queries your handlers should never call directly. Your interface expresses intent; sqlc's expresses database operations.
2. If you ever need to combine multiple sqlc calls into one logical operation (a transaction, or a "create user and return their chirps" call), that logic lives in your repository implementation, not scattered across handlers.

__Goose is simpler__

Goose manages schema migrations — it has no opinion about your Go types at all. You run goose up before starting the server, and that's the entire relationship. It doesn't touch your interfaces.

__The mental model__

```
your app (handlers, business logic)
    ↓ depends on
domain.ChirpRepository interface
    ↓ implemented by
postgres.Repository struct
    ↓ uses internally
sqlc Queries struct
    ↓ runs against
Postgres (schema managed by Goose)
```

Each layer only knows about the layer directly below it. sqlc and Goose are 
implementation details of the bottom two layers. Your interface sits above all of it and stays stable regardless of what changes underneath.
