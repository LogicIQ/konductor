# Semaphore API

The Semaphore resource controls concurrent access to shared resources by limiting the number of permits available.

## Resource Definition

```yaml
apiVersion: konductor.io/v1
kind: Semaphore
metadata:
  name: batch-limit
  namespace: default
spec:
  permits: 10
  ttl: 30m
status:
  inUse: 3
  available: 7
  phase: Ready
  conditions:
  - type: Ready
    status: "True"
    reason: SemaphoreReady
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `permits` | integer | Yes | Maximum number of concurrent permits |
| `ttl` | duration | No | Time-to-live for individual permits (default: 5m) |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `inUse` | integer | Number of permits currently in use |
| `available` | integer | Number of permits available for acquisition |
| `phase` | string | Current phase: `Ready`, `NotReady` |
| `holders` | []string | List of current permit holders |

## Phases

- **Ready**: Semaphore is operational and can grant permits
- **NotReady**: Semaphore is not ready (initialization, errors)

## Examples

### Basic Semaphore

```yaml
apiVersion: konductor.io/v1
kind: Semaphore
metadata:
  name: api-quota
spec:
  permits: 5
  ttl: 10m
```

### Job with Semaphore

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: batch-processor
spec:
  template:
    spec:
      initContainers:
      - name: acquire-permit
        image: logiciq/koncli:latest
        command:
        - koncli
        - semaphore
        - acquire
        - api-quota
        - --wait
        - --ttl=10m
      containers:
      - name: processor
        image: my-processor:latest
        command:
        - /bin/sh
        - -c
        - |
          process-data
          koncli semaphore release api-quota
```

## CLI Usage

```bash
# Acquire a permit
koncli semaphore acquire api-quota --ttl=5m

# Acquire with wait
koncli semaphore acquire api-quota --wait --timeout=30m

# Release a permit
koncli semaphore release api-quota

# Check status
koncli semaphore status api-quota
```

## Use Cases

### Batch Job Throttling
Limit concurrent batch jobs to prevent resource exhaustion:

```yaml
apiVersion: konductor.io/v1
kind: Semaphore
metadata:
  name: batch-limit
spec:
  permits: 10  # Max 10 concurrent jobs
```

### API Rate Limiting
Control external API calls:

```yaml
apiVersion: konductor.io/v1
kind: Semaphore
metadata:
  name: external-api
spec:
  permits: 3   # Max 3 concurrent API calls
  ttl: 1m      # Short TTL for API calls
```

### Database Connection Pool
Limit database connections:

```yaml
apiVersion: konductor.io/v1
kind: Semaphore
metadata:
  name: db-connections
spec:
  permits: 20  # Max 20 concurrent connections
  ttl: 15m     # Connection timeout
```

## Best Practices

1. **Set appropriate TTL**: Use shorter TTL for quick operations, longer for batch jobs
2. **Monitor usage**: Check `inUse` and `available` fields regularly
3. **Handle failures**: Always release permits in error cases
4. **Use initContainers**: Preferred pattern for job gating
5. **Namespace isolation**: Use different namespaces for different environments

## Troubleshooting

### Permits Not Available
```bash
# Check current usage
kubectl describe semaphore my-semaphore

# List current holders
kubectl get semaphore my-semaphore -o jsonpath='{.status.holders}'
```

### Stuck Permits
Permits automatically expire based on TTL. To force cleanup:

```bash
# Delete and recreate semaphore
kubectl delete semaphore my-semaphore
kubectl apply -f semaphore.yaml
```

## Related Resources

- [Barrier API](./barrier.md) - Multi-stage coordination
- [CLI Reference](../cli/semaphore.md) - Detailed CLI usage
- [Examples](../examples/batch-processing.md) - Real-world examples