# Testing at Hexagonal Boundaries

How to write tests that respect and enforce the hexagonal architecture. This doc explains the mental model, identifies where the current tests violate it, and defines the refactoring needed.


## The rule

Each layer tests its own translation. Nothing more.

```
┌──────────────────────────────────────────────────────────────┐
│  httpapi/ tests                                              │
│                                                              │
│  Input:  HTTP request (method, path, headers, JSON body)     │
│  Output: HTTP response (status code, headers, JSON body)     │
│                                                              │
│  Tests verify: "Does this adapter correctly translate        │
│  between HTTP and domain operations?"                        │
│                                                              │
│  Does NOT verify: domain behavior, repository correctness    │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│  memory/ and postgres/ tests                                 │
│                                                              │
│  Input:  domain method call (CreateUser, GetUserByEmail)     │
│  Output: domain types and errors (*domain.User, ErrConflict) │
│                                                              │
│  Tests verify: "Does this adapter correctly implement        │
│  the domain port?"                                           │
│                                                              │
│  Does NOT verify: HTTP behavior, JSON serialization          │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│  domain/ tests (when domain logic exists)                    │
│                                                              │
│  Input:  domain types and values                             │
│  Output: domain types and values                             │
│                                                              │
│  Tests verify: business rules, validation, computation       │
│                                                              │
│  Does NOT verify: HTTP, persistence, or any adapter          │
└──────────────────────────────────────────────────────────────┘
```

If a test in `httpapi/` calls a domain method directly, it's testing the wrong layer. If a test in `memory/` constructs an HTTP request, it's testing the wrong layer. The boundary is the test's boundary too.


## Why this matters

Tests that cross boundaries create invisible coupling. When a test in the HTTP adapter calls `s.CreateUser(ctx, email)` and checks the returned `*domain.User`, it's testing the repository — but it lives in the HTTP package. This has consequences:

1. **The test doesn't verify what the adapter does.** The HTTP adapter's job is to parse JSON, call the repository, map errors to status codes, and serialize the response. A test that bypasses HTTP tests none of that.

2. **The test can't move to an external test package.** It needs access to unexported methods, which means `package httpapi` instead of `package httpapi_test`. This hides the fact that the public API is leaking domain concerns.

3. **The adapter grows domain methods.** If the test calls `Server.CreateUser`, that method must be exported. Now `Server` has a public API that looks like a repository. When you add chirps, it grows `CreateChirp`, `ListChirps`, `DeleteChirp`. The HTTP adapter becomes a god object.

4. **Refactoring becomes expensive.** If you change how the handler maps errors to status codes, the test that bypasses HTTP won't catch the regression. If you change the repository interface, tests in the HTTP package break — even though the HTTP contract didn't change.

The hex architecture's value is in its constraints. Tests that ignore the boundaries erode those constraints silently.


## What's wrong in the current tests

### `users_test.go` — tests the repository through the HTTP adapter

```go
func TestCreateUser(t *testing.T) {
    s := newTestServer("dev")
    user, err := s.CreateUser(ctx, email)  // ← domain call, not HTTP
    // asserts on *domain.User and domain.ErrConflict
}
```

`Server.CreateUser` is an exported method that delegates to the repository. The test is a repository test wearing an HTTP test's clothes.

**Fix:** Delete `Server.CreateUser`. The handler calls `s.cfg.Users.CreateUser` directly. Rewrite the test to speak HTTP:

```go
func TestCreateUser(t *testing.T) {
    srv := newTestServer("dev")
    body := `{"email": "test@example.com"}`
    r := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, r)

    if w.Code != http.StatusCreated {
        t.Errorf("want 201, got %d", w.Code)
    }
    // optionally: decode w.Body and check the email field
}

func TestCreateUserConflict(t *testing.T) {
    srv := newTestServer("dev")
    body := `{"email": "test@example.com"}`

    // first request succeeds
    r := httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, r)

    // second request conflicts
    r = httptest.NewRequest("POST", "/api/users", strings.NewReader(body))
    w = httptest.NewRecorder()
    srv.ServeHTTP(w, r)

    if w.Code != http.StatusConflict {
        t.Errorf("want 409, got %d", w.Code)
    }
}
```

Now the test verifies the HTTP adapter's actual job: JSON parsing, status code mapping, response serialization.

The repository behavior (creating users, detecting conflicts) is already tested in `memory/memory_test.go` and `postgres/users_test.go`. The HTTP test doesn't need to re-verify it — it trusts the port contract and tests the translation.


### `chirp_test.go` — uses unexported `parameters` struct

```go
params := parameters{Body: tc.body}
payload, err := json.Marshal(params)
```

The test constructs the request body using the handler's internal struct. An external test package can't access `parameters`.

**Fix:** Use a raw JSON string or a local test struct:

```go
payload := fmt.Sprintf(`{"body": %q}`, tc.body)
```

Or define a struct local to the test file — the test doesn't need to share the handler's internal representation. In fact, using a different struct is *better*: it verifies that the JSON contract works regardless of the handler's internal types.


### `metrics_test.go` — reaches into unexported `cfg` field

```go
if srv.cfg.Metrics.FileserverHits() != 0 {
    t.Errorf(...)
}
```

The test accesses the server's private config to inspect metrics state directly.

**Fix:** Test through the HTTP endpoints. The metrics are already exposed via `GET /admin/metrics` and reset via `POST /admin/reset`. The test should:

1. Hit some endpoints to generate traffic
2. `GET /admin/metrics` and check the response body
3. `POST /admin/reset`
4. `GET /admin/metrics` and verify the count is zero

This tests the full HTTP contract: the middleware counts hits, the metrics endpoint reports them, the reset endpoint clears them. No internal access needed.


## The external test package as an architectural guardrail

Once these fixes are in place, every test file in `httpapi/` can use `package httpapi_test`. This is the guardrail:

- If a test compiles in `package httpapi_test`, it can only use exported symbols.
- The only exported symbols on `Server` should be `NewServer` and `ServeHTTP`.
- Therefore, every test must go through HTTP. The architecture is enforced by the compiler.

If you later add a method to `Server` and find you need to export it for a test, that's a signal: either the method belongs somewhere else (domain? service?), or the test is crossing a boundary.

The convention:

| Package | Test package | What it tests |
|---------|-------------|---------------|
| `httpapi` | `httpapi_test` | HTTP in → HTTP out |
| `memory` | `memory_test` | Domain call → domain result |
| `postgres` | `postgres_test` | Domain call → domain result (real DB) |
| `domain` | `domain_test` | Business rules (when they exist) |


## Where do repository contract tests live?

The tests currently in `users_test.go` (create a user, check the result; create a duplicate, check for conflict) are valid tests — they're just in the wrong package. They test the `UserRepository` contract.

These belong in `memory/memory_test.go` (which already has them) and `postgres/users_test.go` (which already has them). The HTTP layer doesn't need its own copy.

If you later want to verify that *all* repository implementations satisfy the same contract, you can write a shared test function:

```go
// internal/domain/repositorytest/user.go
func TestUserRepository(t *testing.T, repo domain.UserRepository) {
    // create user, check fields
    // create duplicate, check ErrConflict
    // get by email, check result
}
```

Then call it from both `memory/` and `postgres/` tests. This is the "contract test" pattern — one test suite, multiple implementations. But that's a future refinement, not a prerequisite.


## Checklist

- [ ] Delete `Server.CreateUser` — inline `s.cfg.Users.CreateUser` in the handler
- [ ] Rewrite `users_test.go` to test through HTTP (POST, check status code and body)
- [ ] Rewrite `chirp_test.go` to use raw JSON strings instead of `parameters` struct
- [ ] Rewrite `metrics_test.go` to test through HTTP endpoints instead of `s.cfg`
- [ ] Move all test files to `package httpapi_test`
- [ ] Verify: `Server`'s exported API is only `NewServer` and `ServeHTTP`
- [ ] `make test` passes
