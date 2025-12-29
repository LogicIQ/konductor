# Barrier API

The Barrier resource synchronizes multiple processes at a coordination point - all must arrive before any can proceed.

## Resource Definition

```yaml
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: stage-1-complete
  namespace: default
spec:
  expected: 5
  timeout: 30m
  quorum: 4
status:
  arrived: 3
  phase: Waiting
  conditions:
  - type: Ready
    status: "True"
    reason: BarrierReady
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `expected` | integer | Yes | Number of processes expected to arrive |
| `timeout` | duration | No | Maximum time to wait for all arrivals |
| `quorum` | integer | No | Minimum arrivals needed to open (default: expected) |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `arrived` | integer | Number of processes that have arrived |
| `phase` | string | Current phase: `Waiting`, `Open`, `Failed`, `Timeout` |
| `arrivals` | []string | List of processes that have arrived |
| `openedAt` | timestamp | When the barrier opened |

## Phases

- **Waiting**: Barrier is waiting for more arrivals
- **Open**: Barrier is open, all processes can proceed
- **Failed**: Barrier failed due to error
- **Timeout**: Barrier timed out waiting for arrivals

## Examples

### Basic Barrier

```yaml
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: extract-complete
spec:
  expected: 10
  timeout: 1h
```

### ETL Pipeline Stage

```yaml
# Stage 1: Extract jobs signal completion
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: extract-complete
spec:
  expected: 5
---
# Stage 2: Transform jobs wait for extract completion
apiVersion: batch/v1
kind: Job
metadata:
  name: transform-job
spec:
  template:
    spec:
      initContainers:
      - name: wait-extract
        image: logiciq/koncli:latest
        command:
        - koncli
        - barrier
        - wait
        - extract-complete
        - --timeout=1h
      containers:
      - name: transform
        image: my-transformer:latest
        command:
        - /bin/sh
        - -c
        - |
          transform-data
          koncli barrier arrive transform-complete
```

### Quorum-Based Barrier

```yaml
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: consensus-barrier
spec:
  expected: 10
  quorum: 7    # Only need 7 out of 10 to proceed
  timeout: 15m
```

## CLI Usage

```bash
# Wait for barrier to open
koncli barrier wait extract-complete --timeout=30m

# Signal arrival at barrier
koncli barrier arrive extract-complete

# Check barrier status
koncli barrier status extract-complete
```

## Use Cases

### Multi-Stage ETL Pipeline
Coordinate extract → transform → load stages:

```yaml
# Extract stage barrier
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: extract-complete
spec:
  expected: 5  # 5 extract jobs
---
# Transform stage barrier  
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: transform-complete
spec:
  expected: 3  # 3 transform jobs
```

### Distributed Testing
Wait for all services before running tests:

```yaml
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: services-ready
spec:
  expected: 4    # 4 services
  timeout: 10m   # Services must start within 10m
```

### MapReduce Coordination
Coordinate map and reduce phases:

```yaml
# Map phase completion
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: map-complete
spec:
  expected: 20  # 20 map tasks
  quorum: 18    # Allow 2 failures
```

## Best Practices

1. **Set realistic timeouts**: Account for slowest expected process
2. **Use quorum for fault tolerance**: Allow some processes to fail
3. **Monitor arrivals**: Check `arrived` count and `arrivals` list
4. **Handle timeouts gracefully**: Plan for timeout scenarios
5. **Clean up barriers**: Remove completed barriers to avoid confusion

## Troubleshooting

### Barrier Stuck Waiting
```bash
# Check who has arrived
kubectl get barrier my-barrier -o jsonpath='{.status.arrivals}'

# Check expected vs arrived
kubectl describe barrier my-barrier
```

### Timeout Issues
```bash
# Check barrier events
kubectl describe barrier my-barrier

# Increase timeout if needed
kubectl patch barrier my-barrier --type='merge' -p='{"spec":{"timeout":"1h"}}'
```

### Reset Barrier
```bash
# Delete and recreate to reset
kubectl delete barrier my-barrier
kubectl apply -f barrier.yaml
```

## Advanced Patterns

### Cascading Barriers
Chain multiple barriers for complex workflows:

```yaml
# Stage 1 → Stage 2 → Stage 3
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: stage-1-complete
spec:
  expected: 5
---
apiVersion: konductor.io/v1  
kind: Barrier
metadata:
  name: stage-2-complete
spec:
  expected: 3
```

### Conditional Barriers
Use different barriers based on conditions:

```bash
# In job script
if [ "$ENVIRONMENT" = "production" ]; then
  koncli barrier wait prod-validation-complete
else
  koncli barrier wait dev-validation-complete
fi
```

## Related Resources

- [Semaphore API](./semaphore.md) - Concurrent access control
- [Gate API](./gate.md) - Dependency coordination
- [CLI Reference](../cli/barrier.md) - Detailed CLI usage
- [Examples](../examples/etl-pipeline.md) - ETL pipeline examples