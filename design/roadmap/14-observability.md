# Doc 14 — Observability

## Goal

Make the application's runtime behavior visible to operators. Metrics, logging, and health probes should be swappable adapters behind application-defined interfaces — not concrete types baked into the application layer.

## Scope

### Metrics as interface

`application.ServerMetrics` is currently a concrete struct. Extract a metrics interface in `application/` and move the implementation to an adapter. Assembly wires the appropriate implementation (production Prometheus, no-op for tests, in-memory for dev).

### Structured logging

Define a logging contract in `application/`. Adapters provide implementations (e.g., `slog`-based, JSON, no-op). Handlers and services log through the interface, not directly to `log` or `fmt`.

### Health and readiness

Expose operational endpoints (`/healthz`, `/readyz`) that verify environment fitness — DB connectivity, migration status, dependency availability. These are the operator's window into whether the system is safe to receive traffic.

### Tracing (future)

Request-scoped trace IDs propagated through context. Not needed immediately, but the interface structure should accommodate it without refactoring.

## Principles

- Observability is a cross-cutting concern but not a special case. It follows the same dependency rules as everything else: interfaces in `application/`, implementations in adapters.
- Operators should be able to verify environment fitness before, during, and after deployment.
- No observability code in `domain/`. The domain doesn't know it's being observed.

## Relationship to other docs

- Subsumes the metrics portion of issue 3 from `10-remaining-issues.md`
- Extends doc 13 (always-on readiness) with the broader observability story
