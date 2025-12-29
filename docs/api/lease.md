# Lease API

The Lease resource provides singleton execution and leader election with automatic expiration.

## Resource Definition

```yaml
apiVersion: konductor.io/v1
kind: Lease
metadata:
  name: db-migration
  namespace: default
spec:
  ttl: 10m
  priority: 1
status:
  holder: pod-xyz-123
  acquired: "2024-01-15T10:30:00Z"
  expires: "2024-01-15T10:40:00Z"
  phase: Held
  conditions:
  - type: Ready
    status: "True"
    reason: LeaseReady
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ttl` | duration | Yes | Time-to-live for the lease |
| `priority` | integer | No | Priority for lease acquisition (higher wins) |
| `renewable` | boolean | No | Whether lease can be renewed (default: true) |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `holder` | string | Current lease holder identifier |
| `acquired` | timestamp | When the lease was acquired |
| `expires` | timestamp | When the lease expires |
| `phase` | string | Current phase: `Available`, `Held`, `Expired` |
| `renewals` | integer | Number of times lease has been renewed |

## Phases

- **Available**: Lease is available for acquisition
- **Held**: Lease is currently held by a process
- **Expired**: Lease has expired and will be cleaned up

## Examples

### Basic Lease

```yaml
apiVersion: konductor.io/v1
kind: Lease
metadata:
  name: singleton-job
spec:
  ttl: 30m
```

### Singleton CronJob

```yaml
apiVersion: konductor.io/v1
kind: Lease
metadata:
  name: daily-report
spec:
  ttl: 2h  # Report takes max 2 hours
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: daily-report-job
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
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
              if koncli lease acquire daily-report --holder $HOSTNAME --timeout 0; then
                echo "Acquired lease, generating report..."
                generate-report
                koncli lease release daily-report --holder $HOSTNAME
              else
                echo "Previous job still running, skipping"
                exit 0
              fi
```

### Database Migration

```yaml
apiVersion: konductor.io/v1
kind: Lease
metadata:
  name: db-migration
spec:
  ttl: 15m
  priority: 10  # High priority for migrations
---
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate-db
spec:
  parallelism: 3  # Multiple pods, but only one will migrate
  template:
    spec:
      containers:
      - name: migrate
        image: my-migrator:latest
        command:
        - /bin/sh
        - -c
        - |
          if koncli lease acquire db-migration --holder $HOSTNAME --wait --timeout=5m; then
            echo "Running migration..."
            run-migrations
            koncli lease release db-migration --holder $HOSTNAME
          else
            echo "Migration already completed by another pod"
          fi
```

## CLI Usage

```bash
# Acquire lease
koncli lease acquire db-migration --holder $HOSTNAME --ttl=10m

# Acquire with wait
koncli lease acquire db-migration --holder $HOSTNAME --wait --timeout=5m

# Renew lease
koncli lease renew db-migration --holder $HOSTNAME

# Release lease
koncli lease release db-migration --holder $HOSTNAME

# Check status
koncli lease status db-migration
```

## Use Cases

### Singleton CronJobs
Prevent overlapping executions of long-running CronJobs:

```yaml
apiVersion: konductor.io/v1
kind: Lease
metadata:
  name: backup-job
spec:
  ttl: 4h  # Backup takes max 4 hours
```

### Database Migrations
Ensure only one migration runs across multiple replicas:

```yaml
apiVersion: konductor.io/v1
kind: Lease
metadata:
  name: schema-migration
spec:
  ttl: 30m
  priority: 100  # Highest priority
```

### Leader Election
Elect a leader among multiple instances:

```yaml
apiVersion: konductor.io/v1
kind: Lease
metadata:
  name: service-leader
spec:
  ttl: 30s      # Short TTL for quick failover
  renewable: true
```

### One-Time Initialization
Ensure initialization runs only once:

```yaml
apiVersion: konductor.io/v1
kind: Lease
metadata:
  name: app-init
spec:
  ttl: 10m
  renewable: false  # Cannot be renewed
```

## Best Practices

1. **Set appropriate TTL**: Balance between safety and availability
2. **Use unique holders**: Include pod name or unique identifier
3. **Handle failures gracefully**: Always release leases in cleanup
4. **Monitor expiration**: Set up alerts for lease expiration
5. **Use priority for critical tasks**: Higher priority for important operations

## Troubleshooting

### Lease Stuck
```bash
# Check current holder
kubectl get lease my-lease -o jsonpath='{.status.holder}'

# Check expiration time
kubectl get lease my-lease -o jsonpath='{.status.expires}'

# Force release (if holder is dead)
kubectl patch lease my-lease --type='merge' -p='{"status":{"phase":"Available","holder":""}}'
```

### Acquisition Failures
```bash
# Check lease status
kubectl describe lease my-lease

# Check if holder is still alive
kubectl get pods | grep <holder-name>
```

### Priority Conflicts
```bash
# Check current priority
kubectl get lease my-lease -o jsonpath='{.spec.priority}'

# Update priority if needed
kubectl patch lease my-lease --type='merge' -p='{"spec":{"priority":50}}'
```

## Advanced Patterns

### Lease Renewal Loop
For long-running processes:

```bash
#!/bin/bash
LEASE_NAME="my-service-leader"
HOLDER_ID="$HOSTNAME-$$"

# Acquire initial lease
if koncli lease acquire $LEASE_NAME --holder $HOLDER_ID --ttl=30s; then
  # Start renewal loop in background
  while true; do
    sleep 15  # Renew every 15 seconds
    if ! koncli lease renew $LEASE_NAME --holder $HOLDER_ID; then
      echo "Lost lease, shutting down"
      exit 1
    fi
  done &
  
  # Run main process
  run-main-process
  
  # Cleanup
  koncli lease release $LEASE_NAME --holder $HOLDER_ID
fi
```

### Graceful Handover
Transfer lease between processes:

```bash
# Current holder releases
koncli lease release my-lease --holder $OLD_HOLDER

# New holder acquires
koncli lease acquire my-lease --holder $NEW_HOLDER --ttl=10m
```

## Related Resources

- [Semaphore API](./semaphore.md) - Concurrent access control
- [Barrier API](./barrier.md) - Multi-stage coordination
- [CLI Reference](../cli/lease.md) - Detailed CLI usage
- [Examples](../examples/database-migrations.md) - Migration examples