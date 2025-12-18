# Konductor CLI Usage Examples

This document provides comprehensive examples of using the Konductor CLI (`koncli`) for managing coordination primitives.

## Installation

Build the CLI:
```bash
cd cli
go build -o koncli .
```

## Global Options

All commands support these global options:
- `--namespace, -n`: Kubernetes namespace (auto-detected if running in pod)
- `--kubeconfig`: Path to kubeconfig file
- `--log-level`: Log level (debug, info, warn, error)

## Semaphore Operations

### Create a Semaphore
```bash
# Create semaphore with 5 permits
koncli semaphore create api-limit --permits 5

# Create with TTL for auto-cleanup
koncli semaphore create temp-limit --permits 3 --ttl 10m
```

### List Semaphores
```bash
koncli semaphore list
```

### Acquire a Permit
```bash
# Acquire permit (non-blocking)
koncli semaphore acquire api-limit

# Acquire with custom holder
koncli semaphore acquire api-limit --holder worker-1

# Acquire with timeout
koncli semaphore acquire api-limit --timeout 30s --wait

# Acquire with TTL
koncli semaphore acquire api-limit --ttl 5m
```

### Release a Permit
```bash
# Release permit
koncli semaphore release api-limit --holder worker-1
```

### Delete a Semaphore
```bash
koncli semaphore delete api-limit
```

## Barrier Operations

### Create a Barrier
```bash
# Create barrier expecting 3 arrivals
koncli barrier create stage-gate --expected 3

# Create with timeout
koncli barrier create stage-gate --expected 5 --timeout 10m

# Create with quorum (minimum arrivals to open)
koncli barrier create stage-gate --expected 10 --quorum 7
```

### List Barriers
```bash
koncli barrier list
```

### Signal Arrival
```bash
# Signal arrival at barrier
koncli barrier arrive stage-gate

# Signal arrival with custom holder
koncli barrier arrive stage-gate --holder worker-1
```

### Wait for Barrier
```bash
# Wait for barrier to open
koncli barrier wait stage-gate

# Wait with timeout
koncli barrier wait stage-gate --timeout 5m
```

### Delete a Barrier
```bash
koncli barrier delete stage-gate
```

## Lease Operations

### Create a Lease
```bash
# Create lease with default TTL
koncli lease create singleton-job

# Create with custom TTL
koncli lease create singleton-job --ttl 15m
```

### List Leases
```bash
koncli lease list
```

### Acquire a Lease
```bash
# Acquire lease (non-blocking)
koncli lease acquire singleton-job

# Acquire with priority (higher wins)
koncli lease acquire singleton-job --priority 10

# Acquire with timeout
koncli lease acquire singleton-job --timeout 30s --wait

# Acquire with custom holder
koncli lease acquire singleton-job --holder leader-1
```

### Release a Lease
```bash
# Release lease
koncli lease release singleton-job --holder leader-1
```

### Delete a Lease
```bash
koncli lease delete singleton-job
```

## Gate Operations

### Create a Gate
```bash
# Create gate (conditions managed by controllers)
koncli gate create deployment-gate
```

### List Gates
```bash
koncli gate list
```

### Wait for Gate
```bash
# Wait for gate to open (all conditions met)
koncli gate wait deployment-gate

# Wait with timeout
koncli gate wait deployment-gate --timeout 10m
```

### Manual Gate Control
```bash
# Manually open gate
koncli gate open deployment-gate

# Manually close gate
koncli gate close deployment-gate
```

### Delete a Gate
```bash
koncli gate delete deployment-gate
```

## Status Operations

### Check Overall Status
```bash
# Show status of all coordination primitives
koncli status

# Show status in specific namespace
koncli status -n production
```

## Common Patterns

### Pipeline Stage Synchronization
```bash
# Stage 1: Create barrier for stage completion
koncli barrier create stage1-complete --expected 3

# Workers signal completion
koncli barrier arrive stage1-complete --holder worker-1
koncli barrier arrive stage1-complete --holder worker-2
koncli barrier arrive stage1-complete --holder worker-3

# Stage 2: Wait for stage 1 completion
koncli barrier wait stage1-complete
echo "Stage 1 complete, starting stage 2"
```

### Rate Limited API Access
```bash
# Create semaphore for API rate limiting
koncli semaphore create api-rate-limit --permits 10

# Acquire permit before API call
koncli semaphore acquire api-rate-limit --ttl 1m
# ... make API call ...
koncli semaphore release api-rate-limit
```

### Singleton Job Execution
```bash
# Create lease for singleton job
koncli lease create daily-backup --ttl 1h

# Try to acquire lease
if koncli lease acquire daily-backup --timeout 5s; then
    echo "Running backup job"
    # ... run backup ...
    koncli lease release daily-backup
else
    echo "Backup already running"
fi
```

### Deployment Gate Pattern
```bash
# Create gate for deployment approval
koncli gate create prod-deployment-gate

# Wait for all conditions (managed by controllers)
koncli gate wait prod-deployment-gate --timeout 30m
echo "All deployment conditions met, proceeding with deployment"
```

## Error Handling

The CLI returns appropriate exit codes:
- `0`: Success
- `1`: General error
- Timeout errors include "timeout" in the message

Example with error handling:
```bash
#!/bin/bash
if ! koncli semaphore acquire api-limit --timeout 30s; then
    echo "Failed to acquire semaphore permit"
    exit 1
fi

# Ensure cleanup on exit
trap 'koncli semaphore release api-limit' EXIT

# Your code here
echo "Permit acquired, running protected code"
```

## Integration with Scripts

### Bash Integration
```bash
#!/bin/bash
set -e

# Function to cleanup on exit
cleanup() {
    koncli semaphore release api-limit --holder $HOSTNAME 2>/dev/null || true
}
trap cleanup EXIT

# Acquire permit
koncli semaphore acquire api-limit --holder $HOSTNAME --timeout 60s

# Your protected code here
echo "Running rate-limited operation"
sleep 10
```

### Docker Integration
```dockerfile
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY koncli /usr/local/bin/
COPY script.sh /script.sh
CMD ["/script.sh"]
```

### Kubernetes Job Integration
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: coordinated-job
spec:
  template:
    spec:
      containers:
      - name: worker
        image: my-app:latest
        command:
        - /bin/sh
        - -c
        - |
          # Wait for barrier before starting
          koncli barrier wait job-start-gate --timeout 5m
          
          # Acquire semaphore for resource access
          koncli semaphore acquire db-connections --timeout 30s
          
          # Run job
          ./my-job
          
          # Release semaphore
          koncli semaphore release db-connections
          
          # Signal completion
          koncli barrier arrive job-complete-gate
      restartPolicy: Never
```