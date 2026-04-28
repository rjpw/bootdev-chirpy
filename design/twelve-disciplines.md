# The Twelve Disciplines of a Responsible Web Service

Inspired by the [Twelve-Factor App](https://12factor.net), but written for a different moment. The original twelve factors were about making apps portable across PaaS platforms. These twelve disciplines are about making a web service a responsible citizen in a community of internet-facing services — the kind of service you'd trust with someone's money, someone's data, or someone's time.

They're drawn from hard lessons in financial operations, cloud migration, and the experience of watching systems that worked fine on one server fail in ways nobody predicted when they became always-on, multi-replica, customer-facing services.

This isn't a checklist for day one. It's a checklist for the service your code will become. Read it before you write your first endpoint. Return to it every time you add a feature.


## I. Domain First

The domain is the reason the software exists. Every design decision — API shape, store interface, schema, deployment strategy — should be traceable to a domain need. When the domain and the system disagree, lean toward the domain.

Start from what users need, not from what the database looks like. Define behavior before storage. Let the API endpoint drive the store interface, and the store interface drive the schema. The database is the last thing you touch, not the first.

*See: [feature-development-loop.md](feature-development-loop.md), [design-governance.md](design-governance.md)*


## II. Explicit Contracts

Every endpoint is a promise. The request shape, the response shape, the status codes, the error format — these are contracts with every client that consumes them. Once shipped, changing them is a negotiation, not a refactor.

Design response shapes deliberately. Document error responses in tests. Decide pagination style before you ship the first list endpoint. Know the difference between additive changes (safe) and breaking changes (require a new version). Treat your API with the same care you'd treat a legal agreement, because to your customers, it is one ( and then some ... see [Hyrum's Law](https://www.hyrumslaw.com/) ).

*See: [api-lifecycle.md](api-lifecycle.md), [design-governance.md](design-governance.md)*


## III. Honest Health

A service must report its own health accurately. Not "the process is running" — that's trivial. "I can fulfill my role in the system right now" — that's health.

Separate liveness (is the process stuck?) from readiness (can I serve traffic?). Classify dependencies as private (this instance's state) or shared (all instances use it). Never fail readiness on a shared dependency — you'll cascade a latency spike into a total outage. If startup fails irrecoverably, crash. Let the orchestrator restart you. A degraded process that lingers is worse than one that dies cleanly.

*See: [roadmap/13-always-on-readiness.md](roadmap/13-always-on-readiness.md), [roadmap/devops/04-always-on-deployment.md](roadmap/devops/04-always-on-deployment.md)*


## IV. Schema as Code

The database schema is not a side effect of development. It's a versioned, tested, embedded artifact. Migrations live in the repository, are applied by the binary, and are verified by integration tests against a real database.

Never edit an applied migration. Use timestamps during development, sequential numbers for release. Embed migrations in the binary so deployment doesn't depend on external files or tools. The binary carries everything it needs to bring a database to the correct state.

*See: [roadmap/08-migrate-subcommand.md](roadmap/08-migrate-subcommand.md), [roadmap/devops/02-migration-discipline.md](roadmap/devops/02-migration-discipline.md), [sql-vs-go-migrations.md](sql-vs-go-migrations.md)*


## V. Safe Evolution

A running system must be changeable without downtime. This means every migration must be backward-compatible with the currently running code — not just the previous release, but the version serving traffic right now, during the rolling update.

Destructive schema changes require the expand-contract pattern: add in one release, remove in the next, with a transition period between them. API changes follow the same discipline: version, deprecate, sunset. The gap between releases might be weeks. Both versions must coexist safely.

*See: [roadmap/devops/04-always-on-deployment.md](roadmap/devops/04-always-on-deployment.md), [api-lifecycle.md](api-lifecycle.md), [roadmap/devops/03-release-process.md](roadmap/devops/03-release-process.md)*


## VI. Dependency Boundaries

Every external dependency — database, cache, message broker, third-party service — is accessed through an interface, not a concrete implementation. The interface speaks domain language. The implementation translates.

This isn't abstraction for its own sake. It's how you test without Docker, swap implementations without rewriting handlers, and split a monolith when the business demands it. The memory store proves the interface. The postgres store proves the SQL. The API tests prove the behavior. Each layer has a job.

*See: [roadmap/01-store-interface.md](roadmap/01-store-interface.md), [roadmap/10-scaling-the-store-layer.md](roadmap/10-scaling-the-store-layer.md), [feature-development-loop.md](feature-development-loop.md)*


## VII. Graceful Participation

A service doesn't exist alone. It runs alongside other replicas, behind a load balancer, managed by an orchestrator. It must start cleanly, signal its readiness, drain connections on shutdown, and release resources when asked.

Handle SIGTERM. Use a shutdown timeout. Limit your connection pool. Understand that your orchestrator will start, stop, and replace your process without asking — and design for that. A service that can't be safely restarted can't be safely deployed.

*See: [roadmap/13-always-on-readiness.md](roadmap/13-always-on-readiness.md), [roadmap/devops/04-always-on-deployment.md](roadmap/devops/04-always-on-deployment.md)*


## VIII. Measured Confidence

Ship with hypotheses, not certainties. You think the feed will have 50 chirps. You think p95 latency will be under 200ms. You think the unique constraint will rarely fire. Instrument the boundaries and find out.

Measure endpoint latency, error rates by type, payload sizes, query frequency, and database query time. Let telemetry tell you whether your design assumptions were right. When they're wrong, revise the design — don't defend the assumption.

*See: [design-governance.md](design-governance.md) (Telemetry closes the loop)*


## IX. Idempotent by Default

In a distributed system, every message can be delivered more than once. Every request can be retried. Every network call can time out and be repeated. Design for this.

Operations that change state should be safe to retry. Use natural keys, idempotency keys, or upsert semantics. Know the difference between at-most-once, at-least-once, and exactly-once — and know that exactly-once requires idempotency plus deduplication, not wishful thinking.

*See: [design-governance.md](design-governance.md) (Delivery and reliability checklist)*


## X. Minimal Privilege

A service should have access to exactly what it needs and nothing more. Database credentials come from environment variables or secrets managers, not from source code. The dev environment uses different credentials than production. The reset endpoint is gated behind a platform check.

Don't embed secrets. Don't grant broad database permissions. Don't expose admin endpoints without access control. Every permission is an attack surface.


## XI. Reproducible Environments

A developer should be able to clone the repo, run one command, and have a working system — including the database, migrations, and test data. No tribal knowledge. No "ask Sarah for the staging credentials."

Use testcontainers for integration tests. Embed migrations in the binary. Provide a `.env.example`. Document every prerequisite in the README. If a new developer can't run the tests in 10 minutes, the onboarding is broken.

*See: [roadmap/04-testdb-helper.md](roadmap/04-testdb-helper.md), [roadmap/07-developer-workflow.md](roadmap/07-developer-workflow.md), README.md*


## XII. Continuous Verification

Tests are not a gate you pass once. They're a continuous signal that the system works. Fast tests run on every save. Integration tests run on every push. The test pyramid exists so you can iterate quickly at the top (behavior, no Docker) and verify thoroughly at the bottom (real database, real constraints).

Test the domain with the memory store. Test the SQL with the real database. Test relational behavior — cascades, foreign keys, compound uniqueness — where it lives: in integration tests against Postgres. Don't simulate what the database does for free. Don't skip what only the database can verify.

*See: [feature-development-loop.md](feature-development-loop.md), [roadmap/05-sqlc-query-tests.md](roadmap/05-sqlc-query-tests.md), [roadmap/06-postgres-store.md](roadmap/06-postgres-store.md)*


---

These twelve disciplines aren't rules to follow blindly. They're the distillation of what goes wrong when you don't think about them — learned from systems that handled real money, served real customers, and ran 24/7 in environments where "we'll fix it in the next deploy" meant "we'll fix it at 3am when the on-call gets paged."

A new developer reading this list should understand not just what to build, but why it matters. An experienced developer should recognize the scars. And anyone guiding an AI to write their code should know that the AI can generate the implementation, but only a human who's been on the front lines can judge whether the design is responsible.
