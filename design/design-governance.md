# Design Governance: Domain vs System

Every feature in this project lives in the tension between two forces. The domain pushes down from the top: what do users need? what does the business require? what should the API look like? The system pushes up from the bottom: how does the database enforce integrity? how does the network impose latency? how does storage constrain what's queryable?

Neither force is optional. Ignoring the domain produces software that's correct but unusable. Ignoring the system produces software that's elegant but broken under load. This doc establishes the principles for navigating between them.


## What the system does: functional requirements

A functional requirement (FR) describes what the system does from the user's perspective:

- Users can post chirps (max 140 characters)
- Users can see a feed of chirps from people they follow
- Deleting an account removes the user's chirps

FRs are expressed in domain language. They don't mention tables, columns, indexes, or transactions. They describe behavior that a product manager would recognize.

When designing a feature, start here. The API endpoint, the request/response shape, the error cases — these all flow from the FR. The store interface flows from the endpoint. The database schema flows from the store interface. This is the top-down direction, and it should be the default.


## How well it does it: non-functional requirements

A non-functional requirement (NFR) describes how the system behaves under real-world constraints:

- The chirp feed must load in under 200ms for 10,000 chirps
- Deleting a user must not leave orphaned data
- The API must support pagination for any list endpoint
- Concurrent writes to the same resource must not corrupt data

NFRs are where the system pushes back on the domain. The domain says "show me all chirps." The system says "there are 50,000 of them and the client will time out." Pagination is the resolution — a systemic imposition on the domain that users didn't ask for but can't avoid.

NFRs are not afterthoughts. They should be identified alongside FRs, because they constrain the design in ways that are expensive to retrofit:

- Pagination affects the store interface signature, the API response shape, and the client contract
- Referential integrity affects the migration DDL, the error mapping, and the delete behavior
- Concurrency affects whether the store uses transactions, optimistic locking, or idempotency keys

Discovering an NFR late — after the endpoint is shipped and clients depend on it — turns a design decision into a breaking change.


## The API is the contract

Endpoints are where domain meets system, and where design crystallizes into commitment. Once a client depends on a response shape, changing it is a negotiation with every consumer.

Design decisions to make explicitly for every endpoint:

**Response shape.** Does a chirp include its author inline, or just an `author_id`? Nesting is convenient for clients but couples the resources. A flat ID is flexible but requires an extra request. This choice is permanent — pick it deliberately, not by accident.

**List behavior.** Any endpoint that returns a collection will eventually need pagination. Decide the pagination style (cursor vs offset) early, even if the first version returns everything. Adding pagination later changes the response envelope, which breaks clients.

**Error contract.** What status codes and error shapes does the endpoint return? `400` for validation, `404` for not found, `409` for conflict — these are part of the contract. Document them in tests.

**Idempotency.** Can the client safely retry a `POST`? If not, they need to handle duplicates. If so, you need an idempotency mechanism. This affects the store interface (does `CreateChirp` check for duplicates?) and the database (unique constraints on natural keys).


## The store interface is the negotiation point

The store interface sits between the domain (above) and the system (below). It should feel natural from both directions:

- **From above**: the handler calls methods that read like domain operations. `CreateChirp`, `ListChirpsByAuthor`, `DeleteAccount`. No SQL vocabulary, no column sets, no pagination internals leaking through.
- **From below**: the postgres implementation can satisfy the interface without contortions. If the interface demands something the database can't efficiently provide, the interface needs to negotiate — perhaps accepting a cursor parameter, or returning a result set with a "has more" flag.

When the two directions conflict, lean toward the domain. The database can usually be made to serve the interface. The reverse — reshaping the API to match the database — produces APIs that feel like SQL queries with HTTP wrapping.

But don't ignore the system entirely. An interface method like `GetAllChirps() []Chirp` is domain-pure but system-hostile. The negotiated version — `ListChirps(ctx, cursor) ([]Chirp, NextCursor, error)` — acknowledges the system's constraints while keeping the domain language.


## The memory store reveals design problems

The memory store is the first implementation of every interface. If it's awkward to write, the interface is wrong:

- **Faking pagination**: if the memory store needs to simulate cursors or offsets, the pagination abstraction may be leaking storage details. But if it just slices an array, the interface is clean.
- **Simulating joins**: if the memory store needs to cross-reference two maps to satisfy a method, the interface might be coupling entities that should stay separate. Or the method might genuinely need data from two sources — in which case, acknowledge it.
- **Enforcing constraints**: if the memory store needs to check foreign keys or unique compound keys, the interface is asking it to be a database. These constraints belong in integration tests, not in the memory implementation.

The memory store doesn't need to be production-correct. It needs to be contract-correct — honoring the interface's behavioral promises (create returns a chirp, duplicate email returns `ErrConflict`) without simulating relational mechanics.


## Relational behavior needs relational tests

Some requirements are inherently systemic. They exist because the data is stored relationally, and they can only be verified against a real database:

- Cascade deletes (FR: "deleting an account removes the user's chirps" → NFR: enforced by `ON DELETE CASCADE`)
- Referential integrity (FR: "a chirp belongs to a user" → NFR: enforced by FK constraint)
- Compound uniqueness (FR: "a user can only vote once per chirp" → NFR: enforced by `UNIQUE(user_id, chirp_id)`)
- Query performance (NFR: "feed loads in under 200ms" → verified by query plan, not by Go code)

These are integration tests. They test the schema design, not the Go logic. Write them when you add a migration that creates a relationship, and treat them as verification that the DDL correctly encodes the business rule.


## The impedance mismatch is permanent

The domain model and the relational model are different representations of the same reality. They will never fully agree:

| Concern | Domain view | Relational view |
|---------|-------------|-----------------|
| Identity | UUID the app generated | Primary key + unique constraints + FK references |
| Relationships | "a chirp has an author" | `chirps.user_id REFERENCES users(id)` |
| Lifecycle | "delete this user" | "what about their chirps, tokens, sessions?" |
| Querying | "show me this user's chirps" | Join, subquery, or two queries? Index strategy? |
| Nesting | Natural in structs and JSON | Requires joins to reconstruct |

This mismatch is not a problem to solve. It's a tension to manage, feature by feature. The store interface is where the negotiation happens. The memory store tests the domain side. The integration tests verify the relational side. Experience teaches you where each business rule belongs — and sometimes only time reveals the answer.

The impulse should always lean toward the domain. The domain is the reason the software exists. The system is how it survives.


## When a domain type serves two masters

The impedance mismatch above is between the domain and the database. There's a second mismatch that's harder to see: within the domain itself.

A `User` starts simple — an ID, an email, a creation timestamp. Then auth adds a password hash and token state. Then a social feature adds a display name and bio. Then billing adds a subscription tier and payment method. The type accumulates fields from different contexts, and every consumer imports the same struct even though each one only cares about a subset.

This is the god entity. It's the data equivalent of a god class, and it creates the same problems: every change risks every consumer, the type becomes hard to reason about, and the test surface grows with every field.

The signal is when different parts of the application want different views of the same identity:

- Auth needs `{ID, Email, PasswordHash, TokenState}`
- The social API needs `{ID, DisplayName, Bio, FollowerCount}`
- Billing needs `{ID, Plan, PaymentMethod}`

They share an ID. That's the real relationship — not a shared struct.

The fix is separate types per context, coordinated by the shared identifier. These can live in the same database, even join on `user_id`, and deploy in the same binary. The boundary is in the code:

```go
// internal/auth/user.go
type Credentials struct {
    UserID       uuid.UUID
    Email        string
    PasswordHash string
}

// internal/domain/user.go (social context)
type User struct {
    ID          uuid.UUID
    DisplayName string
    Bio         string
}
```

You don't need this split on day one. The signal to split is when a change to the type for one consumer forces you to think about the impact on another. If adding a `PasswordHash` field to `domain.User` makes you worry about it appearing in API responses, the type is serving two masters and it's time to separate them.

For Chirpy today, `domain.User` serves one context. When auth arrives, watch for the moment it starts serving two.


## Telemetry closes the loop

FRs and NFRs are predictions. You design based on what you think users will do and what you think the system will need. Telemetry tells you whether those predictions were right.

### Design under uncertainty

Some features ship with high confidence. The business says "chirps are limited to 140 characters" — that's a hard constraint, and you design around it. Other features ship with open questions. Will users post 5 chirps a day or 500? Will the feed endpoint serve 10 results or 10,000? You make your best guess, pick a design, and ship it.

Without telemetry, that guess hardens into permanent architecture. With telemetry, it becomes a hypothesis you can revise.

### What to measure

Instrument the boundaries — the places where the domain meets the system:

- **Endpoint latency**: Is `GET /api/chirps` meeting the NFR? If p95 is 400ms against a target of 200ms, the query strategy needs work — an index, a cache, or a schema change.
- **Error rates by type**: How often does the unique constraint on email actually fire? Is `ErrConflict` a common user experience or a rare edge case? If it's common, the UX might need a "check availability" endpoint. If it's rare, the current error response is fine.
- **Payload sizes**: Are clients receiving 50KB chirp lists because you nested author data they don't use? Telemetry on response sizes reveals whether your response shape matches actual consumption patterns.
- **Query frequency by endpoint**: Which endpoints are hot? If the feed is called 100x more than chirp creation, that's where optimization effort belongs. If an endpoint you agonized over gets 3 requests a day, you over-invested.
- **Database query time**: Separate from endpoint latency. If the handler is fast but the query is slow, the bottleneck is in the schema or the query plan, not in Go code.

### Business constraints as design inputs

Sometimes the business gives you numbers that eliminate uncertainty:

- "We'll have at most 10,000 users at launch" — this tells you whether to invest in pagination now or defer it.
- "The SLA is 99.9% uptime" — this tells you whether you need health checks, graceful shutdown, and connection pooling, or whether a simple restart-on-crash is sufficient.
- "Chirps are write-once, never edited" — this tells you the update path doesn't matter, which simplifies the store interface and the schema.
- "We expect 50 concurrent users, not 50,000" — this tells you whether connection pooling and query optimization are day-one concerns or future work.

These constraints are gifts. They narrow the design space and let you defer complexity that isn't justified yet. Document them alongside the FRs and NFRs so future developers understand why certain decisions were made — and when those constraints might change.

### Scalability and bottlenecks

Every system has a bottleneck. The question is whether you know where it is.

In a typical web API backed by Postgres, the likely bottleneck progression is:

1. **Database connections** — a small pool serving many concurrent requests. Connection pooling (or a tool like PgBouncer) is the first lever.
2. **Query performance** — missing indexes, full table scans, N+1 queries. `EXPLAIN ANALYZE` and query-level telemetry reveal these.
3. **Write contention** — hot rows, lock waits, transaction duration. Shows up as latency spikes under concurrent writes.
4. **Response serialization** — large payloads, unnecessary joins, over-fetching. Shows up as high memory usage and slow responses even when queries are fast.

You don't need to solve all of these upfront. You need to know which one you'll hit first, and have the telemetry in place to detect it. A system that's slow but instrumented can be fixed. A system that's slow and opaque requires guesswork.

### The feedback loop

Design governance isn't a one-time activity. It's a cycle:

```
  Design → Ship → Measure → Revise
    ▲                         │
    └─────────────────────────┘
```

Telemetry feeds back into every layer:

- **FRs**: Users aren't using the nested author data → simplify the response shape in the next version
- **NFRs**: p95 latency exceeds the target → add an index, revise the query, or adjust the NFR if the business accepts it
- **Impedance mismatch**: The join you avoided is actually fine at current scale → simplify the store method. Or: the denormalization you added is causing write amplification → normalize and accept the join cost.
- **API contract**: An endpoint nobody calls → deprecate it. An endpoint everyone misuses → the contract is unclear, improve the docs or the error messages.

The impulse to lean toward the domain still holds. But telemetry tells you when the system is pushing back hard enough that the domain needs to negotiate.


## Checklist for introducing a new entity or endpoint

Before writing code, answer these questions. Not every question applies to every feature — but skipping one should be a deliberate decision, not an oversight.

### Functional

- [ ] What domain operations does this entity support? (create, read, list, update, delete)
- [ ] What does the API endpoint look like? (method, path, request/response shape)
- [ ] What error cases exist? (validation, not found, conflict, unauthorized)
- [ ] How does this entity relate to existing entities? (belongs to, has many, independent)
- [ ] Is this operation a command (changes state) or a query (reads state)? Does the endpoint reflect that?

### Delivery and reliability

- [ ] Is this operation idempotent? Can the client safely retry it?
  - If not, do you need an idempotency key? (e.g., client-generated request ID, `Idempotency-Key` header)
  - If yes, what makes it idempotent? (natural key, upsert semantics, duplicate detection)
- [ ] What delivery guarantee does this operation need?
  - At-most-once: fire and forget, acceptable to lose (analytics events, non-critical notifications)
  - At-least-once: retry until acknowledged, tolerate duplicates (webhook delivery, payment initiation)
  - Exactly-once: requires idempotency + deduplication (financial transactions, balance updates)
- [ ] What happens if this operation partially fails? (e.g., chirp created but notification not sent)
  - Is eventual consistency acceptable, or does the caller need a synchronous guarantee?
  - Should partial failure roll back, or should a background process reconcile?
- [ ] What is the retry strategy?
  - Client-side: exponential backoff with jitter? Max retries? Timeout?
  - Server-side: is the operation safe to retry internally? (e.g., retrying a failed DB write vs retrying a payment)
- [ ] Does this operation participate in a larger workflow? (saga, choreography, orchestration)

### State and concurrency

- [ ] Is this endpoint stateless? Can any replica handle any request?
  - If stateful (sessions, in-memory cache, websocket connections), how is affinity managed?
- [ ] Are there concurrency concerns?
  - Simultaneous creates with the same natural key → unique constraint or idempotency check
  - Simultaneous updates to the same resource → optimistic locking (version column), pessimistic locking (SELECT FOR UPDATE), or last-write-wins
  - Read-after-write consistency → does the client need to see their own write immediately, or is eventual consistency acceptable?
- [ ] What isolation level does this operation need?
  - Default (read committed) is usually sufficient
  - Serializable for operations where phantom reads or write skew would cause business errors (e.g., double-spending)

### Serialization and data shape

- [ ] Is the response shape deliberate? (nested vs flat, included fields)
  - Nesting couples resources and increases payload size but reduces round trips
  - Flat IDs decouple resources but require additional requests
  - Consider: will this response shape survive a future service split?
- [ ] What serialization format? (JSON is default, but consider: protobuf for internal services, CSV for bulk export)
- [ ] Are there fields that should never appear in the response? (passwords, internal IDs, audit metadata)
- [ ] Is the request payload validated at the boundary? (max lengths, allowed values, required fields)
- [ ] Does the response need an envelope? (`{ "data": [...], "next_cursor": "..." }` vs bare array)

### Caching and invalidation

- [ ] Is this data cacheable? (read-heavy, rarely changes, tolerates staleness)
- [ ] What is the cache lifetime? (seconds for feeds, hours for user profiles, indefinite for immutable resources)
- [ ] What invalidates the cache?
  - Time-based (TTL) — simple, tolerates staleness
  - Event-based (write-through, write-behind) — consistent, complex
  - Manual (purge on deploy) — last resort
- [ ] Where does caching happen?
  - Client-side (`Cache-Control`, `ETag`, `304 Not Modified`)
  - Server-side (in-memory, Redis, CDN)
  - Database-level (materialized views, query cache)
- [ ] What happens when the cache is wrong? (stale data served, inconsistent reads across replicas)

### Pagination and bulk operations

- [ ] Will list endpoints need pagination? (if yes, decide now — retrofitting changes the response envelope)
  - Cursor-based: stable under concurrent writes, no count, forward-only (or keyset for bidirectional)
  - Offset-based: simple, supports random access, unstable under concurrent writes
- [ ] What is the default and maximum page size?
- [ ] Does the client need a total count? (expensive on large tables — consider whether it's truly needed)
- [ ] Are there bulk operations? (batch create, batch delete) What are the size limits?

### Data lifecycle

- [ ] What happens when a related entity is deleted? (cascade, reject, orphan, soft-delete)
- [ ] Are there uniqueness constraints beyond the primary key? (email, compound keys)
- [ ] Does this data have a retention policy? (delete after 90 days, archive after 1 year)
- [ ] Is this data write-once (immutable after creation) or mutable?
  - Immutable data is simpler to cache, replicate, and reason about
  - Mutable data needs versioning, conflict resolution, or audit trails
- [ ] Does deletion need to be reversible? (soft delete with `deleted_at` vs hard delete)

### Observability

- [ ] What assumptions are you making about usage patterns? (volume, frequency, payload size)
- [ ] Has the business provided constraints that bound the design? (user count, SLA, write patterns)
- [ ] What telemetry would tell you if those assumptions are wrong?
- [ ] Where is the likely bottleneck? (connections, query time, write contention, serialization)
- [ ] What does an alert look like for this feature? (error rate spike, latency threshold, queue depth)

### Contract

- [ ] Are error responses documented in tests? (status codes, error body shape, error codes)
- [ ] Is this endpoint's behavior something clients will depend on long-term?
- [ ] Is there a versioning strategy if this contract needs to change? (URL versioning, header versioning, additive-only changes)
- [ ] Are breaking changes gated behind a major version bump?

### Operational readiness

These questions connect what you build to how it runs. An SLA commitment constrains your design — 99.9% uptime implies zero-downtime deploys, which implies backward-compatible migrations, which constrains the schema. These aren't afterthoughts; they're design inputs.

**Deployment topology:**
- [ ] Does this feature work when multiple replicas are running simultaneously? (old and new code coexist during rolling updates)
- [ ] Is the migration backward-compatible with the currently running version? (not just the previous release — the version serving traffic right now)
- [ ] Does this feature require a new dependency? If so, what happens to existing pods during rollout?

**Health and failure:**
- [ ] Does this feature affect readiness? (new startup dependency, new subsystem that must initialize before serving)
- [ ] Is the new dependency private (this pod's state) or shared (all pods use it)? Private failures can fail readiness; shared failures should not. See [roadmap/13-always-on-readiness.md](roadmap/13-always-on-readiness.md).
- [ ] If initialization fails, does the process crash or degrade? (Crash is almost always correct — let the orchestrator restart with backoff. A degraded pod that passes liveness but fails readiness is invisible to most monitoring.)
- [ ] What is the recovery path if this feature fails at runtime? (automatic restart, manual intervention, circuit breaker, fallback)

**SLA implications:**
- [ ] Does the SLA require zero-downtime deployment? If so, is the migration additive? Is the API change backward-compatible?
- [ ] What is the blast radius if this feature fails? (one user, one endpoint, the entire service)
- [ ] Does this feature change the connection pool requirements? (new external dependency, additional database queries per request)
- [ ] What container restart behavior should the orchestrator use? (K8s: readiness removal vs liveness restart. ECS: task replacement. Understand the difference — see [devops/04-always-on-deployment.md](roadmap/devops/04-always-on-deployment.md).)

**Operational observability:**
- [ ] Will container restarts increase if this feature has transient startup failures? (monitor restart counts as a signal)
- [ ] Are readiness and liveness probe behaviors documented for this feature's failure modes?
- [ ] Does the team know how to diagnose this feature in production? (logs, metrics, health check output)

Answer the functional questions first — they drive the interface. Then work through the rest — they constrain the interface and reveal complexity that's expensive to discover later. Not every feature touches every category, but the categories you skip should be ones you considered and dismissed, not ones you forgot.
