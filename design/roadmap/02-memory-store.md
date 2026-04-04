# 02 — Memory Store: Proving the Interface

In this step you'll build a complete `UserStore` implementation backed by a map. Then you'll wire it into the server, test it, and — in the next doc — replace it with Postgres without changing a single handler or test.

By the end of this doc, you'll have:
- `internal/store/memory/memory.go` — a working `UserStore` backed by `map` + `sync.RWMutex`
- `internal/store/memory/memory_test.go` — the same tests you'll later write for PostgresStore
- The server running with the memory store, creating users via curl

This is disposable scaffolding. The memory store exists to prove the interface works before database complexity arrives. It will be replaced in doc 06. The interface it validates stays forever.


## Why bother

If you only ever have one implementation of an interface, the interface feels like ceremony. The moment you have two, it clicks. Building the memory store first gives you:

1. **Proof the interface is sufficient.** If the memory store can't implement `CreateUser` cleanly, the interface signature is wrong — and it's cheaper to fix now than after you've wired up sqlc and testcontainers.
2. **A test suite you can reuse.** The tests you write here test the `UserStore` contract, not the memory implementation. When you build `PostgresStore` in doc 06, you'll run the same assertions and they should pass without changes.
3. **The swap moment.** In doc 06, you'll change one line in `main.go` and watch everything keep working. That experience is worth more than any design doc.
4. **Flexibility for evolution** When this project takes off and you decide you need a clustered in-memory cache for some part of your storage implementation, you'll appreciate this decision even more.


## Step 1: Implement the memory store

Create `internal/store/memory/memory.go`.

You need:
- A struct with a `map[uuid.UUID]store.User` and a `sync.RWMutex`
- A constructor that initializes the map
- A `CreateUser` method that generates a UUID and timestamps, checks for duplicate emails, and stores the user
- The compile-time interface check: `var _ store.UserStore = (*MemoryStore)(nil)`

The shape:

```go
package memory

type MemoryStore struct {
    mu    sync.RWMutex
    users map[uuid.UUID]store.User
}

func NewMemoryStore() *MemoryStore {
    return &MemoryStore{users: make(map[uuid.UUID]store.User)}
}
```

For `CreateUser`:
- Generate `uuid.New()` and `time.Now().UTC().Truncate(time.Microsecond)` — same as PostgresStore will do later
- Check if any existing user has the same email — if so, return `store.ErrConflict`
- Store the user in the map
- Return the stored user

The email uniqueness check requires iterating the map. That's fine — this isn't a production store, it's a proof of concept. Use `mu.Lock()` / `defer mu.Unlock()` for the write path.

> **Go idiom: sync.RWMutex.** `RLock`/`RUnlock` for reads (multiple readers allowed), `Lock`/`Unlock` for writes (exclusive). For a map that's read more than written, this is the standard approach. The memory store is a good place to practice this pattern.

> **Go idiom: time.Truncate for consistency.** Truncating to `time.Microsecond` matches Postgres's timestamp precision. By doing this in the memory store too, your tests won't accidentally pass with nanosecond precision and then fail against Postgres. Consistent behavior across implementations is the whole point of the interface.


## Step 2: Write the tests

Create `internal/store/memory/memory_test.go`. Use an external test package:

```go
package memory_test
```

Write a helper:

```go
func newStore() store.UserStore {
    return memory.NewMemoryStore()
}
```

Notice the return type is `store.UserStore`, not `*memory.MemoryStore`. This is intentional — your tests should only use the interface methods. If you find yourself needing to access a field on the concrete type, that's a signal the interface is missing something.

Now write two tests:

**TestCreateUser** — create a user, verify the returned fields:
- ID is non-zero
- Email matches what you passed in
- CreatedAt and UpdatedAt are recent (within the last few seconds)

**TestCreateUserDuplicateEmail** — create a user, create another with the same email, verify you get `store.ErrConflict`:

```go
_, err = s.CreateUser(ctx, "dupe@example.com")
if !errors.Is(err, store.ErrConflict) {
    t.Errorf("expected ErrConflict, got: %v", err)
}
```

These are the same assertions you'll use in doc 06 for PostgresStore. The tests don't know or care what's behind the interface.

> **Hint:** Consider extracting the test assertions into a shared helper package (e.g., `internal/store/storetest/`) that both memory and postgres tests can call. This isn't required now, but think about whether it would reduce duplication when you have two implementations.


## Step 3: Run the tests

```bash
go test -v ./internal/store/memory/...
```

These should be near-instant — no Docker, no database, no network. This is the speed your API handler tests will always have, because they'll use a fake (or this memory store) instead of Postgres.


## Step 4: Wire it into the server

Update `internal/config/config.go` to add a `UserStore` field if it doesn't have one:

```go
type Config struct {
    Metrics *metrics.ServerMetrics
    Users   store.UserStore
}
```

Update `cmd/chirpy/main.go`:

```go
userStore := memory.NewMemoryStore()
cfg := &config.Config{
    Metrics: &metrics.ServerMetrics{},
    Users:   userStore,
}
```

Start the server. It won't do anything with users yet (no handler), but it compiles and runs — the store is wired in and ready.


## Step 5: Verify with curl (optional)

If you've already added a `POST /api/users` handler (or want to add a quick one), test it:

```bash
curl -X POST localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email": "alice@example.com"}'
```

If you haven't added the handler yet, that's fine — the wiring is proven by the fact that the server compiles with the store injected. The handler will come when the course introduces it.


## What happens next

In doc 06, you'll build `PostgresStore` — a second implementation of the same interface. Then you'll change one line in `main.go`:

```go
// before
userStore := memory.NewMemoryStore()

// after
userStore := postgres.NewPostgresStore(queries)
```

Every handler, every API test, and every piece of business logic continues to work. The interface held. That's the payoff.

The memory store stays in the repo as a reference implementation and as the backing store for fast tests. Or you delete it — it's served its purpose either way.


## Verify

- [ ] `go build ./internal/store/memory/...` compiles
- [ ] The compile-time interface check is present
- [ ] `go test -v ./internal/store/memory/...` passes in under a second
- [ ] `TestCreateUser` verifies ID, email, and timestamps
- [ ] `TestCreateUserDuplicateEmail` checks for `store.ErrConflict`
- [ ] The server compiles and starts with the memory store wired in
- [ ] Tests use `store.UserStore` as the variable type, not `*MemoryStore`


## Explore

1. **Remove the mutex.** Comment out the `Lock`/`Unlock` calls and run `go test -race ./internal/store/memory/...`. Does the race detector catch it? Write a test that calls `CreateUser` from two goroutines to trigger the detection.

2. **Add a method.** Add `GetUserByEmail` to the `UserStore` interface. The compiler will immediately tell you that `MemoryStore` no longer satisfies the interface. Implement it. This is the interface's enforcement in action — you can't forget an implementation.

3. **Shared test assertions.** If you extracted test helpers into `internal/store/storetest/`, try running them against both the memory store and (later) the Postgres store. Same assertions, different backends, same results. That's the contract.

4. **Compare to a fake.** Doc 03 introduces function-field fakes for API handler tests. How is a fake different from the memory store? (A fake returns canned data you configure per test. The memory store has real behavior — it actually stores and retrieves. Both satisfy the interface, but they serve different testing purposes.)


## Reference

- [sync.RWMutex](https://pkg.go.dev/sync#RWMutex)
- [Go race detector](https://go.dev/doc/articles/race_detector)
