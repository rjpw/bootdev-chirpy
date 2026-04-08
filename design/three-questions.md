# The Three Questions

*An audit framework for responsible services in programmable ecosystems.*


## The premise

The boundary of an application is no longer the binary. It's the full set of artifacts that define the binary's relationship with every system it touches — the Terraform module, the Helm chart, the migration Job, the Datadog monitor, the CI pipeline, the Vault policy, the Ansible playbook. If any of these are wrong, the application doesn't work correctly, and the customer feels it.

These artifacts are all code. They live in repositories (or should). They're versioned, reviewed, tested (or should be). And they're all expressions of programmable APIs — K8s, AWS, Datadog, GitHub Actions, Vault, Postgres. The application doesn't just run on infrastructure. It participates in a programmable ecosystem, and the quality of that participation is part of the application's quality.

This changes what it means to evaluate a service. Internal code quality — clean handlers, tested store interfaces, idempotent operations — is necessary but insufficient. The service must also be fit for its ecosystem: deployable without downtime, observable under failure, recoverable without heroics.

Gene Kim and Steven Spear, in *Wiring the Winning Organization*, provide the vocabulary. They identify three mechanisms that move problem-solving from danger zones to winning zones: slowification, simplification, and amplification. Applied to programmable ecosystems, these mechanisms become three questions you can ask at every layer of the stack, about every artifact, at every stage of the lifecycle.


## The three questions

**1. Did you practice this before production?** (Slowify)

Every action that will happen in production should have happened first in an environment that forgives mistakes. Not "we have a staging environment" — that's a place, not a practice. The question is whether the specific action was rehearsed: this migration, this Terraform plan, this rollout strategy, this alert threshold.

**2. Can a failure here cascade beyond its boundary?** (Simplify)

Every component should have a blast radius. A bad migration shouldn't take down unrelated tables. A misconfigured monitor shouldn't silence alerts for the entire platform. A Terraform state file shouldn't let one `apply` touch both staging and production. The question is whether boundaries exist and whether they hold.

**3. Will someone know in time to act?** (Amplify)

Every failure should produce a signal that reaches someone (or something) with the authority and context to respond. Not "we log errors" — that's emission, not amplification. The question is whether the signal is loud enough, fast enough, and routed to the right place.


## The layers

A service touches many programmable surfaces. The three questions apply to each one independently. A service can be well-slowified at the application layer (comprehensive tests, memory store first) and completely unslowified at the infrastructure layer (Terraform plans applied without review). The audit is per-layer.

### Application code

The binary, its handlers, its store interfaces, its domain logic. This is where the [twelve disciplines](twelve-disciplines.md) live.

| Question | What to look for |
|---|---|
| Slowify | Tests run before merge. Memory store proves the interface before Postgres. Migrations run in CI against a real database. Feature development follows the red-green-refine-refactor loop. |
| Simplify | Store interfaces decouple handlers from storage. Domain boundaries contain failures. Idempotent operations tolerate retries without coordination. Private vs shared dependency classification limits cascade. |
| Amplify | Health endpoints report real status. Telemetry instruments boundaries. Error responses are structured and testable. Metrics expose whether design assumptions hold. |

### Schema and migrations

The database schema, migration files, and the processes that apply them.

| Question | What to look for |
|---|---|
| Slowify | Migrations are tested in CI against a real database. Expand-contract is planned across releases, not improvised. `goose fix` happens on the release branch, not in a panic. The migration Job runs in a pipeline stage before the new binary rolls out. |
| Simplify | Each migration does one thing. State files are scoped (no single migration that creates tables, adds indexes, and backfills data). Backward compatibility means the running version and the new version coexist safely. |
| Amplify | Migration status is checked at startup. Schema version mismatch is a hard failure, not a log line. `./chirpy migrate status` is available in production without external tools. |

### Infrastructure (Terraform, Pulumi, CloudFormation)

The resources the application depends on — VPCs, databases, IAM roles, DNS, load balancers.

| Question | What to look for |
|---|---|
| Slowify | `plan` before `apply`, always. Plans are posted to PRs for review. A staging account mirrors production topology. Failover has been exercised, not just configured. Destructive changes require manual approval. |
| Simplify | State is split per environment and per concern. One bad `apply` can't touch production and staging simultaneously. Resources are grouped so a failure in one state file doesn't affect unrelated infrastructure. IAM roles are scoped to least privilege. |
| Amplify | Drift detection runs on a schedule. Unexpected changes trigger alerts. `plan` output is visible in the PR, not buried in CI logs. Cost anomalies are monitored. |

### Configuration management (Ansible, Packer, cloud-init)

The machine state — OS packages, kernel parameters, file permissions, systemd units.

| Question | What to look for |
|---|---|
| Slowify | Playbooks are idempotent and tested against throwaway instances before the fleet. AMIs are built and validated in CI. Configuration changes are versioned and reviewed like application code. |
| Simplify | Playbooks are scoped to roles. One playbook doesn't configure everything from DNS to application config. Changes to one service's configuration can't break another service on the same host. |
| Amplify | Playbook failures are reported to a channel, not swallowed. Configuration drift is detected. A host that diverges from its expected state is visible, not silent. |

### Orchestration (Kubernetes, ECS, Nomad)

How the application starts, scales, fails, and recovers.

| Question | What to look for |
|---|---|
| Slowify | Readiness probes prevent traffic before the application is ready. Rolling updates proceed one pod at a time. PodDisruptionBudgets prevent the orchestrator from taking down too many replicas. `initialDelaySeconds` is calibrated against measured startup time, not guessed. Canary deployments exist for high-risk changes. |
| Simplify | Namespace isolation. Resource quotas prevent one service from starving others. Network policies limit blast radius. Separate node pools for separate failure domains. The migration Job is a separate pipeline stage, not an init container that blocks the rollout. |
| Amplify | Probe failures are visible in monitoring, not just in `kubectl describe`. Rollout stalls are detected and alerted. Container restart counts are a monitored metric. Failed Jobs produce notifications, not silent `BackoffLimitExceeded` events. |

### Observability (Datadog, Prometheus, OpenTelemetry)

What's measured, what's alerted on, what's dashboarded.

| Question | What to look for |
|---|---|
| Slowify | Alert thresholds are tested against historical data, not guessed. Dashboards are built before the feature ships, not after the first incident. Runbooks exist for every alert. Someone has verified that the pager actually fires. |
| Simplify | Monitors are scoped to services. One misconfigured monitor can't silence alerts for the entire platform. Alert routing sends the right signal to the right team. Dashboards separate business metrics from infrastructure metrics. |
| Amplify | This is the meta-layer: the observability of the observability. Is there a monitor that checks whether the agent is running? Is there an alert that fires when the alert pipeline itself is broken? GitLab's backup had been failing silently for weeks. The monitoring system is the last place you can afford a silent failure. |

### CI/CD pipelines (GitHub Actions, ArgoCD, Flux)

The deployment contract — what gates exist between a commit and production.

| Question | What to look for |
|---|---|
| Slowify | There is a gate between CI and production. It might be a manual approval, a canary stage, or a progressive rollout — but it exists. The pipeline itself is tested (pipeline-as-code, not click-ops). Changes to the pipeline are reviewed like application changes. |
| Simplify | A bad pipeline definition can only affect its own service. Shared pipeline templates have versioned interfaces. A failure in one service's deployment can't block or break another service's deployment. |
| Amplify | Failed gates block the deployment, not post a warning that gets ignored. Pipeline failures are reported to the team, not just logged. Deployment frequency, lead time, and failure rate are tracked — these are the DORA metrics, and they're amplification signals about the deployment process itself. |

### Secrets management (Vault, AWS Secrets Manager, SOPS)

How credentials are provisioned, rotated, and scoped.

| Question | What to look for |
|---|---|
| Slowify | Secret rotation is automated and tested. The rotation process has been exercised before it's needed in an emergency. New secrets are provisioned through a reviewed process, not ad hoc. |
| Simplify | Secrets are scoped to the service that needs them. One compromised credential can't access unrelated systems. Dev and production use different credentials. Rotation of one secret doesn't require restarting unrelated services. |
| Amplify | Secret access is audited. Anomalous access patterns trigger alerts. Expiring credentials produce warnings before they expire, not errors after. Failed rotations are reported immediately. |


## Using the framework

This isn't a checklist you complete once. It's a lens you apply continuously.

When you add a feature, ask the three questions at every layer the feature touches. A new endpoint touches application code, maybe a migration, maybe a new Terraform resource, definitely the CI pipeline, probably an alert rule. Each layer gets the same three questions.

When you review an incident, trace which question wasn't asked. Knight Capital: slowify wasn't asked at the deployment layer (no staging gate). GitLab: amplify wasn't asked at the observability layer (backup failures were silent). CrowdStrike: slowify wasn't asked at the CI/CD layer (no canary, no staged rollout). AWS DynamoDB: simplify wasn't asked at the infrastructure layer (DNS failure cascaded to every dependent service).

When you evaluate a team's maturity, count the layers where all three questions have answers. A team that has comprehensive application tests but no Terraform plan review and no alert threshold validation is strong on one layer and exposed on two others.

The three questions are fractal. They apply at the level of a single PR ("did I test this migration?"), at the level of a release ("can this rollout be stopped safely?"), and at the level of an organization ("do our deployment practices reward caution or speed?").


## The acceleration problem

AI is increasing the rate of change. Code generation, automated refactoring, infrastructure-as-code generation — the bottleneck is shifting from "can we write it" to "can we evaluate it." When a developer can generate a Terraform module, a migration, a handler, and a Helm chart in an afternoon, the question is no longer whether the code is correct in isolation. It's whether the full set of artifacts, across all layers, is fit for the ecosystem.

Higher rates of change demand stronger answers to all three questions. If you're deploying more often, you need more rehearsal (slowify), tighter blast radii (simplify), and faster feedback (amplify). The three questions don't change. The urgency of answering them does.

The front-line workers — the SREs, the on-call engineers, the people who debug production at 3am — have always been the ones who understand the ecosystem. They know which Terraform state file is fragile, which alert is noisy, which migration path hasn't been tested. As AI accelerates the rate of change, the value of that ecosystem awareness increases, not decreases. The people who understand the ecosystem are the people who can evaluate whether the AI's output is safe to ship.

Kim and Spear's deepest insight is that the wiring of an organization determines whether it learns from problems or repeats them. The three questions are how you audit the wiring. The twelve disciplines are how you wire the application. The ecosystem artifacts are how you wire the application's participation in the world. And the humans who ask "have we practiced this?" are the ones who keep the whole thing honest.


---

*See also: [twelve-disciplines.md](twelve-disciplines.md) for the application-internal instantiation of these principles. [slowify-simplify-amplify.md](slowify-simplify-amplify.md) for the organizational context and the historical record.*
