# konductor

![Konductor](https://raw.githubusercontent.com/LogicIQ/konductor/main/docs/images/konductor.webp)

Kubernetes operator for workflow coordination and job orchestration. Synchronize Kubernetes Jobs, coordinate multi-stage pipelines, and manage complex workflows in your cluster.

🌐 **[Visit Project Page](https://logiciq.ca/konductor)**

## Why Konductor?

Kubernetes Jobs are powerful but lack built-in coordination. When you need to:
- Wait for multiple Jobs to complete before starting the next stage
- Prevent CronJobs from overlapping when they run longer than their schedule
- Ensure only one Job runs database migrations across multiple replicas
- Limit how many batch Jobs run concurrently to avoid overwhelming your cluster
- Coordinate Pods with each other using CLI or SDK

Konductor provides simple primitives to solve these problems natively in Kubernetes.

**Native Kubernetes Integration**
- CRDs for declarative workflow definition
- Works seamlessly with Jobs, CronJobs, and Pods
- No external dependencies or services required

**Simple and Lightweight**
- Single operator deployment
- Minimal resource overhead
- Easy to understand primitives

**Flexible Usage**
- **CLI** for shell scripts and initContainers
- **SDK** for application-level integration
- **kubectl** for manual operations

**Production Ready**
- Automatic cleanup and TTL expiration
- Leader election for HA operator
- Comprehensive observability

## Features

- **Barrier** - Synchronize multiple Jobs at coordination points
- **Gate** - Wait for dependencies before starting Jobs
- **Lease** - Singleton Job execution and leader election
- **Semaphore** - Control concurrent Job execution
- **CLI** - Command-line tool for workflow management
- **SDK** - Go SDK for programmatic integration

## Synchronization Primitives

### Barrier
Synchronize multiple processes at a coordination point.

```
┌──────────┐     ┌──────────┐     ┌──────────┐
│  Pod A   │────▶│ Barrier  │◀────│  Pod B   │
│ (waiting)│     │ Expected:│     │ (waiting)│
└──────────┘     │    3     │     └──────────┘
                 │ Arrived: │
┌──────────┐     │    2     │
│  Pod C   │────▶│          │
│(arriving)│     └──────────┘
└──────────┘          │
                      ▼
              [All arrive → Open]
```

**Use Cases:**
- Multi-stage ETL pipelines (wait for all extractors before transforming)
- Coordinated batch job execution
- Distributed testing (wait for all services before running tests)
- MapReduce-style workflows

### Gate
Wait for multiple conditions before proceeding.

```
┌─────────────┐
│    Gate     │  Conditions:
│ (workflow)  │  ✓ Job "etl" = Complete
└─────────────┘  ✓ Job "validation" = Complete
       │         ✗ Barrier "workers" = Open
       │
       ▼
  [All met → Open]
```

**Use Cases:**
- Job dependency management (Job B waits for Job A)
- Complex workflow orchestration
- Deployment gates (wait for validation before deploy)
- Multi-job coordination

### Lease
Singleton execution and leader election with automatic expiration.

```
┌──────────┐     ┌─────────┐     ┌──────────┐
│  Pod A   │────▶│  Lease  │     │  Pod B   │
│ (leader) │     │ Holder: │     │(standby) │
│          │     │  Pod A  │     │          │
└──────────┘     │ TTL: 30s│     └──────────┘
                 └─────────┘
                      │
              [Expires → Available]
```

**Use Cases:**
- Singleton CronJobs (prevent overlapping executions)
- Database migration coordination
- One-time initialization Jobs
- Leader election for distributed Jobs

### Semaphore
Control concurrent Job execution.

```
┌─────────────┐
│  Semaphore  │  Permits: 3
│(batch-jobs) │  In-Use: 2
└─────────────┘  Available: 1
      │
      ├─── [Permit 1] → Job A (running)
      ├─── [Permit 2] → Job B (running)
      └─── [Permit 3] → Available
```

**Use Cases:**
- Limit concurrent batch Jobs (e.g., max 10 Jobs at once)
- Resource-constrained Job execution
- Throttle parallel Job processing
- Control cluster load from Jobs

## Installation

### Helm

```bash
# Add the LogicIQ Helm repository
helm repo add logiciq https://logiciq.github.io/helm-charts
helm repo update

# Install konductor
helm install my-konductor logiciq/konductor

# Install with custom values
helm install my-konductor logiciq/konductor -f values.yaml
```

## Quick Start

### Using kubectl

```bash
# Create a barrier for 3 Jobs
kubectl apply -f - <<EOF
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: stage-1-complete
spec:
  expected: 3
EOF

# Check status
kubectl get barrier stage-1-complete
```

### Using CLI

```bash
# Install CLI
go install github.com/LogicIQ/konductor/cli@latest

# Wait for barrier in Job
koncli barrier wait stage-1-complete --timeout 30m
```

### Using SDK

```go
import konductor "github.com/LogicIQ/konductor/sdk/go"

client, _ := konductor.New(nil)

// Wait for dependencies
client.WaitGate(ctx, "dependencies-ready")

// Signal completion
client.ArriveBarrier(ctx, "stage-complete")
```

## Usage Examples

### Multi-Stage ETL Pipeline

```yaml
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: stage-1-complete
spec:
  expected: 3
---
apiVersion: batch/v1
kind: Job
metadata:
  name: stage-2-processor
spec:
  template:
    spec:
      initContainers:
      - name: wait-stage-1
        image: logiciq/koncli:latest
        command: ["koncli", "barrier", "wait", "stage-1-complete"]
      containers:
      - name: processor
        image: my-processor:latest
```

### Job Dependency with Gate

```yaml
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: validation-gate
spec:
  conditions:
  - type: Job
    name: data-validation
    state: Complete
---
apiVersion: batch/v1
kind: Job
metadata:
  name: data-processing
spec:
  template:
    spec:
      initContainers:
      - name: wait-validation
        image: logiciq/koncli:latest
        command: ["koncli", "gate", "wait", "validation-gate"]
      containers:
      - name: processor
        image: my-processor:latest
```

### Singleton CronJob

```yaml
apiVersion: konductor.io/v1
kind: Lease
metadata:
  name: daily-report
spec:
  ttl: 1h
---
apiVersion: batch/v1
kind: CronJob
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: report
            image: my-report:latest
            command:
            - /bin/sh
            - -c
            - |
              if koncli lease acquire daily-report --holder $HOSTNAME; then
                generate-report
                koncli lease release daily-report --holder $HOSTNAME
              fi
```

## Documentation

- **[CLI Documentation](./cli/README.md)** - Command-line tool usage and examples
- **[SDK Documentation](./sdk/go/README.md)** - Go SDK integration guide
- **[CLI Examples](./cli/examples/README.md)** - Real-world CLI usage scenarios

## Real-World Scenarios

### Scenario 1: ETL Pipeline with Stages

**Problem:** 10 extract Jobs must complete before 5 transform Jobs can start.

**Solution:**
```yaml
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: extract-complete
spec:
  expected: 10
```

Extract Jobs signal completion, transform Jobs wait for barrier.

### Scenario 2: Prevent Overlapping CronJobs

**Problem:** Daily report CronJob takes 2 hours but runs every hour.

**Solution:**
```bash
if koncli lease acquire daily-report --holder $HOSTNAME --timeout 0; then
  generate-report
  koncli lease release daily-report --holder $HOSTNAME
else
  echo "Previous job still running, skipping"
fi
```

### Scenario 3: Job Dependencies

**Problem:** Processing Job needs validation Job and cleanup Job to complete first.

**Solution:**
```yaml
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: processing-gate
spec:
  conditions:
  - type: Job
    name: validation-job
    state: Complete
  - type: Job
    name: cleanup-job
    state: Complete
```

### Scenario 4: Limit Concurrent Batch Jobs

**Problem:** 100 batch Jobs would overwhelm cluster resources.

**Solution:**
```yaml
apiVersion: konductor.io/v1
kind: Semaphore
metadata:
  name: batch-limit
spec:
  permits: 10  # Max 10 concurrent Jobs
```

Each Job acquires permit before starting, releases on completion.

## Architecture

```
┌─────────────────────────────────────────────┐
│           Kubernetes Cluster                │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │     Konductor Operator               │  │
│  │  (Watches CRDs, Manages State)       │  │
│  └──────────────────────────────────────┘  │
│         │         │         │         │     │
│         ▼         ▼         ▼         ▼     │
│  ┌─────────┐┌─────────┐┌──────┐┌──────┐   │
│  │Semaphore││ Barrier ││Lease ││ Gate │   │
│  │   CRD   ││   CRD   ││ CRD  ││ CRD  │   │
│  └─────────┘└─────────┘└──────┘└──────┘   │
│         ▲         ▲         ▲         ▲     │
│         │         │         │         │     │
│  ┌──────┴─────────┴─────────┴─────────┴──┐ │
│  │                                        │ │
│  │  ┌────────┐  ┌────────┐  ┌─────────┐ │ │
│  │  │  Pods  │  │  CLI   │  │   SDK   │ │ │
│  │  └────────┘  └────────┘  └─────────┘ │ │
│  │                                        │ │
│  └────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

## Contributing

Contributions welcome! Please read our contributing guidelines.

## License

Apache 2.0
