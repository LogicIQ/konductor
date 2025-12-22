# Konductor CLI Examples

Real-world usage scenarios for the Konductor CLI (koncli).

## Table of Contents

- [Rate Limiting](#rate-limiting)
- [Batch Processing](#batch-processing)
- [Leader Election](#leader-election)
- [Multi-Stage Pipelines](#multi-stage-pipelines)
- [Service Coordination](#service-coordination)
- [CI/CD Integration](#cicd-integration)

## Rate Limiting

### API Rate Limiting

Limit concurrent API calls across multiple workers:

```bash
#!/bin/bash
# api-worker.sh

# Create semaphore (run once)
koncli semaphore create api-limit --permits 10

# Each worker acquires permit before API call
if koncli semaphore acquire api-limit --holder "$HOSTNAME-$$"; then
  echo "Acquired permit, calling API..."
  curl https://api.example.com/data
  koncli semaphore release api-limit --holder "$HOSTNAME-$$"
else
  echo "Failed to acquire permit"
  exit 1
fi
```

### Database Connection Pool

Manage database connections across pods:

```bash
#!/bin/bash
# db-query.sh

koncli semaphore create db-pool --permits 20

# Acquire connection
koncli semaphore acquire db-pool --holder "$POD_NAME"

# Run query
psql -h db.example.com -c "SELECT * FROM users"

# Release connection
koncli semaphore release db-pool --holder "$POD_NAME"
```

## Batch Processing

### Parallel Job Coordination

Process 1000 items with 50 concurrent workers:

```bash
#!/bin/bash
# batch-processor.sh

# Create semaphore for concurrency control
koncli semaphore create batch-workers --permits 50

# Process items
for item in $(cat items.txt); do
  (
    koncli semaphore acquire batch-workers --holder "worker-$item"
    process_item "$item"
    koncli semaphore release batch-workers --holder "worker-$item"
  ) &
done

wait
echo "All items processed"
```

### ETL Pipeline Stages

Coordinate multi-stage ETL pipeline:

```bash
#!/bin/bash
# etl-stage-1.sh (Extract)

# Do extraction work
extract_data

# Signal completion
koncli barrier arrive etl-stage-1-done --holder "$POD_NAME"
```

```bash
#!/bin/bash
# etl-stage-2.sh (Transform)

# Wait for all extractors to finish
koncli barrier wait etl-stage-1-done --timeout 30m

# Do transformation work
transform_data

# Signal completion
koncli barrier arrive etl-stage-2-done --holder "$POD_NAME"
```

```bash
#!/bin/bash
# etl-stage-3.sh (Load)

# Wait for all transformers to finish
koncli barrier wait etl-stage-2-done --timeout 30m

# Load data
load_data
```

## Leader Election

### Singleton Cron Job

Ensure only one instance runs the job:

```bash
#!/bin/bash
# singleton-job.sh

LEASE_NAME="daily-report-job"
HOLDER="$HOSTNAME-$$"

# Try to acquire lease
if koncli lease acquire "$LEASE_NAME" --holder "$HOLDER" --ttl 1h; then
  echo "I am the leader, running job..."
  
  # Run the actual job
  generate_daily_report
  
  # Release lease
  koncli lease release "$LEASE_NAME" --holder "$HOLDER"
else
  echo "Another instance is running the job, exiting"
  exit 0
fi
```

### Active-Passive Service

Implement active-passive failover:

```bash
#!/bin/bash
# service-with-leader-election.sh

LEASE_NAME="service-leader"
HOLDER="$POD_NAME"

while true; do
  if koncli lease acquire "$LEASE_NAME" --holder "$HOLDER" --ttl 30s; then
    echo "I am the leader"
    
    # Run active service
    run_active_service &
    SERVICE_PID=$!
    
    # Keep renewing lease
    while koncli lease acquire "$LEASE_NAME" --holder "$HOLDER" --ttl 30s; do
      sleep 20
    done
    
    # Lost leadership
    echo "Lost leadership, stopping service"
    kill $SERVICE_PID
  else
    echo "Standby mode, waiting..."
    sleep 10
  fi
done
```

## Multi-Stage Pipelines

### Distributed Testing

Wait for all services before running tests:

```bash
#!/bin/bash
# service-startup.sh

# Each service signals readiness
koncli barrier arrive services-ready --holder "$SERVICE_NAME"
```

```bash
#!/bin/bash
# run-tests.sh

# Wait for all 5 services to be ready
koncli barrier wait services-ready --timeout 5m

# Run integration tests
pytest tests/integration/
```

### Deployment Coordination

Coordinate blue-green deployment:

```bash
#!/bin/bash
# deploy-blue.sh

# Deploy blue environment
kubectl apply -f blue-deployment.yaml

# Wait for blue to be healthy
wait_for_healthy "blue"

# Signal blue is ready
koncli barrier arrive blue-ready --holder "deployer"
```

```bash
#!/bin/bash
# switch-traffic.sh

# Wait for blue to be ready
koncli barrier wait blue-ready --timeout 10m

# Switch traffic to blue
kubectl patch service app -p '{"spec":{"selector":{"version":"blue"}}}'

# Signal traffic switched
koncli gate open traffic-switched
```

```bash
#!/bin/bash
# cleanup-green.sh

# Wait for traffic switch
koncli gate wait traffic-switched --timeout 15m

# Clean up green environment
kubectl delete -f green-deployment.yaml
```

## Service Coordination

### Dependency Management

Service B waits for Service A:

```bash
#!/bin/bash
# service-a-startup.sh

# Start service A
start_service_a

# Signal service A is ready
koncli gate open service-a-ready
```

```bash
#!/bin/bash
# service-b-startup.sh

# Wait for service A
koncli gate wait service-a-ready --timeout 5m

# Start service B
start_service_b
```

### Database Migration Lock

Prevent concurrent migrations:

```bash
#!/bin/bash
# run-migrations.sh

LEASE_NAME="db-migrations"
HOLDER="$POD_NAME"

# Try to acquire migration lock
if koncli lease acquire "$LEASE_NAME" --holder "$HOLDER" --ttl 10m; then
  echo "Acquired migration lock"
  
  # Run migrations
  alembic upgrade head
  
  # Release lock
  koncli lease release "$LEASE_NAME" --holder "$HOLDER"
  
  echo "Migrations complete"
else
  echo "Another pod is running migrations"
  
  # Wait for migrations to complete
  while ! koncli lease acquire "$LEASE_NAME" --holder "$HOLDER" --ttl 1s; do
    sleep 5
  done
  koncli lease release "$LEASE_NAME" --holder "$HOLDER"
  
  echo "Migrations complete (by another pod)"
fi
```

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Install koncli
        run: |
          curl -L https://github.com/LogicIQ/konductor/releases/latest/download/koncli-linux-amd64 -o koncli
          chmod +x koncli
          sudo mv koncli /usr/local/bin/
      
      - name: Acquire deployment lock
        run: |
          koncli lease acquire deploy-lock --holder "$GITHUB_RUN_ID" --ttl 30m
      
      - name: Deploy application
        run: |
          kubectl apply -f k8s/
      
      - name: Release deployment lock
        if: always()
        run: |
          koncli lease release deploy-lock --holder "$GITHUB_RUN_ID"
```

### GitLab CI

```yaml
# .gitlab-ci.yml
deploy:
  stage: deploy
  script:
    - koncli semaphore acquire deploy-quota --holder "$CI_JOB_ID"
    - kubectl apply -f k8s/
    - koncli semaphore release deploy-quota --holder "$CI_JOB_ID"
  after_script:
    - koncli semaphore release deploy-quota --holder "$CI_JOB_ID" || true
```

### Jenkins Pipeline

```groovy
// Jenkinsfile
pipeline {
    agent any
    
    stages {
        stage('Deploy') {
            steps {
                script {
                    def holder = "${env.BUILD_TAG}"
                    
                    try {
                        sh "koncli lease acquire deploy-lock --holder ${holder} --ttl 30m"
                        sh "kubectl apply -f k8s/"
                    } finally {
                        sh "koncli lease release deploy-lock --holder ${holder} || true"
                    }
                }
            }
        }
    }
}
```

## Advanced Patterns

### Retry with Backoff

```bash
#!/bin/bash
# acquire-with-retry.sh

MAX_RETRIES=5
RETRY_DELAY=5

for i in $(seq 1 $MAX_RETRIES); do
  if koncli semaphore acquire api-limit --holder "$HOSTNAME"; then
    echo "Acquired permit"
    
    # Do work
    call_api
    
    # Release
    koncli semaphore release api-limit --holder "$HOSTNAME"
    exit 0
  fi
  
  echo "Failed to acquire permit, retry $i/$MAX_RETRIES"
  sleep $((RETRY_DELAY * i))
done

echo "Failed to acquire permit after $MAX_RETRIES retries"
exit 1
```

### Health Check Integration

```bash
#!/bin/bash
# health-check.sh

# Check if service has required permits
AVAILABLE=$(koncli status semaphore api-limit -o json | jq -r '.available')

if [ "$AVAILABLE" -gt 0 ]; then
  echo "Healthy: $AVAILABLE permits available"
  exit 0
else
  echo "Unhealthy: No permits available"
  exit 1
fi
```

### Monitoring and Alerting

```bash
#!/bin/bash
# monitor-semaphores.sh

# Check semaphore utilization
while true; do
  koncli semaphore list -o json | jq -r '.[] | 
    select(.inUse / .permits > 0.9) | 
    "ALERT: \(.name) is \(.inUse)/\(.permits) (\(.inUse / .permits * 100)%)"'
  
  sleep 60
done
```

## Troubleshooting

### List All Resources

```bash
# Show all coordination primitives
koncli status all

# Show specific types
koncli semaphore list
koncli barrier list
koncli lease list
koncli gate list
```

### Debug Stuck Barriers

```bash
# Check barrier status
koncli status barrier my-barrier

# See who has arrived
koncli status barrier my-barrier -o json | jq -r '.arrivals[]'

# Force reset (careful!)
kubectl delete barrier my-barrier
```

### Release Stuck Leases

```bash
# Check lease holder
koncli status lease my-lease

# Force release (if holder is dead)
kubectl delete lease my-lease
```

## Best Practices

1. **Always use unique holder IDs**: Use `$HOSTNAME-$$` or `$POD_NAME`
2. **Set appropriate TTLs**: Longer than expected work duration
3. **Use timeouts**: Prevent indefinite waiting
4. **Clean up on exit**: Use `trap` for cleanup
5. **Handle errors**: Check exit codes and retry
6. **Monitor usage**: Track permit utilization
7. **Use JSON output for automation**: Parse with `jq`

## Example: Complete Workflow

```bash
#!/bin/bash
# complete-workflow.sh

set -e

HOLDER="$POD_NAME"

# Cleanup on exit
trap 'koncli semaphore release api-limit --holder "$HOLDER" || true' EXIT

# Wait for dependencies
echo "Waiting for dependencies..."
koncli gate wait dependencies-ready --timeout 5m

# Acquire rate limit permit
echo "Acquiring API permit..."
koncli semaphore acquire api-limit --holder "$HOLDER"

# Do work
echo "Processing data..."
process_data

# Signal completion
echo "Signaling completion..."
koncli barrier arrive processing-done --holder "$HOLDER"

# Wait for all workers
echo "Waiting for all workers..."
koncli barrier wait processing-done --timeout 30m

echo "Workflow complete!"
```
