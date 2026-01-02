# Mutex

Mutex provides mutual exclusion for critical sections in Kubernetes workloads.

## Overview

A Mutex ensures that only one holder can execute a critical section at a time. Unlike Lease, Mutex requires explicit unlock and is simpler for basic mutual exclusion scenarios.

```
┌──────────┐     ┌─────────┐     ┌──────────┐
│  Pod A   │────▶│  Mutex  │     │  Pod B   │
│ (holder) │     │ Holder: │     │(waiting) │
│          │     │  Pod A  │     │          │
└──────────┘     │ Locked  │     └──────────┘
                 └─────────┘
                      │
              [Unlock → Available]
```

## Key Differences from Lease

| Feature | Mutex | Lease |
|---------|-------|-------|
| Unlock | Explicit | Automatic (TTL) |
| TTL | Optional | Required |
| Use Case | Simple locks | Singleton jobs |
| Complexity | Simpler | More features |

## Spec

```yaml
apiVersion: sync.konductor.io/v1
kind: Mutex
metadata:
  name: my-mutex
spec:
  # Optional: TTL for automatic unlock
  ttl: 10m
```

## Status

```yaml
status:
  holder: pod-abc-123
  phase: Locked  # or Unlocked
  lockedAt: "2024-01-01T12:00:00Z"
  expiresAt: "2024-01-01T12:10:00Z"  # if TTL is set
```

## CLI Usage

### Create Mutex

```bash
# Create mutex without TTL (manual unlock required)
koncli mutex create critical-section

# Create mutex with TTL (auto-unlock after 5 minutes)
koncli mutex create critical-section --ttl 5m
```

### Lock Mutex

```bash
# Lock with default holder (hostname)
koncli mutex lock critical-section

# Lock with custom holder
koncli mutex lock critical-section --holder worker-1

# Lock with timeout (wait up to 30s)
koncli mutex lock critical-section --timeout 30s
```

### Unlock Mutex

```bash
# Unlock (must be the holder)
koncli mutex unlock critical-section --holder worker-1
```

### List Mutexes

```bash
koncli mutex list
```

## SDK Usage

### Basic Lock/Unlock

```go
import (
    konductor "github.com/LogicIQ/konductor/sdk/go"
)

client, _ := konductor.New(nil)

// Lock
mutex, err := konductor.MutexLock(client, ctx, "critical-section")
if err != nil {
    log.Fatal(err)
}

// Critical section
performCriticalWork()

// Unlock
mutex.Unlock()
```

### Try Lock (Non-blocking)

```go
mutex, err := konductor.MutexTryLock(client, ctx, "critical-section")
if err != nil {
    log.Println("Lock not available")
    return
}
defer mutex.Unlock()

performCriticalWork()
```

### With Pattern (Auto-unlock)

```go
err := konductor.MutexWith(client, ctx, "critical-section", func() error {
    return performCriticalWork()
})
```

## Use Cases

### 1. Database Migration Coordination

Ensure only one pod runs migrations:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migration
spec:
  template:
    spec:
      initContainers:
      - name: acquire-lock
        image: logiciq/koncli:latest
        command:
        - koncli
        - mutex
        - lock
        - db-migration
        - --timeout
        - "5m"
      containers:
      - name: migrate
        image: my-app:latest
        command: ["./migrate"]
      - name: release-lock
        image: logiciq/koncli:latest
        command:
        - koncli
        - mutex
        - unlock
        - db-migration
```

### 2. Shared Resource Access

Serialize access to a shared resource:

```go
err := konductor.MutexWith(client, ctx, "shared-file", func() error {
    // Only one pod can access the file at a time
    return writeToSharedFile(data)
})
```

### 3. Critical Section in CronJob

Prevent concurrent executions:

```bash
#!/bin/bash
if koncli mutex lock daily-job --holder $HOSTNAME --timeout 0; then
    run-daily-job
    koncli mutex unlock daily-job --holder $HOSTNAME
else
    echo "Job already running, skipping"
fi
```

## Best Practices

1. **Always unlock**: Use defer or ensure unlock in error paths
2. **Use TTL for safety**: Prevents deadlocks if holder crashes
3. **Unique holders**: Use hostname or pod name for identification
4. **Timeout on lock**: Avoid indefinite waiting
5. **Short critical sections**: Minimize lock hold time

## Comparison with Other Primitives

**Use Mutex when:**
- Simple mutual exclusion needed
- Explicit control over unlock timing
- No complex coordination required

**Use Lease when:**
- Automatic expiration required
- Singleton job execution
- Priority-based acquisition needed

**Use Semaphore when:**
- Multiple concurrent holders allowed
- Rate limiting needed

**Use Barrier when:**
- Coordinating multiple processes
- Waiting for group completion
