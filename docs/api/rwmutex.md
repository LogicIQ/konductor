# RWMutex API

The RWMutex resource provides read-write locks allowing multiple concurrent readers or a single exclusive writer.

## Resource Definition

```yaml
apiVersion: konductor.io/v1
kind: RWMutex
metadata:
  name: cache-lock
  namespace: default
spec:
  ttl: 5m
status:
  writeHolder: ""
  readHolders:
  - reader-1
  - reader-2
  lockedAt: "2024-01-15T10:30:00Z"
  expiresAt: "2024-01-15T10:35:00Z"
  phase: ReadLocked
  conditions:
  - type: Ready
    status: "True"
    reason: RWMutexReady
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ttl` | duration | No | Time-to-live for automatic unlock |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `writeHolder` | string | Current write lock holder (empty if read locked) |
| `readHolders` | []string | List of current read lock holders |
| `lockedAt` | timestamp | When the lock was acquired |
| `expiresAt` | timestamp | When the lock expires (if TTL set) |
| `phase` | string | Current phase: `Unlocked`, `ReadLocked`, `WriteLocked` |

## Phases

- **Unlocked**: No locks held, available for read or write
- **ReadLocked**: One or more read locks held, write blocked
- **WriteLocked**: Write lock held, all other locks blocked

## Examples

### Basic RWMutex

```yaml
apiVersion: konductor.io/v1
kind: RWMutex
metadata:
  name: cache-lock
spec:
  ttl: 5m
```

### Cache Coordination

```yaml
apiVersion: konductor.io/v1
kind: RWMutex
metadata:
  name: cache-lock
spec:
  ttl: 10m
---
apiVersion: batch/v1
kind: Job
metadata:
  name: cache-reader
spec:
  parallelism: 5
  template:
    spec:
      containers:
      - name: reader
        image: my-app:latest
        command:
        - /bin/sh
        - -c
        - |
          koncli rwmutex rlock cache-lock --holder $HOSTNAME
          read-from-cache
          koncli rwmutex unlock cache-lock --holder $HOSTNAME
---
apiVersion: batch/v1
kind: Job
metadata:
  name: cache-writer
spec:
  template:
    spec:
      containers:
      - name: writer
        image: my-app:latest
        command:
        - /bin/sh
        - -c
        - |
          koncli rwmutex lock cache-lock --holder $HOSTNAME --timeout 1m
          update-cache
          koncli rwmutex unlock cache-lock --holder $HOSTNAME
```

### Configuration File Access

```yaml
apiVersion: konductor.io/v1
kind: RWMutex
metadata:
  name: config-lock
spec:
  ttl: 2m
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: config-reader
spec:
  replicas: 10
  template:
    spec:
      containers:
      - name: app
        image: my-app:latest
        command:
        - /bin/sh
        - -c
        - |
          while true; do
            koncli rwmutex rlock config-lock --holder $HOSTNAME
            cat /shared/config.json
            koncli rwmutex unlock config-lock --holder $HOSTNAME
            sleep 60
          done
```

## CLI Usage

```bash
# Acquire read lock (multiple allowed)
koncli rwmutex rlock cache-lock --holder reader-1

# Acquire write lock (exclusive)
koncli rwmutex lock cache-lock --holder writer-1 --timeout 30s

# Release lock
koncli rwmutex unlock cache-lock --holder $HOSTNAME

# Create rwmutex
koncli rwmutex create cache-lock --ttl 5m

# List rwmutexes
koncli rwmutex list
```

## Use Cases

### Cache Coordination
Multiple readers, single writer for cache updates:

```yaml
apiVersion: konductor.io/v1
kind: RWMutex
metadata:
  name: app-cache
spec:
  ttl: 5m
```

### Configuration Files
Many readers, occasional writer for config updates:

```yaml
apiVersion: konductor.io/v1
kind: RWMutex
metadata:
  name: config-file
spec:
  ttl: 2m
```

### Shared Data Structures
Concurrent reads, exclusive writes:

```yaml
apiVersion: konductor.io/v1
kind: RWMutex
metadata:
  name: shared-data
spec:
  ttl: 10m
```

### Read-Heavy Workloads
Optimize for concurrent reads:

```yaml
apiVersion: konductor.io/v1
kind: RWMutex
metadata:
  name: metrics-data
spec:
  ttl: 1m
```

## Best Practices

1. **Use read locks for read-only operations**: Maximize concurrency
2. **Use write locks for modifications**: Ensure data consistency
3. **Always set TTL**: Prevent deadlocks from crashes
4. **Use unique holder IDs**: Include pod name or unique identifier
5. **Always unlock**: Use trap or defer to ensure cleanup
6. **Set appropriate timeouts**: Avoid indefinite blocking

## Troubleshooting

### RWMutex Stuck
```bash
# Check current state
kubectl get rwmutex my-rwmutex -o yaml

# Check write holder
kubectl get rwmutex my-rwmutex -o jsonpath='{.status.writeHolder}'

# Check read holders
kubectl get rwmutex my-rwmutex -o jsonpath='{.status.readHolders}'

# Verify holders are alive
kubectl get pods | grep <holder-name>
```

### Write Lock Blocked
```bash
# Check if readers are holding lock
kubectl get rwmutex my-rwmutex -o jsonpath='{.status.readHolders}'

# Check phase
kubectl get rwmutex my-rwmutex -o jsonpath='{.status.phase}'
```

### Read Lock Blocked
```bash
# Check if writer is holding lock
kubectl get rwmutex my-rwmutex -o jsonpath='{.status.writeHolder}'

# Check expiration
kubectl get rwmutex my-rwmutex -o jsonpath='{.status.expiresAt}'
```

## Advanced Patterns

### Multiple Readers Pattern
```bash
#!/bin/bash
# Multiple readers can run concurrently
for i in {1..5}; do
  (
    HOLDER="reader-$i"
    koncli rwmutex rlock data-lock --holder $HOLDER
    echo "Reader $i: reading data"
    sleep 2
    koncli rwmutex unlock data-lock --holder $HOLDER
  ) &
done
wait
```

### Read-Write Coordination
```bash
#!/bin/bash
RWMUTEX="config-lock"

# Reader function
read_config() {
  koncli rwmutex rlock $RWMUTEX --holder "reader-$HOSTNAME"
  cat /shared/config.json
  koncli rwmutex unlock $RWMUTEX --holder "reader-$HOSTNAME"
}

# Writer function
update_config() {
  koncli rwmutex lock $RWMUTEX --holder "writer-$HOSTNAME" --timeout 1m
  echo '{"updated": true}' > /shared/config.json
  koncli rwmutex unlock $RWMUTEX --holder "writer-$HOSTNAME"
}
```

### Try-Lock Pattern
```bash
#!/bin/bash
# Try to acquire write lock without blocking
if koncli rwmutex lock my-rwmutex --holder $HOSTNAME --timeout 0; then
  trap "koncli rwmutex unlock my-rwmutex --holder $HOSTNAME" EXIT
  update-data
else
  echo "Write lock busy, skipping update"
fi
```

### Upgrade from Read to Write
```bash
#!/bin/bash
RWMUTEX="data-lock"
HOLDER="$HOSTNAME"

# Acquire read lock
koncli rwmutex rlock $RWMUTEX --holder $HOLDER
data=$(read-data)

# Need to write? Release read lock first
if [ "$data" == "needs-update" ]; then
  koncli rwmutex unlock $RWMUTEX --holder $HOLDER
  
  # Acquire write lock
  if koncli rwmutex lock $RWMUTEX --holder $HOLDER --timeout 30s; then
    update-data
    koncli rwmutex unlock $RWMUTEX --holder $HOLDER
  fi
else
  koncli rwmutex unlock $RWMUTEX --holder $HOLDER
fi
```

## Comparison with Mutex

| Feature | Mutex | RWMutex |
|---------|-------|---------|
| Concurrent Reads | ❌ No | ✅ Yes |
| Exclusive Writes | ✅ Yes | ✅ Yes |
| Use Case | Simple mutual exclusion | Read-heavy workloads |
| Complexity | Lower | Higher |
| Performance | Good for writes | Better for reads |

## Related Resources

- [Mutex API](./mutex.md) - Simple mutual exclusion
- [Lease API](./lease.md) - Singleton execution
- [CLI Reference](../cli/rwmutex.md) - Detailed CLI usage
