# Mutex API

The Mutex resource provides mutual exclusion for critical sections with optional automatic expiration.

## Resource Definition

```yaml
apiVersion: konductor.io/v1
kind: Mutex
metadata:
  name: db-migration
  namespace: default
spec:
  ttl: 10m
status:
  holder: pod-xyz-123
  lockedAt: "2024-01-15T10:30:00Z"
  expiresAt: "2024-01-15T10:40:00Z"
  phase: Locked
  conditions:
  - type: Ready
    status: "True"
    reason: MutexReady
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ttl` | duration | No | Time-to-live for automatic unlock |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `holder` | string | Current lock holder identifier |
| `lockedAt` | timestamp | When the mutex was locked |
| `expiresAt` | timestamp | When the mutex expires (if TTL set) |
| `phase` | string | Current phase: `Unlocked`, `Locked` |

## Phases

- **Unlocked**: Mutex is available for locking
- **Locked**: Mutex is currently held by a process

## Examples

### Basic Mutex

```yaml
apiVersion: konductor.io/v1
kind: Mutex
metadata:
  name: file-writer
spec:
  ttl: 5m
```

### Database Migration

```yaml
apiVersion: konductor.io/v1
kind: Mutex
metadata:
  name: db-migration
spec:
  ttl: 15m
---
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate-db
spec:
  parallelism: 3
  template:
    spec:
      containers:
      - name: migrate
        image: my-migrator:latest
        command:
        - /bin/sh
        - -c
        - |
          if koncli mutex lock db-migration --holder $HOSTNAME --timeout 30s; then
            trap "koncli mutex unlock db-migration --holder $HOSTNAME" EXIT
            run-migrations
          else
            echo "Migration already running"
          fi
```

### Critical Section

```yaml
apiVersion: konductor.io/v1
kind: Mutex
metadata:
  name: shared-resource
spec:
  ttl: 2m
---
apiVersion: v1
kind: Pod
metadata:
  name: worker
spec:
  containers:
  - name: app
    image: my-app:latest
    command:
    - /bin/sh
    - -c
    - |
      koncli mutex lock shared-resource --holder $HOSTNAME
      access-shared-resource
      koncli mutex unlock shared-resource --holder $HOSTNAME
```

## CLI Usage

```bash
# Lock mutex
koncli mutex lock db-migration --holder $HOSTNAME

# Lock with timeout
koncli mutex lock db-migration --holder $HOSTNAME --timeout 30s

# Unlock mutex
koncli mutex unlock db-migration --holder $HOSTNAME

# Create mutex
koncli mutex create db-migration --ttl 10m

# List mutexes
koncli mutex list
```

## Use Cases

### Database Migrations
Ensure only one migration runs across multiple replicas:

```yaml
apiVersion: konductor.io/v1
kind: Mutex
metadata:
  name: schema-migration
spec:
  ttl: 30m
```

### File Writing
Serialize writes to shared files:

```yaml
apiVersion: konductor.io/v1
kind: Mutex
metadata:
  name: log-writer
spec:
  ttl: 1m
```

### Configuration Updates
Prevent concurrent configuration changes:

```yaml
apiVersion: konductor.io/v1
kind: Mutex
metadata:
  name: config-update
spec:
  ttl: 5m
```

### Resource Initialization
Ensure initialization runs serially:

```yaml
apiVersion: konductor.io/v1
kind: Mutex
metadata:
  name: app-init
spec:
  ttl: 10m
```

## Best Practices

1. **Always set TTL**: Prevent deadlocks from crashes
2. **Use unique holders**: Include pod name or unique identifier
3. **Always unlock**: Use trap or defer to ensure cleanup
4. **Set appropriate timeouts**: Avoid indefinite blocking
5. **Handle lock failures**: Implement retry logic or skip gracefully

## Troubleshooting

### Mutex Stuck
```bash
# Check current holder
kubectl get mutex my-mutex -o jsonpath='{.status.holder}'

# Check if holder is alive
kubectl get pods | grep <holder-name>

# Check expiration
kubectl get mutex my-mutex -o jsonpath='{.status.expiresAt}'
```

### Lock Acquisition Failures
```bash
# Check mutex status
kubectl describe mutex my-mutex

# Verify holder can unlock
koncli mutex unlock my-mutex --holder <holder-name>
```

## Advanced Patterns

### Try-Lock Pattern
Non-blocking lock attempt:

```bash
#!/bin/bash
if koncli mutex lock my-mutex --holder $HOSTNAME --timeout 0; then
  trap "koncli mutex unlock my-mutex --holder $HOSTNAME" EXIT
  do-critical-work
else
  echo "Lock busy, skipping"
fi
```

### Retry with Backoff
```bash
#!/bin/bash
MUTEX="my-mutex"
HOLDER="$HOSTNAME"
MAX_RETRIES=5

for i in $(seq 1 $MAX_RETRIES); do
  if koncli mutex lock $MUTEX --holder $HOLDER --timeout 5s; then
    trap "koncli mutex unlock $MUTEX --holder $HOLDER" EXIT
    do-work
    exit 0
  fi
  echo "Retry $i/$MAX_RETRIES"
  sleep $((i * 2))
done

echo "Failed to acquire lock"
exit 1
```

## Related Resources

- [RWMutex API](./rwmutex.md) - Read-write locks
- [Lease API](./lease.md) - Singleton execution
- [CLI Reference](../cli/mutex.md) - Detailed CLI usage
