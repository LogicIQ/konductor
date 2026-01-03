# API Reference

Konductor provides Custom Resource Definitions (CRDs) that implement synchronization primitives for Kubernetes workloads.

## Core Resources

| Resource | Purpose | Status |
|----------|---------|--------|
| [Semaphore](./semaphore.md) | Limit concurrent access | ✅ Available |
| [Barrier](./barrier.md) | Coordinate multi-stage workflows | ✅ Available |
| [Lease](./lease.md) | Singleton execution | ✅ Available |
| [Gate](./gate.md) | Dependency coordination | ✅ Available |
| [Mutex](./mutex.md) | Mutual exclusion | ✅ Available |
| [RWMutex](./rwmutex.md) | Read-write locks | ✅ Available |
| [Once](./once.md) | One-time execution | ✅ Available |

## Common Fields

All Konductor resources share common patterns:

### Metadata
Standard Kubernetes metadata with optional labels for organization:

```yaml
metadata:
  name: my-resource
  namespace: default
  labels:
    app: my-app
    stage: production
```

### Status Conditions
All resources report status through standard Kubernetes conditions:

```yaml
status:
  conditions:
  - type: Ready
    status: "True"
    reason: ResourceReady
    message: "Resource is ready for use"
  - type: Available
    status: "True" 
    reason: PermitsAvailable
    message: "2 permits available"
```

### TTL and Cleanup
Resources support automatic cleanup through TTL fields:

```yaml
spec:
  ttl: 10m  # Automatic cleanup after 10 minutes
```

## Resource Lifecycle

### Creation
Resources are created through standard Kubernetes APIs:

```bash
kubectl apply -f semaphore.yaml
```

### Monitoring
Check resource status:

```bash
kubectl get semaphore my-semaphore -o yaml
kubectl describe barrier my-barrier
```

### Cleanup
Resources clean up automatically based on TTL or can be deleted manually:

```bash
kubectl delete semaphore my-semaphore
```

## API Versions

| Version | Status | Notes |
|---------|--------|-------|
| v1 | Stable | Current stable API |
| v1alpha1 | Deprecated | Legacy, will be removed |

## Next Steps

- [Semaphore API](./semaphore.md) - Concurrent access control
- [Barrier API](./barrier.md) - Multi-stage coordination  
- [Lease API](./lease.md) - Singleton execution
- [Gate API](./gate.md) - Dependency coordination
- [Mutex API](./mutex.md) - Mutual exclusion
- [RWMutex API](./rwmutex.md) - Read-write locks
- [Once API](./once.md) - One-time execution
- [CLI Reference](../cli/overview.md) - Command-line usage