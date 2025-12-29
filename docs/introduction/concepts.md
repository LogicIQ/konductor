# Core Concepts

Konductor provides four synchronization primitives that solve common coordination problems in Kubernetes workloads.

## Synchronization Primitives

### Semaphore
Controls concurrent access to shared resources by limiting the number of permits available.

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
- Throttle external API calls
- Control database connections

### Barrier
Synchronizes multiple processes at a coordination point - all must arrive before any can proceed.

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
- Multi-stage ETL pipelines
- Coordinated batch job execution
- MapReduce-style workflows

### Lease
Provides singleton execution and leader election with automatic expiration.

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
- Leader election for distributed Jobs

### Gate
Waits for multiple conditions to be met before allowing processes to proceed.

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
- Job dependency management
- Complex workflow orchestration
- Deployment gates

## Integration Patterns

### InitContainer Pattern (Recommended)
Use initContainers to gate pod startup:

```yaml
initContainers:
- name: acquire-semaphore
  image: logiciq/koncli:latest
  command:
    - koncli
    - semaphore
    - acquire
    - api-quota
    - --wait
    - --ttl=10m
```

### CLI Integration
Use the CLI tool in your scripts:

```bash
# Acquire permit
if koncli semaphore acquire batch-limit --ttl=5m; then
  # Run your workload
  process-data
  # Release permit
  koncli semaphore release batch-limit
fi
```

### SDK Integration
Use the Go SDK for programmatic access:

```go
import konductor "github.com/LogicIQ/konductor/sdk/go"

client, _ := konductor.New(nil)

// Wait for dependencies
client.WaitGate(ctx, "dependencies-ready")

// Signal completion
client.ArriveBarrier(ctx, "stage-complete")
```

## Design Principles

### Kubernetes-Native
- Everything achievable via `kubectl apply`
- CRDs as primary interface
- GitOps-friendly YAML configuration
- Controller reconciliation driven

### Self-Enforced Coordination
- Operator provides state and arbitration, not control
- Pods voluntarily gate their own progress
- No external blocking or pausing of containers
- Idempotent and crash-safe operations

### Observable and Debuggable
- All coordination state visible via Kubernetes APIs
- Clear ownership and TTL semantics
- Human-readable status conditions

## Next Steps

- **[Installation Guide](../guides/installation.md)** - Install Konductor in your cluster
- **[Quick Start](../guides/quick-start.md)** - Try the basic examples
- **[API Reference](../api/overview.md)** - Detailed API documentation
- **[Examples](../examples/overview.md)** - Real-world usage patterns