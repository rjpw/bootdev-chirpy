# API Lifecycle: Versioning, Routing, and Sunsetting

Every endpoint is a contract. This doc covers how contracts evolve — how to introduce new versions, run them alongside old ones, signal deprecation, and eventually retire them.

This is a domain concern, not an infrastructure concern. It applies whether you're on a single server or a hundred Kubernetes pods. For the infrastructure side of running multiple versions simultaneously, see [roadmap/devops/04-always-on-deployment.md](roadmap/devops/04-always-on-deployment.md).


## When to version

Not every change needs a new version. The decision depends on whether existing clients will break.

### Additive changes (no new version needed)

- Add a new field to a response body (clients should ignore unknown fields)
- Add a new endpoint
- Add a new optional query parameter
- Add a new error code to an existing error response
- Relax a validation rule (accept a wider range of inputs)

These are backward-compatible. Existing clients continue to work. Document the change in release notes, but don't bump the API version.

### Breaking changes (new version required)

- Remove a field from a response body
- Rename a field
- Change a field's type (string → integer, flat → nested)
- Change the response envelope structure
- Remove an endpoint
- Tighten a validation rule (reject inputs that were previously accepted)
- Change the meaning of a status code for a given endpoint

These break existing clients. They require a new API version.

### The gray area

- Adding a required field to a request body — breaking for clients that don't send it
- Changing pagination style (offset → cursor) — breaking for clients that use the old style
- Changing the default sort order — technically breaking, often unnoticed

When in doubt, version it. The cost of an unnecessary version is low (a bit of routing complexity). The cost of breaking clients silently is high (lost trust, support burden, emergency patches).


## Versioning strategy

Three common approaches, with tradeoffs:

### URL path versioning

```
/api/v1/chirps
/api/v2/chirps
```

**Pros:** Visible, explicit, easy to route, easy to test, easy to log and monitor separately. Clients know exactly which version they're using.

**Cons:** URL changes propagate through client code, documentation, and tooling. Can feel heavy for small changes.

**Best for:** Public APIs, B2B integrations, any context where clients are external and contracts are formal.

### Header versioning

```
GET /api/chirps
Accept: application/vnd.chirpy.v2+json
```

Or a custom header:

```
GET /api/chirps
X-API-Version: 2
```

**Pros:** URLs stay stable. Versioning is metadata, not structure.

**Cons:** Invisible in browser, logs, and monitoring unless you extract the header. Harder to test casually (curl needs explicit headers). Clients can forget to set the header.

**Best for:** Internal APIs where clients are controlled and sophisticated.

### Additive-only (no explicit versioning)

Never break the contract. Only add fields, endpoints, and capabilities. Old fields are never removed, only deprecated.

**Pros:** No version management. No routing complexity. Clients never break.

**Cons:** The API accumulates cruft. Deprecated fields live forever. Response payloads grow. Internal complexity increases as the code must support every field ever shipped.

**Best for:** APIs with very long-lived clients that can't be updated (embedded devices, mobile apps with slow update cycles).

### Recommendation for this project

URL path versioning. It's the most explicit, the easiest to reason about, and the most common in B2B contexts. Start with `/api/v1/` and don't think about v2 until you need it.


## Routing multiple versions

When v2 exists alongside v1, both must work simultaneously. How you route them depends on where the versions diverge.

### Same binary, different handlers

The simplest approach. Both versions are registered in the same router:

```go
func (s *Server) registerRoutes() {
    // v1
    s.mux.HandleFunc("GET /api/v1/chirps", s.handleListChirpsV1)
    s.mux.HandleFunc("POST /api/v1/chirps", s.handleCreateChirpV1)

    // v2 — different response shape
    s.mux.HandleFunc("GET /api/v2/chirps", s.handleListChirpsV2)
    s.mux.HandleFunc("POST /api/v2/chirps", s.handleCreateChirpV2)
}
```

The v1 and v2 handlers can share the same store interface. The difference is in request parsing and response serialization — the domain operation is the same, the wire format differs.

This works well when:
- The versions differ only in response shape or validation rules
- The number of versioned endpoints is small
- Both versions use the same underlying data

### Shared logic, version-specific serialization

Factor the domain logic out of the handler. The handler becomes a thin adapter:

```go
func (s *Server) handleListChirpsV1(w http.ResponseWriter, r *http.Request) {
    chirps, err := s.listChirps(r.Context(), r)
    if err != nil { ... }
    s.respondWithJSON(w, http.StatusOK, toV1ChirpList(chirps))
}

func (s *Server) handleListChirpsV2(w http.ResponseWriter, r *http.Request) {
    chirps, err := s.listChirps(r.Context(), r)
    if err != nil { ... }
    s.respondWithJSON(w, http.StatusOK, toV2ChirpList(chirps))
}
```

The `toV1ChirpList` and `toV2ChirpList` functions are the only code that knows about the wire format difference. The domain logic (`listChirps`) is shared.

### Separate services

When versions diverge significantly — different data models, different stores, different business logic — they may warrant separate services. This is rare and expensive. Exhaust the simpler options first.


## Deprecation

Deprecation is a signal, not an action. It tells clients: this version still works, but it won't forever. Start planning your migration.

### The `Sunset` header

RFC 8594 defines a standard header for signaling deprecation:

```
Sunset: Sat, 01 Nov 2026 00:00:00 GMT
```

Include it in every response from a deprecated version. Clients that check for it can alert their developers. Clients that don't check still work — until the sunset date.

### The `Deprecation` header

A draft standard (not yet RFC) that signals when the deprecation was announced:

```
Deprecation: Mon, 01 Jun 2026 00:00:00 GMT
```

Together with `Sunset`, this tells the client: "We deprecated this on June 1, and it will stop working on November 1. You have 5 months."

### Documentation

Update API docs to mark deprecated endpoints visibly. Include:
- What's deprecated
- What replaces it
- When it will be removed
- A migration guide (what clients need to change)

### Telemetry

Track usage of deprecated endpoints. This tells you:
- How many clients are still using v1
- Whether your sunset timeline is realistic
- Which clients need direct outreach before you remove the version


## Sunsetting

Sunsetting is the action: removing a deprecated version so it no longer works.

### Timeline

How long to support a deprecated version depends on:

- **Customer contracts.** If your SLA guarantees 12 months of support for any API version, that's your minimum. This is common in B2B and financial services.
- **Client update cycles.** Mobile apps with slow review processes need more time than internal services you control. Embedded devices might need years.
- **Usage telemetry.** If zero clients have called v1 in 30 days, you can sunset sooner. If 40% of traffic is still v1, your timeline is too aggressive.
- **Operational cost.** Every supported version is code you maintain, test, and monitor. The cost of keeping v1 alive is real — factor it into the timeline.

A reasonable default for a B2B API: announce deprecation at least 6 months before sunset. Communicate directly with known consumers. Monitor usage. Extend the timeline if adoption of the new version is slow.

### What happens at sunset

On the sunset date, deprecated endpoints return `410 Gone`:

```
HTTP/1.1 410 Gone
Content-Type: application/json

{"error": "This API version has been retired. Use /api/v2/chirps instead."}
```

Don't just delete the route and return 404. A 410 is explicit — it tells the client the resource existed but was intentionally removed. Include a pointer to the replacement.

Keep the 410 response in place for a reasonable period after sunset. Clients that missed the deprecation window will get a clear signal instead of a confusing 404.

### Removing the code

After the 410 has been live long enough that no clients are hitting it (telemetry confirms), remove the v1 handlers, the v1 serialization functions, and any v1-specific tests. This is a cleanup commit, not a feature — it should be its own PR with a clear description of what was removed and why.


## API versions and schema versions are coupled

A v2 endpoint might need a database column that v1 doesn't. This creates a dependency between API versioning and schema migration:

```
v1.2.0: Migration adds `chirps.edited_at` column (nullable, backward-compatible)
        v1 handlers ignore it
        v2 handlers return it in the response

v1.3.0: Migration drops the old `chirps.body_html` column (v1 used it, v2 doesn't)
        v1 is past its sunset date
        Only v2 handlers remain
```

This is the expand-contract pattern from [devops/04-always-on-deployment.md](roadmap/devops/04-always-on-deployment.md), applied at the API level:

- **Expand**: add the new column, ship v2 handlers that use it, keep v1 handlers working
- **Transition**: deprecate v1, monitor usage, wait for clients to migrate
- **Contract**: sunset v1, remove v1 handlers, drop columns only v1 needed

The schema migration timeline and the API sunset timeline must be coordinated. You can't drop a column that a still-supported API version reads from. The column lives as long as the oldest supported version that uses it.

This is where telemetry, deprecation signals, and customer communication all converge. The technical decision (when to drop the column) is gated by the business decision (when to sunset the API version), which is gated by the customer reality (when they've actually migrated).


## Checklist for a new API version

- [ ] What's breaking in the current version that requires a new one?
- [ ] Can the change be made additively instead? (add a field, don't remove one)
- [ ] What's the migration path for existing clients? (document it)
- [ ] Are both versions sharing domain logic, or do they diverge?
- [ ] What schema changes does the new version need? Are they backward-compatible with the old version?
- [ ] What's the deprecation timeline for the old version?
- [ ] How will you track usage of the old version? (telemetry)
- [ ] Who needs to be notified? (direct outreach to known consumers)
