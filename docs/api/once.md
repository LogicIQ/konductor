# Once API

The Once resource ensures an action is executed exactly once across multiple pods or jobs.

## Resource Definition

```yaml
apiVersion: konductor.io/v1
kind: Once
metadata:
  name: app-init
  namespace: default
spec:
  ttl: 1h
status:
  executed: true
  executor: pod-xyz-123
  executedAt: "2024-01-15T10:30:00Z"
  phase: Executed
  conditions:
  - type: Ready
    status: "True"
    reason: OnceReady
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ttl` | duration | No | Time-to-live for cleanup |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `executed` | boolean | Whether action has been executed |
| `executor` | string | Who executed the action |
| `executedAt` | timestamp | When action was executed |
| `phase` | string | Current phase: `Pending`, `Executed` |

## Phases

- **Pending**: Action not yet executed
- **Executed**: Action has been executed

## Examples

### Database Initialization

```yaml
apiVersion: konductor.io/v1
kind: Once
metadata:
  name: db-init
spec:
  ttl: 24h
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  replicas: 3
  template:
    spec:
      initContainers:
      - name: init-db
        image: my-app:latest
        command:
        - /bin/sh
        - -c
        - |
          if ! koncli once check db-init | grep -q "has been executed"; then
            echo "Running migrations..."
            run-migrations
          fi
      containers:
      - name: app
        image: my-app:latest
```

### Configuration Setup

```yaml
apiVersion: konductor.io/v1
kind: Once
metadata:
  name: config-setup
spec:
  ttl: 1h
---
apiVersion: batch/v1
kind: Job
metadata:
  name: setup-config
spec:
  parallelism: 5
  template:
    spec:
      containers:
      - name: setup
        image: my-app:latest
        command:
        - /bin/sh
        - -c
        - |
          if ! koncli once check config-setup | grep -q "has been executed"; then
            write-default-config
          fi
```

### Resource Provisioning

```yaml
apiVersion: konductor.io/v1
kind: Once
metadata:
  name: provision-s3
spec:
  ttl: 24h
---
apiVersion: batch/v1
kind: Job
metadata:
  name: provision
spec:
  template:
    spec:
      containers:
      - name: provision
        image: aws-cli:latest
        command:
        - /bin/sh
        - -c
        - |
          if ! koncli once check provision-s3 | grep -q "has been executed"; then
            aws s3 mb s3://my-app-bucket
          fi
```

## CLI Usage

```bash
# Check if executed
koncli once check app-init

# Create once
koncli once create app-init --ttl 1h

# List onces
koncli once list

# Delete once
koncli once delete app-init
```

## Use Cases

### Database Migrations
Ensure migrations run exactly once:

```yaml
apiVersion: konductor.io/v1
kind: Once
metadata:
  name: schema-migration
spec:
  ttl: 24h
```

### Application Initialization
One-time app setup across replicas:

```yaml
apiVersion: konductor.io/v1
kind: Once
metadata:
  name: app-init
spec:
  ttl: 1h
```

### Cloud Resource Provisioning
Create resources once:

```yaml
apiVersion: konductor.io/v1
kind: Once
metadata:
  name: provision-infra
spec:
  ttl: 24h
```

### Data Seeding
Seed database once:

```yaml
apiVersion: konductor.io/v1
kind: Once
metadata:
  name: seed-data
spec:
  ttl: 24h
```

## Best Practices

1. **Set appropriate TTL**: Clean up after completion
2. **Use descriptive names**: Clearly indicate the action
3. **Check before execute**: Verify execution status
4. **Idempotent actions**: Ensure actions can safely retry
5. **Handle failures**: Implement error handling

## Troubleshooting

### Once Not Executing
```bash
# Check status
kubectl get once my-once -o yaml

# Verify phase
kubectl get once my-once -o jsonpath='{.status.phase}'

# Check if already executed
kubectl get once my-once -o jsonpath='{.status.executed}'
```

### Multiple Executions
```bash
# Check executor
kubectl get once my-once -o jsonpath='{.status.executor}'

# Check execution time
kubectl get once my-once -o jsonpath='{.status.executedAt}'

# Verify only one execution
kubectl describe once my-once
```

## Advanced Patterns

### Conditional Initialization
```bash
#!/bin/bash
ONCE_NAME="app-init"

if koncli once check $ONCE_NAME | grep -q "not been executed"; then
  echo "First pod, running initialization..."
  initialize-app
else
  echo "Already initialized, skipping"
fi

start-app
```

### Reset Once
```bash
#!/bin/bash
# Delete and recreate to reset
koncli once delete app-init
koncli once create app-init --ttl 1h
```

### Multiple Stages
```yaml
apiVersion: konductor.io/v1
kind: Once
metadata:
  name: stage-1-init
---
apiVersion: konductor.io/v1
kind: Once
metadata:
  name: stage-2-init
---
apiVersion: konductor.io/v1
kind: Once
metadata:
  name: stage-3-init
```

## Comparison with Lease

| Feature | Once | Lease |
|---------|------|-------|
| Execution | One-time only | Renewable |
| Use Case | Initialization | Singleton process |
| Expiration | After completion | During execution |
| Renewal | No | Yes |

## Related Resources

- [Mutex API](./mutex.md) - Mutual exclusion
- [Lease API](./lease.md) - Singleton execution
- [CLI Reference](../cli/once.md) - Detailed CLI usage
