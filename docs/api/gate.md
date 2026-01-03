# Gate API

The Gate resource provides dependency coordination by waiting for multiple conditions before proceeding.

## Resource Definition

```yaml
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: processing-gate
  namespace: default
spec:
  conditions:
  - type: Job
    name: data-validation
    state: Complete
  - type: Job
    name: data-cleanup
    state: Complete
  - type: Barrier
    name: extractors-done
    state: Open
status:
  phase: Closed
  conditions:
  - type: Ready
    status: "True"
    reason: GateReady
  conditionsMet: 2
  conditionsTotal: 3
```

## Spec Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `conditions` | []Condition | Yes | List of conditions that must be met |
| `conditions[].type` | string | Yes | Resource type: `Job`, `Barrier`, `Lease`, `Gate` |
| `conditions[].name` | string | Yes | Resource name to check |
| `conditions[].state` | string | Yes | Expected state: `Complete`, `Open`, `Available` |
| `conditions[].namespace` | string | No | Resource namespace (defaults to gate namespace) |

## Status Fields

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current phase: `Open`, `Closed` |
| `conditionsMet` | integer | Number of conditions currently met |
| `conditionsTotal` | integer | Total number of conditions |

## Phases

- **Open**: All conditions met, gate is open
- **Closed**: One or more conditions not met, gate is closed

## Examples

### Job Dependencies

```yaml
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: processing-gate
spec:
  conditions:
  - type: Job
    name: data-validation
    state: Complete
  - type: Job
    name: data-cleanup
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
      - name: wait-gate
        image: logiciq/koncli:latest
        command: ["koncli", "gate", "wait", "processing-gate", "--timeout", "30m"]
      containers:
      - name: processor
        image: my-processor:latest
```

### Multi-Stage Pipeline

```yaml
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: extract-complete
spec:
  expected: 10
---
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: transform-gate
spec:
  conditions:
  - type: Barrier
    name: extract-complete
    state: Open
  - type: Job
    name: validation-job
    state: Complete
---
apiVersion: batch/v1
kind: Job
metadata:
  name: transform-job
spec:
  template:
    spec:
      initContainers:
      - name: wait-gate
        image: logiciq/koncli:latest
        command: ["koncli", "gate", "wait", "transform-gate"]
      containers:
      - name: transform
        image: my-transform:latest
```

### Complex Workflow

```yaml
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: deployment-gate
spec:
  conditions:
  - type: Job
    name: unit-tests
    state: Complete
  - type: Job
    name: integration-tests
    state: Complete
  - type: Job
    name: security-scan
    state: Complete
  - type: Gate
    name: staging-gate
    state: Open
---
apiVersion: batch/v1
kind: Job
metadata:
  name: deploy-production
spec:
  template:
    spec:
      initContainers:
      - name: wait-gate
        image: logiciq/koncli:latest
        command: ["koncli", "gate", "wait", "deployment-gate", "--timeout", "1h"]
      containers:
      - name: deploy
        image: my-deployer:latest
```

## CLI Usage

```bash
# Wait for gate to open
koncli gate wait processing-gate --timeout 30m

# Check gate status
koncli gate status processing-gate

# Create gate
koncli gate create processing-gate

# Delete gate
koncli gate delete processing-gate
```

## Use Cases

### Job Dependencies
Wait for prerequisite jobs before starting:

```yaml
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: job-deps
spec:
  conditions:
  - type: Job
    name: prerequisite-1
    state: Complete
  - type: Job
    name: prerequisite-2
    state: Complete
```

### Multi-Stage Pipelines
Coordinate complex ETL workflows:

```yaml
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: transform-gate
spec:
  conditions:
  - type: Barrier
    name: extract-complete
    state: Open
```

### Deployment Gates
Ensure all checks pass before deployment:

```yaml
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: deploy-gate
spec:
  conditions:
  - type: Job
    name: tests
    state: Complete
  - type: Job
    name: security-scan
    state: Complete
```

### Cross-Namespace Dependencies
Wait for resources in other namespaces:

```yaml
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: cross-ns-gate
spec:
  conditions:
  - type: Job
    name: shared-job
    namespace: shared-services
    state: Complete
```

## Best Practices

1. **Set appropriate timeouts**: Use `--timeout` to avoid indefinite waits
2. **Order conditions logically**: List dependencies in execution order
3. **Use descriptive names**: Make gate purpose clear
4. **Monitor gate status**: Check conditions regularly
5. **Handle failures**: Implement retry logic for transient failures

## Troubleshooting

### Gate Not Opening
```bash
# Check which conditions are not met
kubectl get gate my-gate -o yaml

# Check individual conditions
kubectl get job data-validation -o jsonpath='{.status.conditions}'
kubectl get barrier extractors-done -o jsonpath='{.status.phase}'
```

### Timeout Waiting
```bash
# Check gate status
koncli gate status my-gate

# Check conditions met
kubectl get gate my-gate -o jsonpath='{.status.conditionsMet}/{.status.conditionsTotal}'

# Describe gate for details
kubectl describe gate my-gate
```

### Condition Never Met
```bash
# Verify resource exists
kubectl get job prerequisite-job

# Check resource state
kubectl get job prerequisite-job -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}'

# Check for errors
kubectl describe job prerequisite-job
```

## Advanced Patterns

### Nested Gates
Gates can depend on other gates:

```yaml
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: stage-1-gate
spec:
  conditions:
  - type: Job
    name: job-1
    state: Complete
---
apiVersion: konductor.io/v1
kind: Gate
metadata:
  name: stage-2-gate
spec:
  conditions:
  - type: Gate
    name: stage-1-gate
    state: Open
  - type: Job
    name: job-2
    state: Complete
```

### Polling Pattern
Wait for gate with custom polling:

```bash
#!/bin/bash
GATE="my-gate"
MAX_WAIT=1800  # 30 minutes

start_time=$(date +%s)
while true; do
  if koncli gate status $GATE | grep -q "Open"; then
    echo "Gate is open"
    break
  fi
  
  elapsed=$(($(date +%s) - start_time))
  if [ $elapsed -gt $MAX_WAIT ]; then
    echo "Timeout waiting for gate"
    exit 1
  fi
  
  echo "Waiting for gate... ($elapsed/$MAX_WAIT seconds)"
  sleep 10
done
```

### Conditional Execution
Execute different paths based on gate:

```bash
#!/bin/bash
if koncli gate wait my-gate --timeout 5m; then
  echo "Gate opened, proceeding with main workflow"
  run-main-workflow
else
  echo "Gate timeout, running fallback"
  run-fallback-workflow
fi
```

## Related Resources

- [Barrier API](./barrier.md) - Multi-stage coordination
- [Lease API](./lease.md) - Singleton execution
- [CLI Reference](../cli/gate.md) - Detailed CLI usage
