# Koncli Usage Examples

## Semaphore Operations

### Acquire a permit
```bash
# Basic acquire
koncli semaphore acquire api-quota

# Acquire with wait and timeout
koncli semaphore acquire api-quota --wait --timeout=30s --ttl=5m

# Acquire with custom holder
koncli semaphore acquire api-quota --holder=my-job-123
```

### Release a permit
```bash
# Release permit
koncli semaphore release api-quota

# Release with custom holder
koncli semaphore release api-quota --holder=my-job-123
```

### List semaphores
```bash
koncli semaphore list
```

## Barrier Operations

### Wait for barrier
```bash
# Wait for barrier to open
koncli barrier wait stage-2

# Wait with timeout
koncli barrier wait stage-2 --timeout=10m
```

### Signal arrival
```bash
# Signal arrival at barrier
koncli barrier arrive stage-2

# Signal with custom holder
koncli barrier arrive stage-2 --holder=worker-pod-123
```

### List barriers
```bash
koncli barrier list
```

## Lease Operations

### Acquire lease
```bash
# Basic acquire
koncli lease acquire db-migration

# Acquire with priority and wait
koncli lease acquire db-migration --priority=5 --wait --timeout=1m

# Acquire with custom holder
koncli lease acquire db-migration --holder=migration-job-456
```

### Release lease
```bash
# Release lease
koncli lease release db-migration

# Release with custom holder
koncli lease release db-migration --holder=migration-job-456
```

### List leases
```bash
koncli lease list
```

## Gate Operations

### Wait for gate
```bash
# Wait for gate to open
koncli gate wait processing-gate

# Wait with timeout
koncli gate wait processing-gate --timeout=30m
```

### List gates
```bash
koncli gate list
```

## Status Operations

### Check specific resource status
```bash
# Semaphore status
koncli status semaphore api-quota

# Barrier status
koncli status barrier stage-2

# Lease status
koncli status lease db-migration

# Gate status
koncli status gate processing-gate
```

### Check all resources
```bash
koncli status all
```

## Common Usage Patterns

### InitContainer Pattern
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: worker-job
spec:
  template:
    spec:
      initContainers:
      - name: acquire-permit
        image: konductor/koncli:latest
        command:
        - koncli
        - semaphore
        - acquire
        - api-quota
        - --wait
        - --ttl=10m
      containers:
      - name: worker
        image: my-worker:latest
        command: ["./process-data.sh"]
```

### Multi-stage Pipeline
```bash
#!/bin/bash
set -e

# Stage 1: Acquire resources
koncli semaphore acquire db-connections --wait --ttl=30m
koncli lease acquire migration-lock --wait --timeout=5m

# Stage 2: Do work
./run-migration.sh

# Stage 3: Signal completion
koncli barrier arrive migration-complete

# Stage 4: Wait for downstream
koncli barrier wait validation-ready --timeout=10m

# Cleanup happens automatically via TTL and owner references
```

### Namespace Operations
```bash
# Use specific namespace
koncli -n production semaphore list

# Use kubeconfig context namespace
koncli status all
```