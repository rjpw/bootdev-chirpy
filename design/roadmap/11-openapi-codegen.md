# OpenAPI Codegen

Replace the hand-written HTTP layer with a contract-first approach: define the API in an OpenAPI spec, generate the server interface and types, and implement the adapter as a thin bridge to domain ports.

This is the driving-adapter equivalent of what sqlc does for the driven side. SQL defines the database contract, sqlc generates the Go code, and the postgres adapter bridges to the domain. Here, OpenAPI defines the HTTP contract, oapi-codegen generates the Go code, and the httpapi adapter bridges to the domain.


## Why now

After doc 10 (scaling the repository layer), you'll have multiple entities and enough endpoints to feel the repetition: parse JSON, call a repository, map errors to status codes, serialize the response. Each handler follows the same pattern. The pattern is the boilerplate that codegen eliminates.

The hex restructure (doc 09) and test boundary enforcement already established that `httpapi/`'s job is translation, not logic. Codegen makes that explicit — the generated code handles the mechanical translation, and you write only the domain delegation.


## Prerequisites

- Doc 10 complete (multiple entities, enough endpoints to justify the investment)
- `httpapi/` tests already in `package httpapi_test` (doc 09 step 6)
- Familiarity with the current hand-written handlers (you need to understand what the generator replaces)


## What you'll build

- An OpenAPI 3.0 spec describing the existing Chirpy API
- Generated server interface, request/response types, and routing via oapi-codegen
- A thin adapter implementing the generated interface by delegating to domain ports
- Updated tests that verify the adapter logic, not the generated plumbing


## Step 1: Write the OpenAPI spec

Create `api/openapi.yaml` (at the project root, not under `internal/` — the spec is a public contract).

Start by describing the endpoints you already have. This is documentation of existing behavior, not new design:

```yaml
openapi: "3.0.0"
info:
  title: Chirpy API
  version: 0.1.0
paths:
  /api/healthz:
    get:
      operationId: getHealthz
      responses:
        "200":
          description: OK
  /api/users:
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/CreateUserRequest"
      responses:
        "201":
          description: User created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
        "409":
          description: Email already exists
components:
  schemas:
    CreateUserRequest:
      type: object
      required: [email]
      properties:
        email:
          type: string
          format: email
    User:
      type: object
      properties:
        id:
          type: string
          format: uuid
        created_at:
          type: string
          format: date-time
        email:
          type: string
```

Validation constraints (max chirp length, required fields) live in the spec. The generated code enforces them. Your handlers no longer need to.


## Step 2: Generate the server code

Install oapi-codegen and add a generate target:

```bash
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

Add to `Makefile`:

```makefile
api-generate:
	oapi-codegen --config api/codegen.yaml api/openapi.yaml
```

The config file controls what's generated:

```yaml
# api/codegen.yaml
package: httpapi
output: internal/httpapi/openapi.gen.go
generate:
  std-http-server: true
  models: true
  embedded-spec: true
```

This generates:
- Request/response structs (replacing your hand-written `parameters`, `jsonSuccess`, `jsonError`)
- A `ServerInterface` with one method per operation
- Routing that maps HTTP methods and paths to interface methods
- Request validation based on the spec


## Step 3: Implement the generated interface

Your `Server` struct implements the generated `ServerInterface`:

```go
// internal/httpapi/server.go
type Server struct {
    users domain.UserRepository
}

func (s *Server) CreateUser(w http.ResponseWriter, r *http.Request) {
    // request parsing is handled by generated code
    // you receive typed, validated input
    // call s.users.CreateUser(...)
    // map domain errors to HTTP responses
    // write response using generated response helpers
}
```

The handler shrinks to just the domain delegation and error mapping — the irreducible core that can't be generated.


## Step 4: Update tests

The generated routing and validation don't need testing — that's the generator's responsibility. Your tests focus on the adapter logic:

- Does the handler call the right repository method with the right arguments?
- Does it map `domain.ErrConflict` to HTTP 409?
- Does it map `domain.ErrNotFound` to HTTP 404?
- Does the response body match the spec's schema?

Tests still use `package httpapi_test` and speak HTTP via `httptest`. The difference is you're testing fewer things — only the things you wrote.


## Step 5: Remove hand-written boilerplate

Delete the code the generator replaced:
- `parameters`, `jsonSuccess`, `jsonError` structs
- `registerRoutes` (generated)
- Request parsing and validation logic in handlers
- Content-Type header setting (generated)

What remains in your hand-written code:
- `Server` struct with domain port fields
- One method per operation: parse generated types → call domain → map errors → respond
- `NewServer` constructor


## What changes in the development loop

The feature development loop (see [feature-development-loop.md](../feature-development-loop.md)) gains a step at the beginning:

```
  0. Add the endpoint to the OpenAPI spec
  1. make api-generate
  2. Implement the generated interface method (red → green)
  3. ... (rest of the loop unchanged)
```

The spec becomes the first artifact, before the test. The test verifies your implementation of the generated interface, not the HTTP mechanics.


## Checklist

- [ ] `api/openapi.yaml` describes all existing endpoints
- [ ] `api/codegen.yaml` configures oapi-codegen
- [ ] `make api-generate` produces `internal/httpapi/openapi.gen.go`
- [ ] `Server` implements the generated `ServerInterface`
- [ ] Hand-written request/response structs removed
- [ ] Hand-written routing removed
- [ ] Hand-written validation replaced by spec constraints
- [ ] Tests updated — no longer testing generated behavior
- [ ] `make test` passes
- [ ] README documents the codegen workflow
