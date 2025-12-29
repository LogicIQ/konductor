# Quick Start

Get up and running with Konductor in 5 minutes. This guide walks through basic usage of each synchronization primitive.

## Prerequisites

- Konductor installed in your cluster ([Installation Guide](./installation.md))
- `kubectl` configured
- `koncli` CLI tool installed

## 1. Semaphore - Limit Concurrent Jobs

Create a semaphore to limit concurrent batch jobs:

```bash
# Create semaphore with 3 permits
kubectl apply -f - <<EOF
apiVersion: konductor.io/v1
kind: Semaphore
metadata:
  name: batch-limit
spec:
  permits: 3
  ttl: 10m
EOF
```

Test with CLI:

```bash
# Acquire a permit
koncli semaphore acquire batch-limit --ttl=5m
echo "Permit acquired, running workload..."
sleep 10
koncli semaphore release batch-limit
echo "Permit released"
```

Use in a Job:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: batch-job-1
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
        - batch-limit
        - --wait
        - --ttl=10m
      containers:
      - name: worker
        image: busybox
        command:
        - /bin/sh
        - -c
        - |
          echo "Processing batch data..."
          sleep 30
          echo "Batch complete"
          koncli semaphore release batch-limit
      restartPolicy: Never
```

## 2. Barrier - Coordinate Multi-Stage Pipeline

Create a barrier for coordinating pipeline stages:

```bash
# Create barrier expecting 3 jobs
kubectl apply -f - <<EOF
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: extract-complete
spec:
  expected: 3
  timeout: 30m
EOF
```

Create extract jobs that signal completion:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: extract-job-1
spec:
  template:
    spec:
      containers:
      - name: extractor
        image: busybox
        command:
        - /bin/sh
        - -c
        - |
          echo "Extracting data from source 1..."
          sleep 20
          echo "Extract complete, signaling barrier"
          koncli barrier arrive extract-complete
      restartPolicy: Never
```

Create transform job that waits for all extracts:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: transform-job
spec:
  template:
    spec:
      initContainers:
      - name: wait-extract
        image: logiciq/koncli:latest
        command:
        - koncli
        - barrier
        - wait
        - extract-complete
        - --timeout=30m
      containers:
      - name: transformer
        image: busybox
        command:
        - /bin/sh
        - -c
        - |
          echo "All extracts complete, starting transform..."
          sleep 15
          echo "Transform complete"
      restartPolicy: Never
```

## 3. Lease - Singleton Execution

Create a lease for singleton job execution:

```bash
# Create lease for database migration
kubectl apply -f - <<EOF
apiVersion: konductor.io/v1
kind: Lease
metadata:
  name: db-migration
spec:
  ttl: 15m
EOF
```

Test singleton behavior:

```bash
# Terminal 1 - Acquire lease
koncli lease acquire db-migration --holder terminal-1 --ttl=5m
echo "Running migration..."
sleep 30
koncli lease release db-migration --holder terminal-1

# Terminal 2 - Try to acquire (will fail while terminal 1 holds it)
koncli lease acquire db-migration --holder terminal-2 --timeout=0
```

Use in CronJob to prevent overlaps:

```yaml
apiVersion: konductor.io/v1
kind: Lease
metadata:
  name: daily-backup
spec:
  ttl: 2h
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: backup-job
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: busybox
            command:
            - /bin/sh
            - -c
            - |
              HOLDER_ID="backup-$HOSTNAME-$$"
              if koncli lease acquire daily-backup --holder $HOLDER_ID --timeout=0; then
                echo "Running backup..."
                sleep 60  # Simulate backup
                echo "Backup complete"
                koncli lease release daily-backup --holder $HOLDER_ID
              else
                echo "Previous backup still running, skipping"
              fi
          restartPolicy: Never
```

## 4. Monitor Resources

Check the status of your resources:

```bash
# List all konductor resources
kubectl get semaphores,barriers,leases

# Get detailed status
kubectl describe semaphore batch-limit
kubectl describe barrier extract-complete
kubectl describe lease db-migration

# Use CLI for status
koncli semaphore status batch-limit
koncli barrier status extract-complete
koncli lease status db-migration
```

## 5. Real-World Example: ETL Pipeline

Here's a complete ETL pipeline example:

```yaml
# Stage 1: Create barriers for coordination
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: extract-complete
spec:
  expected: 3
  timeout: 1h
---
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: transform-complete
spec:
  expected: 2
  timeout: 1h
---
# Stage 2: Create semaphore to limit concurrent loads
apiVersion: konductor.io/v1
kind: Semaphore
metadata:
  name: load-limit
spec:
  permits: 2
  ttl: 30m
---
# Stage 3: Extract jobs (3 parallel extractors)
apiVersion: batch/v1
kind: Job
metadata:
  name: extract-customers
spec:
  template:
    spec:
      containers:
      - name: extractor
        image: my-extractor:latest
        command:
        - /bin/sh
        - -c
        - |
          echo "Extracting customer data..."
          extract-customers
          koncli barrier arrive extract-complete
      restartPolicy: Never
---
# Stage 4: Transform jobs (wait for extract, then transform)
apiVersion: batch/v1
kind: Job
metadata:
  name: transform-customers
spec:
  template:
    spec:
      initContainers:
      - name: wait-extract
        image: logiciq/koncli:latest
        command:
        - koncli
        - barrier
        - wait
        - extract-complete
        - --timeout=1h
      containers:
      - name: transformer
        image: my-transformer:latest
        command:
        - /bin/sh
        - -c
        - |
          echo "Transforming customer data..."
          transform-customers
          koncli barrier arrive transform-complete
      restartPolicy: Never
---
# Stage 5: Load jobs (wait for transform, acquire semaphore)
apiVersion: batch/v1
kind: Job
metadata:
  name: load-customers
spec:
  template:
    spec:
      initContainers:
      - name: wait-transform
        image: logiciq/koncli:latest
        command:
        - koncli
        - barrier
        - wait
        - transform-complete
        - --timeout=1h
      - name: acquire-load-permit
        image: logiciq/koncli:latest
        command:
        - koncli
        - semaphore
        - acquire
        - load-limit
        - --wait
        - --ttl=30m
      containers:
      - name: loader
        image: my-loader:latest
        command:
        - /bin/sh
        - -c
        - |
          echo "Loading customer data..."
          load-customers
          koncli semaphore release load-limit
      restartPolicy: Never
```

Apply the pipeline:

```bash
kubectl apply -f etl-pipeline.yaml
```

Monitor progress:

```bash
# Watch jobs
kubectl get jobs -w

# Check barriers
kubectl get barriers
koncli barrier status extract-complete
koncli barrier status transform-complete

# Check semaphore
koncli semaphore status load-limit
```

## Cleanup

Remove the test resources:

```bash
# Delete resources
kubectl delete semaphore batch-limit
kubectl delete barrier extract-complete
kubectl delete lease db-migration

# Delete jobs
kubectl delete jobs --all
kubectl delete cronjobs --all
```

## Next Steps

Now that you've seen the basics, explore more advanced usage:

- **[Core Concepts](../introduction/concepts.md)** - Deeper understanding of primitives
- **[API Reference](../api/overview.md)** - Complete API documentation
- **[CLI Reference](../cli/overview.md)** - Full CLI command reference
- **[Examples](../examples/overview.md)** - Real-world usage patterns
- **[Best Practices](./best-practices.md)** - Production deployment guidance