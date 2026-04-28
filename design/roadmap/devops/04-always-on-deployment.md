# From Single Server to Always-On Service

This doc covers what changes when you move from "one server, deploy and restart" to "multiple replicas, zero downtime, 24/7." It's written for teams who've built and operated single-server systems and are encountering Kubernetes, rolling deployments, and shared database connections for the first time.

The other devops docs in this directory assume a simpler model. This doc explains where that model breaks and what replaces it.


## The single-server mental model

In the traditional model, deployment looks like this:

```
1. Stop the server
2. Run migrations
3. Start the new version
```

There's a maintenance window. During that window, the service is down. Nobody is hitting the old code against the new schema, or the new code against the old schema. The migration and the deploy are one atomic event from the user's perspective.

This model is simple, safe, and incompatible with 24/7 uptime.


## The always-on reality

In Kubernetes (or any multi-replica environment), deployment looks like this:

```
1. Run migrations (old version still serving traffic)
2. Rolling update begins: new pods start, old pods continue serving
3. New pods pass readiness checks, receive traffic
4. Old pods drain connections and terminate
5. All traffic now on new version
```

Between steps 1 and 5, old code and new code are running simultaneously against the same database. This window might last seconds or hours, depending on rollout speed, pod startup time, and readiness probe configuration.

This has consequences for everything: migrations, health checks, connection management, and rollback.


## Migrations must be backward-compatible with the running version

In the single-server model, backward compatibility means "the previous release can work against the new schema" — a safety net for rollback.

In the always-on model, backward compatibility is mandatory for normal operation. The old version is still serving traffic when the migration runs. If the migration breaks the old version, you have an outage during the rollout — exactly the thing you're trying to avoid.

### What's safe during a rolling update

- Add a table (old code doesn't know about it, doesn't care)
- Add a column with a default (old code ignores it, new rows get the default)
- Add an index (transparent to application code)
- Add a constraint that existing data already satisfies

### What's not safe

- Drop a column (old code still selects it → error)
- Rename a column (old code uses the old name → error)
- Add a NOT NULL column without a default (existing inserts from old code fail)
- Change a column type (old code sends the old type → error)

### The expand-contract pattern

Destructive changes require two releases with a transition period between them:

**Release v1.2 (expand):**
- Migration adds the new column
- Code writes to both old and new columns
- Code reads from the new column, falls back to old
- Old pods (v1.1) still work — they ignore the new column

**Release v1.3 (contract):**
- Migration drops the old column
- Code uses only the new column
- v1.2 pods during the v1.3 rollout still work — they write to both, but the old column is gone. This is safe only if v1.2's writes to the old column use `IF EXISTS` or the column drop happens after all v1.2 pods are gone.

The gap between v1.2 and v1.3 might be days or weeks. Both versions must be in production simultaneously during their respective rollouts. Document the two-release plan in both sets of release notes.

This is not optional in a zero-downtime environment. There is no maintenance window to hide behind.


## Migration is a pipeline step, not a manual command

In the single-server model, an operator runs `./chirpy migrate up` by hand. In Kubernetes, migrations run as part of the deployment pipeline:

### Option 1: Kubernetes Job (pre-deploy)

A Job runs before the Deployment rolls out. The pipeline waits for the Job to succeed before updating the Deployment:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: chirpy-migrate-v1-2-0
spec:
  template:
    spec:
      containers:
        - name: migrate
          image: chirpy:v1.2.0
          command: ["./chirpy", "migrate", "up"]
          env:
            - name: DB_URL
              valueFrom:
                secretKeyRef:
                  name: chirpy-db
                  key: url
      restartPolicy: Never
  backoffLimit: 3
```

The pipeline (Argo CD, Flux, GitHub Actions, etc.) applies the Job, waits for completion, then applies the Deployment update.

**Pros:** Migration is explicit, auditable, and separate from application startup. Failed migrations block the rollout.
**Cons:** Requires pipeline orchestration. The Job uses the new binary, so the image must be built before migration runs.

### Option 2: Init container

The migration runs as an init container on every pod. The first pod to start runs the migration; subsequent pods see no pending migrations and proceed immediately:

```yaml
initContainers:
  - name: migrate
    image: chirpy:v1.2.0
    command: ["./chirpy", "migrate", "up"]
    env:
      - name: DB_URL
        valueFrom:
          secretKeyRef:
            name: chirpy-db
            key: url
```

**Pros:** Simple. No separate Job to manage. Migration happens automatically.
**Cons:** Multiple pods starting simultaneously can race on migrations. Goose uses advisory locks to prevent this, but it adds startup latency and complexity. A failed migration blocks all pods from starting — which might be what you want, or might cause a cascading outage if the migration is slow.

### Recommendation

Use a Job for production. It's explicit, it's auditable, and it separates "change the schema" from "start the application." Init containers are fine for development and staging where the consequences of a race or a slow migration are lower.


## Health checks become load balancer signals

In the single-server model, a health check is informational — you look at it in the logs. In Kubernetes, health checks control traffic routing:

**Readiness probe:** "Is this pod ready to receive traffic?" If it fails, the pod is removed from the Service's endpoint list. No traffic is routed to it.

**Liveness probe:** "Is this pod still functioning?" If it fails, Kubernetes kills and restarts the pod.

The schema version check from [roadmap/11-schema-version-check.md](../11-schema-version-check.md) maps directly to the readiness probe. A pod that starts against a stale schema should fail readiness so the load balancer doesn't send it requests that will fail with SQL errors.

```yaml
readinessProbe:
  httpGet:
    path: /api/healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

If `/api/healthz` returns 503 when migrations are pending, Kubernetes won't route traffic to that pod until the migration Job completes and the pod's next readiness check passes.

The liveness probe should be simpler — just "can the process respond." Don't tie it to database connectivity or schema state. A pod that can't reach the database is unhealthy but shouldn't be killed and restarted in a loop (the database being down isn't fixed by restarting the application).


## Connection management across replicas

A single server opens one connection pool to Postgres. Three replicas open three pools. If each pool has 10 connections, that's 30 connections to Postgres — and Postgres has a default limit of 100.

### The math

```
connections = replicas × pool_size
```

With autoscaling, replicas can spike. 10 replicas × 10 connections = 100, which hits the Postgres default. New connections fail, requests fail, pods fail readiness, Kubernetes starts more pods (which need more connections), and you have a cascading failure.

### Solutions

**Right-size the pool.** Set `max_open_conns` in your Go `sql.DB` configuration. A pool of 5 per replica is often enough. Measure before increasing.

**PgBouncer.** A connection pooler that sits between your application and Postgres. Your pods connect to PgBouncer (which accepts thousands of connections) and PgBouncer maintains a small pool to Postgres. This decouples replica count from database connection count.

**Serverless Postgres.** Managed services like RDS Proxy, Neon, or Supabase's connection pooler handle this transparently. Worth evaluating if you're on a managed platform.

### Graceful shutdown

When Kubernetes terminates a pod (during a rolling update or scale-down), it sends SIGTERM. The application should:

1. Stop accepting new requests
2. Finish in-flight requests
3. Close database connections
4. Exit

The `runUntilInterrupt` function in `main.go` already handles SIGTERM via `signal.Notify`. In Kubernetes, you also need a `preStop` hook or a `terminationGracePeriodSeconds` long enough for in-flight requests to complete:

```yaml
terminationGracePeriodSeconds: 30
lifecycle:
  preStop:
    exec:
      command: ["sleep", "5"]  # allow load balancer to deregister the pod
```

The 5-second sleep gives the load balancer time to stop routing new traffic before the application begins draining. Without it, the pod starts shutting down while the load balancer is still sending it requests.


## Rollback means rolling back the binary, not the schema

In Kubernetes, rolling back a Deployment reverts the container image to the previous version. It does not roll back the database.

If the migration was backward-compatible (additive), this is fine — the old code works against the new schema. This is the normal case and the reason backward-compatible migrations are mandatory, not optional.

If the migration was not backward-compatible, you have a problem:
- The old binary can't work against the new schema
- Rolling back the migration requires manual intervention (`goose down-to`)
- During the rollback window, the service may be degraded or down

This is why the expand-contract pattern exists. It ensures that at every point in the deployment lifecycle, the running code is compatible with the current schema.

### Rollback checklist

1. Is the migration backward-compatible? → Roll back the Deployment. Done.
2. Is the migration not backward-compatible? → This should not happen if you followed expand-contract. If it did:
   a. Roll back the Deployment
   b. Run `goose down-to <previous>` manually or via a Job
   c. Verify the old version works
   d. Investigate why a non-backward-compatible migration shipped without the expand phase


## What changes in the existing docs

The other devops docs in this directory were written with the single-server model in mind. Here's how they map to the always-on model:

| Doc | Single-server assumption | Always-on reality |
|-----|--------------------------|-------------------|
| [02-migration-discipline](02-migration-discipline.md) | Migrations run during a maintenance window | Migrations run while old code serves traffic |
| [03-release-process](03-release-process.md) | "Migrate then deploy" is one step | Migrate is a Job; deploy is a rolling update; there's a window between them |
| [03-release-process](03-release-process.md) | Customer runs `./chirpy migrate up` by hand | Pipeline runs it as a Job; customer configures the pipeline |
| [03-release-process](03-release-process.md) | Rollback = stop, migrate down, start old version | Rollback = revert Deployment; schema stays; backward compatibility is mandatory |

The migration discipline (timestamps, `goose fix`, `--allow-missing`) is unchanged. The release process (tagging, release notes, semver) is unchanged. What changes is the deployment mechanics and the stronger requirement on backward compatibility.


## Health checks across orchestrators

The `/readyz` and `/livez` endpoints are application-level signals. The orchestrator decides which to use and what action to take on failure. The application reports its state honestly and doesn't know or care what's consuming the endpoints.

Different orchestrators interpret health checks differently:

| Orchestrator | Readiness equivalent | Liveness equivalent | Failure action |
|---|---|---|---|
| Kubernetes | Readiness probe | Liveness probe | Readiness: remove from Service. Liveness: restart pod. |
| ECS (Fargate/EC2) | ALB target group health check or task health check | None (single check) | Unhealthy task is replaced entirely |
| Docker Swarm | `HEALTHCHECK` in Dockerfile | None (single check) | Unhealthy container is replaced |
| Nomad | `check` block (service discovery) | Restart policy | Check failure: deregister from service discovery. Restart: replace task. |
| Systemd + LB | Load balancer health check | `WatchdogSec` (sd_notify) | LB: stop routing. Watchdog: restart unit. |

**The key difference is between one-signal and two-signal systems.**

Kubernetes has two signals: readiness (stop routing) and liveness (restart). This lets a pod stay alive while temporarily unable to serve — waiting for a dependency to recover, for example.

ECS, Swarm, and most managed container services have one signal. A failing health check means replacement, not just traffic removal. This makes the shared-dependency cascade problem from [Breck's article](https://blog.colinbreck.com/kubernetes-liveness-and-readiness-probes-how-to-avoid-shooting-yourself-in-the-foot/) worse: if all tasks fail because a shared database is slow, the orchestrator replaces them all simultaneously. New tasks start, the database is still slow, they fail too. You get a replacement storm.

**Implication for the application:** the private/shared dependency classification in `/readyz` (see [roadmap/12-always-on-readiness.md](../12-always-on-readiness.md)) matters even more on one-signal orchestrators. A shared dependency failure that causes the health check to fail will trigger task replacement cascades. On any orchestrator, `/readyz` should only fail on private dependencies — things specific to this instance's state, not shared infrastructure.

The zombie pod problem (a pod that never becomes ready but stays alive) only exists on Kubernetes, because K8s is the only orchestrator that keeps unhealthy containers running. On ECS and Swarm, unhealthy containers are replaced automatically — which is simpler but also means you can't keep a degraded instance alive while investigating.

Build your health endpoints for the most capable orchestrator (K8s, two signals). Simpler orchestrators use a subset — typically just `/readyz` as their single health check.


## Summary

| Concern | Single server | Always-on (K8s) |
|---------|---------------|-----------------|
| Deployment | Stop, migrate, start | Job migrates, rolling update deploys |
| Downtime during deploy | Yes (maintenance window) | No (zero-downtime required) |
| Old and new code coexist | Never | Always, during rollout |
| Migration backward compat | Nice to have (for rollback) | Mandatory (for normal operation) |
| Destructive schema changes | One release, one window | Expand-contract across two releases |
| Health checks | Informational | Controls traffic routing |
| Connection pooling | One pool, one server | Replicas × pool size; use PgBouncer |
| Graceful shutdown | Stop the process | SIGTERM → drain → close → exit |
| Rollback | Migrate down + restart | Revert Deployment; schema stays |
| Migration execution | Manual CLI command | Pipeline Job or init container |
