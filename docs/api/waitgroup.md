# WaitGroup API

The WaitGroup resource coordinates a dynamic number of workers, similar to Go's sync.WaitGroup.

## Resource Definition

```yaml
apiVersion: konductor.io/v1
kind: WaitGroup
metadata:
  name: worker-group
  namespace: default
spec:
  ttl: 1h
status:
  counter: 3
  phase: Waiting
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ttl` | duration | No | Time-to-live for cleanup |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `counter` | int32 | Current counter value |
| `phase` | string | Current phase: `Waiting`, `Done` |

## Phases

- **Waiting**: Counter > 0, waiting for workers
- **Done**: Counter = 0, all workers complete

## Examples

### Parallel Job Processing

```yaml
apiVersion: konductor.io/v1
kind: WaitGroup
metadata:
  name: batch-jobs
spec:
  ttl: 1h
---
apiVersion: batch/v1
kind: Job
metadata:
  name: coordinator
spec:
  template:
    spec:
      containers:
      - name: coord
        image: my-app:latest
        command:
        - /bin/sh
        - -c
        - |
          # Add count
          koncli waitgroup add batch-jobs --delta 10
          
          # Start workers
          for i in {1..10}; do
            start-worker $i &
          done
          
          # Wait
          koncli waitgroup wait batch-jobs --timeout 30m
```

### Dynamic Workers

```yaml
apiVersion: konductor.io/v1
kind: WaitGroup
metadata:
  name: workers
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: worker
spec:
  replicas: 5
  template:
    spec:
      containers:
      - name: worker
        image: worker:latest
        command:
        - /bin/sh
        - -c
        - |
          koncli waitgroup add workers --delta 1
          do-work
          koncli waitgroup done workers
```

## CLI Usage

```bash
# Add to counter
koncli waitgroup add workers --delta 5

# Signal done
koncli waitgroup done workers

# Wait for completion
koncli waitgroup wait workers --timeout 10m
```

## Use Cases

### Parallel Batch Processing
Coordinate dynamic number of workers:

```yaml
apiVersion: konductor.io/v1
kind: WaitGroup
metadata:
  name: batch-process
```

### Multi-Stage Pipeline
Wait for variable worker count:

```yaml
apiVersion: konductor.io/v1
kind: WaitGroup
metadata:
  name: pipeline-stage
```

## Best Practices

1. **Add before starting**: Call Add() before launching workers
2. **Always call Done()**: Ensure Done() is called on completion
3. **Set timeouts**: Use timeout to avoid indefinite waits
4. **Use TTL**: Auto-cleanup after completion

## Troubleshooting

### Counter Not Decreasing
```bash
# Check current counter
kubectl get waitgroup my-wg -o jsonpath='{.status.counter}'

# Verify workers are calling Done()
kubectl logs <worker-pod>
```

### Wait Timeout
```bash
# Check phase
kubectl get waitgroup my-wg -o jsonpath='{.status.phase}'

# Check for stuck workers
kubectl get pods -l app=worker
```

## Comparison with Barrier

| Feature | WaitGroup | Barrier |
|---------|-----------|---------|
| Count | Dynamic | Fixed |
| Add/Done | Yes | Arrive only |
| Use Case | Variable workers | Known count |

## Related Resources

- [Barrier API](./barrier.md) - Fixed-count synchronization
- [CLI Reference](../cli/waitgroup.md) - Detailed CLI usage
