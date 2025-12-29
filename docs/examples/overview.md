# Examples

Real-world examples demonstrating how to use Konductor synchronization primitives to solve common coordination problems in Kubernetes.

## Example Categories

### ETL and Data Processing
- **[ETL Pipeline](./etl-pipeline.md)** - Multi-stage extract, transform, load coordination
- **[Batch Processing](./batch-processing.md)** - Controlled concurrent batch job execution
- **[MapReduce Workflows](./mapreduce.md)** - Coordinate map and reduce phases

### Database Operations
- **[Database Migrations](./database-migrations.md)** - Singleton migration execution
- **[Backup Coordination](./backup-coordination.md)** - Prevent overlapping backups
- **[Connection Pooling](./connection-pooling.md)** - Limit database connections

### CI/CD and Deployments
- **[Deployment Gates](./deployment-gates.md)** - Wait for validation before deploy
- **[Test Coordination](./test-coordination.md)** - Coordinate distributed testing
- **[Release Orchestration](./release-orchestration.md)** - Multi-service deployment coordination

### Microservices Patterns
- **[Leader Election](./leader-election.md)** - Service leader election
- **[Circuit Breaker](./circuit-breaker.md)** - Coordinate service degradation
- **[Rate Limiting](./rate-limiting.md)** - Distributed rate limiting

## Quick Reference

### Common Patterns

#### InitContainer Gating
```yaml
initContainers:
- name: wait-dependencies
  image: logiciq/koncli:latest
  command:
  - koncli
  - barrier
  - wait
  - dependencies-ready
```

#### Semaphore in Script
```bash
if koncli semaphore acquire resource-limit --ttl=10m; then
  # Do work
  koncli semaphore release resource-limit
fi
```

#### Singleton CronJob
```bash
HOLDER_ID="$HOSTNAME-$$"
if koncli lease acquire singleton-job --holder $HOLDER_ID --timeout=0; then
  # Run job
  koncli lease release singleton-job --holder $HOLDER_ID
fi
```

#### Pipeline Stage Coordination
```bash
# Stage 1: Signal completion
koncli barrier arrive stage-1-complete

# Stage 2: Wait for stage 1
koncli barrier wait stage-1-complete --timeout=30m
```

## Integration Examples

### Argo Workflows
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: coordinated-workflow
spec:
  templates:
  - name: wait-barrier
    container:
      image: logiciq/koncli:latest
      command: [koncli, barrier, wait, "{{inputs.parameters.barrier-name}}"]
```

### Tekton Pipelines
```yaml
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: acquire-semaphore
spec:
  steps:
  - name: acquire
    image: logiciq/koncli:latest
    script: |
      koncli semaphore acquire "$(params.semaphore-name)" --ttl="$(params.ttl)"
```

### Flux/GitOps
```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: app-deployment
spec:
  dependsOn:
  - name: database-migration
  postBuild:
    substitute:
      BARRIER_NAME: "deployment-ready"
```

## Performance Examples

### High-Throughput Processing
```yaml
# Process 1000 items with max 50 concurrent workers
apiVersion: konductor.io/v1
kind: Semaphore
metadata:
  name: worker-pool
spec:
  permits: 50
  ttl: 5m
```

### Large-Scale Coordination
```yaml
# Coordinate 100 parallel jobs
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: massive-parallel
spec:
  expected: 100
  timeout: 2h
  quorum: 95  # Allow 5% failure rate
```

## Troubleshooting Examples

### Debug Stuck Barriers
```bash
# Check barrier status
kubectl describe barrier my-barrier

# List arrivals
kubectl get barrier my-barrier -o jsonpath='{.status.arrivals}'

# Check for missing jobs
kubectl get jobs -l barrier=my-barrier
```

### Monitor Semaphore Usage
```bash
# Watch semaphore status
watch kubectl get semaphore my-semaphore

# Check permit holders
kubectl get semaphore my-semaphore -o jsonpath='{.status.holders}'
```

### Lease Debugging
```bash
# Check lease holder
kubectl get lease my-lease -o jsonpath='{.status.holder}'

# Verify holder is alive
kubectl get pods | grep "$(kubectl get lease my-lease -o jsonpath='{.status.holder}')"
```

## Best Practices from Examples

1. **Use meaningful names** - Choose descriptive resource names
2. **Set appropriate timeouts** - Balance safety with responsiveness  
3. **Handle failures gracefully** - Always clean up resources
4. **Monitor coordination state** - Use status fields for observability
5. **Test coordination logic** - Verify behavior under failure conditions

## Contributing Examples

Have a useful Konductor pattern? We'd love to include it! 

1. Fork the repository
2. Add your example to the appropriate category
3. Include complete YAML manifests and explanations
4. Submit a pull request

## Next Steps

- Choose an example that matches your use case
- Adapt the patterns to your specific requirements
- Check the [API Reference](../api/overview.md) for detailed configuration options
- Use the [CLI Reference](../cli/overview.md) for command-line integration