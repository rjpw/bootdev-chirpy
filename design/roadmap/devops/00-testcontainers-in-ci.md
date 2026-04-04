# Running Testcontainers in CI

Your integration tests use testcontainers-go to spin up a real Postgres in Docker. Locally, that just works — Docker Desktop or the Docker daemon is already there. In CI, you need to make sure a Docker daemon is available to the test process.

This doc covers the options, from simplest to most involved.


## GitHub Actions

Works out of the box. The default `ubuntu-latest` runners ship with Docker installed and the daemon running. No configuration beyond the test command itself:

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Fast tests
        run: go test -race ./...
      - name: Integration tests
        run: go test -race -tags integration -count=1 ./...
```

Testcontainers detects the GitHub Actions environment automatically and adjusts its behavior (resource reaper settings, timeouts).

**Pros:** Zero setup. Free for public repos, generous minutes for private.
**Cons:** You're on GitHub's infrastructure. Runner specs are fixed (2 vCPU, 7 GB RAM for the free tier).


## Jenkins on a VM or bare metal

If your Jenkins agents run on a machine with Docker installed, it works the same as local development. The Jenkins user needs to be in the `docker` group (or use `sudo`).

```groovy
pipeline {
    agent any
    stages {
        stage('Test') {
            steps {
                sh 'go test -race -tags integration -count=1 ./...'
            }
        }
    }
}
```

**Pros:** Simple. Full control over the machine.
**Cons:** You're managing Jenkins agents and Docker installations yourself.


## Jenkins on Kubernetes (EKS, GKE, etc.)

Kubernetes pods don't have a Docker daemon by default. You need to provide one. Three approaches, in order of increasing operational complexity:


### Option 1: Docker-in-Docker sidecar

Run a `docker:dind` container alongside your build container in the same pod. The build container talks to the DinD daemon over localhost.

```yaml
# Jenkins Kubernetes pod template
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: golang
      image: golang:1.23
      command: [sleep]
      args: [infinity]
      env:
        - name: DOCKER_HOST
          value: tcp://localhost:2375
    - name: dind
      image: docker:dind
      securityContext:
        privileged: true
      env:
        - name: DOCKER_TLS_CERTDIR
          value: ""
```

The `DOCKER_TLS_CERTDIR=""` disables TLS between the sidecar and the build container (they share a pod network, so TLS is unnecessary). Port 2375 is the unencrypted Docker API.

**Requirements:**
- The namespace must allow privileged pods. Check Pod Security Standards: `kubectl get ns <ns> -o yaml` and look for `pod-security.kubernetes.io/enforce` labels. You need the `privileged` profile.
- EC2-backed nodes (managed node groups or self-managed). **Fargate does not support privileged containers.**

**Pros:** Well-understood pattern. Testcontainers documents it. No node-level changes.
**Cons:** `privileged: true` gives the pod full host kernel access. Security teams may push back.


### Option 2: Sysbox runtime

Sysbox is an OCI runtime (replaces `runc`) that lets you run Docker-in-Docker without `privileged: true`. It virtualizes the kernel features that the inner Docker daemon needs, keeping the outer container unprivileged.

**Setup:**
1. Install Sysbox on your nodes (DaemonSet or custom AMI)
2. Create a RuntimeClass:
   ```yaml
   apiVersion: node.k8s.io/v1
   kind: RuntimeClass
   metadata:
     name: sysbox-runc
   handler: sysbox-runc
   ```
3. Reference it in your CI pod:
   ```yaml
   spec:
     runtimeClassName: sysbox-runc
     containers:
       - name: build
         image: docker:latest
         # No privileged: true
   ```

**Requirements:**
- Linux kernel 5.5+ (ideally 5.12+). Amazon Linux 2023 and recent Ubuntu AMIs qualify.
- Node-level installation — you'll likely need a dedicated node group for Sysbox-enabled nodes.
- Sysbox is open source (Sysbox-CE), originally by Nestybox, now maintained under Docker Inc.

**Pros:** No privileged pods. Stronger isolation than DinD.
**Cons:** Operational overhead — custom AMI or DaemonSet installer, dedicated node pool, kernel version requirements. Development pace has slowed since the Docker acquisition; check release activity before committing.


### Option 3: Testcontainers Cloud

A managed service from the Testcontainers team. Instead of running Docker locally, your tests connect to a remote Docker daemon hosted by Testcontainers Cloud.

**Setup:**
1. Sign up at [testcontainers.cloud](https://testcontainers.cloud)
2. Add the Testcontainers Cloud agent as a sidecar or init container
3. Set the `TC_CLOUD_TOKEN` environment variable

Your test code doesn't change. The testcontainers library detects the cloud agent and routes container operations to the remote daemon.

**Pros:** No Docker daemon needed in the pod at all. No privileged containers. No node-level changes. Works on Fargate.
**Cons:** Paid service. Adds network latency between your tests and the database container. External dependency for your CI pipeline.


## Comparison

| Approach | Docker on node? | Privileged? | Node changes? | Cost |
|----------|----------------|-------------|---------------|------|
| GitHub Actions | Pre-installed | N/A | N/A | Free / included |
| Jenkins on VM | You install it | No (docker group) | No | Your infra |
| DinD sidecar | No | Yes | No | Your infra |
| Sysbox | No | No | Yes (runtime) | Your infra |
| TC Cloud | No | No | No | Subscription |


## Recommendations

**Starting out or small team:** GitHub Actions. It works immediately and you can focus on writing tests instead of managing infrastructure.

**Already on Jenkins/K8s with relaxed security:** DinD sidecar. It's the simplest Kubernetes option and well-documented. Isolate CI workloads to a dedicated node pool if your security team wants a boundary.

**Jenkins/K8s with strict security policies:** Evaluate Testcontainers Cloud first (lowest operational burden), then Sysbox if you need to keep everything in-house.

**Fargate:** DinD and Sysbox are both off the table. Use Testcontainers Cloud, or run integration tests on EC2-backed nodes only.


## What doesn't change

Regardless of which CI environment you choose, your Go test code is identical:

```bash
go test -race -tags integration -count=1 ./...
```

The `internal/testdb` package calls `testcontainers-go`, which connects to whatever Docker daemon is available (local socket, DinD over TCP, or Testcontainers Cloud). Your tests don't know or care which one it is. That's the point of the abstraction.
