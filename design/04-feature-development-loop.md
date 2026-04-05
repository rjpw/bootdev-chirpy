# Feature Development Loop

This doc describes how to add a feature to Chirpy. It's behavior-first: you start from what the user should experience, not from what the database needs. The database is the last thing you touch.


## The loop

Every feature follows the same cycle:

```
  Requirement → API test (red) → Interface → Memory store (green) → Refactor
                                                                       │
                                                          feature works, no DB
                                                                       │
                                                                       ▼
                                              Migration → sqlc → Postgres store → Integration test
```

The top row is fast. No Docker, no database, no waiting. You stay here until the behavior is right. The bottom row is mechanical — translating a proven design into SQL.


## Step 1: Start with the behavior

You have a requirement: "Users can post chirps. A chirp has a body (max 140 chars) and belongs to a user."

Don't open pgAdmin. Don't think about table schemas. Ask: what does the API look like?

```
POST /api/chirps
Request:  { "body": "hello world" }
Response: { "id": "...", "body": "hello world", "author_id": "...", "created_at": "..." }
Status:   201 Created
```

What can go wrong?

```
Empty body        → 400
Body > 140 chars  → 400
User not found    → 401 (later, when auth exists)
```

You now know enough to write a test.


## Step 2: Write the API test (red)

```go
func TestCreateChirp(t *testing.T) {
    srv := newTestServer("dev")
    // ... create a user first, get their context ...

    body := `{"body": "hello world"}`
    r := httptest.NewRequest("POST", "/api/chirps", strings.NewReader(body))
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, r)

    if w.Code != http.StatusCreated {
        t.Errorf("want 201, got %d", w.Code)
    }
}
```

This won't compile. There's no chirp handler, no chirp store, no chirp type. That's the point — the test tells you what to build.


## Step 3: Define the interface

The test needs a store that can create chirps. What's the minimum interface?

```go
type ChirpStore interface {
    CreateChirp(ctx context.Context, body string, authorID uuid.UUID) (*Chirp, error)
}
```

Notice what's not here:
- No `InsertChirpParams` struct — the caller provides `body` and `authorID`, not a column set
- No `created_at` parameter — the store decides timestamps, not the handler
- No database vocabulary — `Create`, not `Insert`

The interface should make sense to someone who's never seen the database. If you described it to a product manager, they'd nod. That's the test.

Define the domain type the same way:

```go
type Chirp struct {
    ID        uuid.UUID `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    Body      string    `json:"body"`
    AuthorID  uuid.UUID `json:"author_id"`
}
```

This is what the application thinks about. It might not match the database columns exactly — and that's fine.


## Step 4: Implement the memory store (green)

Make the test pass with the simplest thing that works:

```go
func (s *Store) CreateChirp(_ context.Context, body string, authorID uuid.UUID) (*store.Chirp, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    chirp := store.Chirp{
        ID:        uuid.New(),
        CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
        Body:      body,
        AuthorID:  authorID,
    }
    s.chirps[chirp.ID] = chirp
    return &chirp, nil
}
```

Write the handler. Wire it up. Run the test. Green.

You now have a working feature with no database. The handler works. The validation works. The error cases work. You can iterate on the API design, add edge cases, refactor the handler — all in the fast loop.


## Step 5: Refactor

Now that it's green, look at what you've built:

- Is the interface right? Does it have the methods the handler actually calls, and nothing more?
- Is the domain type right? Does it have the fields the handler actually uses?
- Are the error cases covered? Does the handler return the right status codes?
- Is the memory store a natural implementation of the interface, or is it fighting the design?

That last question is the canary. If the memory store feels awkward — if you're faking pagination, or simulating joins, or working around parameters that only make sense for SQL — the interface is leaking database concerns. Fix the interface now, while it's cheap.


## Step 6: Add the database layer

The behavior is proven. Now make it durable.

1. `make sql-create` — write the migration
2. Write queries in `internal/schema/queries/chirps.sql`
3. `make sql-generate` — sqlc generates the Go code
4. Implement `postgres/chirps.go` — the `toStoreChirp` mapper and the `ChirpStore` methods
5. Add the compile-time check: `var _ store.ChirpStore = (*Store)(nil)`
6. Wire into `Config` and `main.go`

This is mechanical. The interface already exists. The domain type already exists. You're just writing the translation layer between sqlc's generated types and your domain types.


## Step 7: Integration tests (targeted)

You don't need to re-test every behavior against Postgres. The API tests already proved the behavior works. Integration tests verify the translation:

- Does `mapError` correctly turn a unique violation into `ErrConflict`?
- Does `toStoreChirp` map all the fields?
- Does a multi-query store method (e.g., "look up user, then create chirp") work with committed data?

If the store method is a trivial pass-through — one sqlc call, one mapper — you might not need a dedicated integration test at all. The API tests exercise the same code path through the memory store, and the mapper is obviously correct by inspection.

Write integration tests for the interesting parts, not for ceremony.


## Why this order matters

The traditional order is database-first: design the schema, generate the code, build up to the handler. This works, but it has costs:

- You make schema decisions before you understand the behavior
- You write SQL for columns the handler might not need
- You discover interface design problems late, after the migration is written
- Every iteration requires Docker and real Postgres

The behavior-first order inverts this:

- The handler tells you what the store needs
- The store interface tells you what the database needs
- The memory store proves the interface is implementable
- The database layer is the last step, informed by everything above

You spend most of your time in the fast loop (no Docker, sub-second tests) and only drop into the slow loop (Postgres, integration tests) when the design is settled.


## The memory store is not a toy

It's tempting to think of the memory store as scaffolding you'll throw away. It's not. It serves three ongoing purposes:

1. **Fast API tests** — every `newTestServer` uses it. These tests run in milliseconds and cover all your HTTP behavior. They'll always be faster than hitting Postgres.

2. **Interface design feedback** — if a new interface method is awkward to implement in memory, the interface is wrong. The memory store is your first customer for the interface, and it's the most honest one.

3. **Isolation** — API tests with a memory store have zero shared state between test cases. No snapshot/restore, no transaction rollback, no cleanup. Each test gets a fresh store. This makes tests independent and parallelizable.

The postgres store is the production implementation. The memory store is the development implementation. Both are first-class.
