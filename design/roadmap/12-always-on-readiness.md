# Always-On Readiness

Make the Chirpy binary suitable for multi-replica, zero-downtime deployment. This covers the minimum changes needed to run safely in Kubernetes or any environment where pods start, stop, and scale without a maintenance window.

Design rationale: [design/roadmap/devops/04-always-on-deployment.md](devops/04-always-on-deployment.md)


## What you'll build

- SIGTERM handling and shutdown timeout
- Database connection pool limits
- Separate readiness and liveness endpoints
- Startup schema gate (builds on [11-schema-version-check.md](11-schema-version-check.md))


## Step 1: SIGTERM and shutdown timeout

Currently `runUntilInterrupt` only catches `os.Interrupt` (SIGINT, i.e., Ctrl-C). Kubernetes sends `SIGTERM` when terminating a pod. The server doesn't hear it, so K8s force-kills the pod after `terminationGracePeriodSeconds` with in-flight requests still running.

### Red

Write a test (or verify manually) that sending SIGTERM to the process triggers a graceful shutdown. This is hard to unit test — it's a process-level signal. A manual verification is acceptable:

```bash
./chirpy &
PID=$!
kill -TERM $PID
# should see graceful shutdown log, not a crash
```

### Green

Two changes in `runUntilInterrupt`:

1. Add `syscall.SIGTERM` to the signal list alongside `os.Interrupt`
2. Replace `context.Background()` in the `Shutdown` call with a timeout context — 15 seconds is a reasonable default. If in-flight requests don't finish by then, the server force-closes.

Hints:
- `signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)`
- `ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)`
- Log which signal was received — useful for debugging shutdown behavior in production

### Explore

- What happens to in-flight database queries when the server shuts down? Does `db.Close()` wait for them, or does it cancel them?
- What's the relationship between the Go shutdown timeout and Kubernetes `terminationGracePeriodSeconds`? (The K8s value should be longer than the Go value, so the application finishes draining before K8s force-kills it.)


## Step 2: Connection pool limits

`sql.Open` returns a `*sql.DB` with no connection limits. The pool grows on demand and never shrinks. With multiple replicas, each pool grows independently, and the total connections can exceed Postgres's `max_connections` (default 100).

### Red

There's no good way to test connection limits in a unit test. This is a configuration change verified by inspection and load testing. The "test" is: after the change, `db.Stats()` should show bounded values.

### Green

After `sql.Open` in `main.go` (or in `postgres.Open`), set pool limits:

```go
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

Rules of thumb:
- `MaxOpenConns`: start with 10. Measure under load. Increase only if queries are waiting for connections.
- `MaxIdleConns`: half of `MaxOpenConns` or less. Idle connections consume memory on both sides.
- `ConnMaxLifetime`: prevents stale connections after Postgres restarts or network changes. 5 minutes is a safe default.

### Explore

- What happens when all connections are in use and a new request arrives? (It blocks until one is returned to the pool, or the request's context times out.)
- If you have 3 replicas with `MaxOpenConns=10`, that's 30 connections. What's your Postgres `max_connections`? Is there headroom for the migration Job and monitoring tools?
- Should these values come from environment variables or be hardcoded? (Environment variables are more flexible for different deployment targets.)


## Step 3: Readiness and liveness probes

Currently `/api/healthz` always returns 200. It's useless as a Kubernetes readiness probe — it says "I'm ready" even when the database is unreachable or the schema is stale.

Kubernetes needs two signals:

- **Readiness** (`/readyz`): "Can I fulfill my role in the system right now?" If no, K8s removes the pod from the Service. Traffic stops flowing to it. The pod is not restarted.
- **Liveness** (`/livez`): "Is this process stuck?" If no, K8s kills and restarts the pod.

These probes are powerful and dangerous. Implemented carelessly, they make availability worse, not better. Read Colin Breck's [Kubernetes Liveness and Readiness Probes: How to Avoid Shooting Yourself in the Foot](https://blog.colinbreck.com/kubernetes-liveness-and-readiness-probes-how-to-avoid-shooting-yourself-in-the-foot/) before designing your probes. The key lessons:

- **A readiness probe that checks a shared dependency can take down the entire service.** If all pods check the same database and it has a latency spike, all pods fail readiness simultaneously. K8s removes them all from the Service. The service returns 404 for every request. The database was slow for one second; your service was down for thirty.
- **A liveness probe that checks a dependency causes restart cascades.** Database goes down → liveness fails → pod restarts → comes back, database still down → liveness fails again → restart loop. Meanwhile the restart adds load to other pods, which also start failing. The cure is worse than the disease.
- **Probe timeouts and failure thresholds must be conservative.** Don't fail on the first slow response. System dynamics change over time — latency increases, startup times grow, networks get congested.

### Liveness: keep it simple

The `/livez` endpoint should verify that the process itself is responsive — not that its dependencies are healthy. If the HTTP handler can execute and write a response, the process is alive.

```go
func (s *Server) handleLivez(w http.ResponseWriter, _ *http.Request) {
    w.WriteHeader(http.StatusOK)
    fmt.Fprintf(w, "ok")
}
```

That's it. No database check. No dependency check. A pod that can't reach the database is unhealthy (readiness handles that) but not stuck (restarting it won't fix the database).

One subtlety from Breck: the liveness probe must exercise the same HTTP server that handles real traffic. If you run probes on a separate listener, the probe can pass while the main server is deadlocked. Since Chirpy uses a single `http.ServeMux`, this isn't a concern — `/livez` goes through the same listener as `/api/chirps`.

Configure the K8s probe conservatively:

```yaml
livenessProbe:
  httpGet:
    path: /livez
    port: 8080
  initialDelaySeconds: 30    # longer than worst-case startup
  periodSeconds: 10
  timeoutSeconds: 5          # same magnitude as client timeouts
  failureThreshold: 3        # don't restart on a single slow response
```

`initialDelaySeconds` must be longer than the maximum time the application takes to start — and that time changes as data grows and networks change. Be generous. Regularly exercise restarts to verify the value is still sufficient.

### Readiness: private vs shared dependencies

Readiness is more nuanced. The question is not just "are my dependencies up?" but "will failing this probe make things better or worse?"

**Private dependencies** are exclusive to this pod. If they're down, this specific pod can't serve traffic, but other pods are unaffected. Failing readiness is safe — it routes traffic to healthy pods.

- Schema status (cached at startup — private to this pod's state)
- A local cache that hasn't finished loading

**Shared dependencies** are used by all pods. If they're slow or down, all pods fail readiness simultaneously, and the service goes completely offline. Failing readiness makes things worse.

- The Postgres database (all pods share it)
- An authentication service
- A message broker

For shared dependencies, the right response to a transient failure is to let requests through and let them fail with meaningful errors — not to remove the pod from the load balancer. The client gets a 500 or 503 with an error message, which is better than a 404 from K8s because no backends exist.

### Design: dependency check registry

Define a registry that each subsystem contributes to, with a classification for how failures should be handled:

```go
type CheckType int

const (
    CheckPrivate CheckType = iota  // failure → pod not ready
    CheckShared                     // failure → log/alert, but don't fail readiness
)

type DependencyCheck struct {
    Name  string
    Type  CheckType
    Check func(ctx context.Context) error
}
```

At startup, register checks:

```go
checks := []DependencyCheck{
    {Name: "schema", Type: CheckPrivate, Check: schemaCheck},
    {Name: "database", Type: CheckShared, Check: dbPingCheck},
    // future: {Name: "cache-partition", Type: CheckPrivate, Check: cachePartitionCheck},
    // future: {Name: "auth-service", Type: CheckShared, Check: authCheck},
}
```

The `/readyz` handler runs all checks. Private check failures return 503. Shared check failures are reported in the response body (for observability) but don't fail the probe:

```json
{
  "status": "ready",
  "checks": {
    "schema": "ok",
    "database": "timeout (shared, non-blocking)"
  }
}
```

This way, a database latency spike shows up in monitoring (the response body says the check failed) but doesn't cascade into a service-wide outage.

When a future release adds a dependency — a data grid partition (private, must be loaded before serving), an external auth service (shared, don't fail readiness if it's slow) — the team classifies it and adds one check. The readiness handler doesn't change.

### Red

Write tests for both endpoints:

```go
// Readiness: returns 200 when private checks pass (even if shared checks fail)
func TestReadyzAllHealthy(t *testing.T) { ... }
func TestReadyzPrivateCheckFailing(t *testing.T) { ... }   // → 503
func TestReadyzSharedCheckFailing(t *testing.T) { ... }    // → 200, but body reports failure

// Liveness: always returns 200
func TestLivez(t *testing.T) { ... }
```

Inject mock checks that return nil or an error. The readiness handler shouldn't know what it's checking — it just runs the registered checks and applies the private/shared classification.

### Green

Add two endpoints:

```
GET /readyz   → readiness (private dependency checks must pass)
GET /livez    → liveness (process is responsive)
```

For the initial implementation, register two checks:
- **Schema** (private): cached boolean set at startup (from [11-schema-version-check.md](11-schema-version-check.md)). If the schema isn't current, this pod specifically can't serve correct responses.
- **Database** (shared): `db.PingContext(ctx)` with a generous timeout. Reported in the response for observability, but doesn't fail the probe. All pods share the same database — if it's down, removing all pods from the Service makes things worse.

Configure the K8s probe conservatively:

```yaml
readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3       # don't remove from Service on a single failure
```

### Zombie pods: readiness without liveness is not enough

A pod that fails to initialize but doesn't crash will stay `Running` forever, never becoming ready. K8s won't restart it (liveness passes — the process responds). K8s won't route traffic to it (readiness fails). It just sits there, consuming resources and reducing the effective replica count.

Over time, as pods restart for routine reasons (node rebalancing, scaling events), more pods can land in this state. The service gradually loses capacity with no alerts — every pod shows `Running` in `kubectl get pods`.

Our startup sequence avoids this for the schema check case — `log.Fatalf` crashes the process, K8s restarts it with exponential backoff, and eventually the migration Job completes. But the general principle matters for any future startup dependency: **if a pod can never become ready, it should crash, not linger.**

Breck's follow-up post channels Joe Armstrong's Erlang philosophy: when you encounter an error you don't know how to recover from, let the process die. Kubernetes is the supervisor — it restarts the container with backoff. Trying to handle the error gracefully (log and keep running in a degraded state) is worse than crashing, because a degraded pod that passes liveness but fails readiness is invisible to most monitoring.

The rule: if startup fails in a way that means the pod will never become ready, `log.Fatalf`. Don't try to be clever. Let K8s do its job.

Reference: [Kubernetes Liveness and Readiness Probes Revisited](https://blog.colinbreck.com/kubernetes-liveness-and-readiness-probes-revisited-how-to-avoid-shooting-yourself-in-the-other-foot/)

### What about the existing `/api/healthz`?

Keep `/api/healthz` returning 200 OK for backward compatibility (existing monitoring, curl checks, the Boot.dev course tests). Add `/readyz` and `/livez` as the K8s-specific endpoints.

### Explore

- Read the follow-up: [How to Avoid Shooting Yourself in the Other Foot](https://blog.colinbreck.com/kubernetes-liveness-and-readiness-probes-revisited-how-to-avoid-shooting-yourself-in-the-other-foot/).
- What happens if the readiness probe fails during a rolling update? (K8s stops routing traffic to new pods. If all new pods fail readiness, the rollout stalls. The old pods keep serving. This is what you want.)
- How would you add a check for membership in a distributed data grid? Is it private or shared? (Private — it's this pod's partition state, not a shared service.)
- What if a dependency is private at startup (cache loading) but shared at runtime (cache cluster)? How would you model that transition?
- Breck recommends regularly restarting pods to exercise startup dynamics. How does this interact with `initialDelaySeconds` and the schema version check?


## Step 4: Wire it together

After completing steps 1–3, verify the full startup sequence:

1. Open database connection (with pool limits)
2. Check for pending migrations (from doc 10)
3. If pending: log the list, exit non-zero
4. If current: set schema status to ready
5. Register readiness and liveness endpoints
6. Start the HTTP server
7. On SIGTERM: drain in-flight requests (with timeout), close database, exit

### Manual verification

```bash
# Build and start against a migrated database
make build
DB_URL="..." ./tmp/main &

# Readiness should be 200
curl -s localhost:8080/readyz

# Liveness should be 200
curl -s localhost:8080/livez

# Graceful shutdown
kill -TERM $!
# should see shutdown log, exit 0
```

```bash
# Start against a database with pending migrations
# should exit immediately with error message
DB_URL="..." ./tmp/main
```


## Checklist

- [ ] `runUntilInterrupt` handles SIGTERM
- [ ] Shutdown uses a timeout context (not `context.Background()`)
- [ ] Database connection pool has `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`
- [ ] Dependency check registry with private/shared classification
- [ ] `/readyz` returns 503 when a private check fails, 200 otherwise (shared failures reported but non-blocking)
- [ ] `/livez` returns 200 unconditionally (no dependency checks)
- [ ] Existing `/api/healthz` still works (backward compatibility)
- [ ] Startup exits if migrations are pending
- [ ] README documents the new health endpoints
