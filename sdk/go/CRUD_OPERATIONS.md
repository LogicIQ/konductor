# Konductor SDK CRUD Operations

This document provides a complete reference of all CRUD (Create, Read, Update, Delete) operations available in the Konductor SDK for all coordination primitives.

## Summary

✅ **All objects now have complete CRUD operations:**
- **Create**: Create new instances
- **Read**: Get individual objects and List all objects  
- **Update**: Modify existing objects
- **Delete**: Remove objects

## Semaphore Operations

### Create
```go
// Create semaphore with permits
err := semaphore.Create(client, ctx, "api-limit", 10, konductor.WithTTL(5*time.Minute))

// Via main package
err := konductor.SemaphoreCreate(client, ctx, "api-limit", 10)
```

### Read
```go
// Get specific semaphore
sem, err := semaphore.Get(client, ctx, "api-limit")

// List all semaphores
semaphores, err := semaphore.List(client, ctx)

// Via main package
sem, err := konductor.SemaphoreGet(client, ctx, "api-limit")
semaphores, err := konductor.SemaphoreList(client, ctx)
```

### Update
```go
// Update semaphore
sem.Spec.Permits = 20
err := semaphore.Update(client, ctx, sem)

// Via main package
err := konductor.SemaphoreUpdate(client, ctx, sem)
```

### Delete
```go
// Delete semaphore
err := semaphore.Delete(client, ctx, "api-limit")

// Via main package
err := konductor.SemaphoreDelete(client, ctx, "api-limit")
```

### Additional Operations
```go
// Acquire permit
permit, err := semaphore.Acquire(client, ctx, "api-limit", opts...)

// Use with automatic cleanup
err := semaphore.With(client, ctx, "api-limit", func() error {
    // Protected code
    return nil
}, opts...)
```

## Barrier Operations

### Create
```go
// Create barrier expecting N arrivals
err := barrier.Create(client, ctx, "stage-gate", 5)

// Via main package
err := konductor.BarrierCreate(client, ctx, "stage-gate", 5)
```

### Read
```go
// Get specific barrier
bar, err := barrier.Get(client, ctx, "stage-gate")

// Get barrier status
status, err := barrier.GetStatus(client, ctx, "stage-gate")

// List all barriers
barriers, err := barrier.List(client, ctx)

// Via main package
bar, err := konductor.BarrierGet(client, ctx, "stage-gate")
barriers, err := konductor.BarrierList(client, ctx)
```

### Update
```go
// Update barrier
bar.Spec.Expected = 10
err := barrier.Update(client, ctx, bar)

// Via main package
err := konductor.BarrierUpdate(client, ctx, bar)
```

### Delete
```go
// Delete barrier
err := barrier.Delete(client, ctx, "stage-gate")

// Via main package
err := konductor.BarrierDelete(client, ctx, "stage-gate")
```

### Additional Operations
```go
// Wait for barrier to open
err := barrier.Wait(client, ctx, "stage-gate", opts...)

// Signal arrival
err := barrier.Arrive(client, ctx, "stage-gate", opts...)

// Execute function and signal arrival
err := barrier.With(client, ctx, "stage-gate", func() error {
    // Work to do
    return nil
}, opts...)
```

## Gate Operations

### Create
```go
// Create gate
err := gate.Create(client, ctx, "deployment-gate")

// Via main package
err := konductor.GateCreate(client, ctx, "deployment-gate")
```

### Read
```go
// Get specific gate
g, err := gate.Get(client, ctx, "deployment-gate")

// Get gate status
status, err := gate.GetStatus(client, ctx, "deployment-gate")

// Get conditions
conditions, err := gate.GetConditions(client, ctx, "deployment-gate")

// List all gates
gates, err := gate.List(client, ctx)

// Check if gate is open
isOpen, err := gate.Check(client, ctx, "deployment-gate")

// Via main package
g, err := konductor.GateGet(client, ctx, "deployment-gate")
gates, err := konductor.GateList(client, ctx)
isOpen, err := konductor.GateCheck(client, ctx, "deployment-gate")
```

### Update
```go
// Update gate
g.Spec.Conditions = append(g.Spec.Conditions, newCondition)
err := gate.Update(client, ctx, g)

// Via main package
err := konductor.GateUpdate(client, ctx, g)
```

### Delete
```go
// Delete gate
err := gate.Delete(client, ctx, "deployment-gate")

// Via main package
err := konductor.GateDelete(client, ctx, "deployment-gate")
```

### Additional Operations
```go
// Wait for gate to open
err := gate.Wait(client, ctx, "deployment-gate", opts...)

// Wait for specific conditions
err := gate.WaitForConditions(client, ctx, "deployment-gate", []string{"job1", "job2"}, opts...)

// Execute function after gate opens
err := gate.With(client, ctx, "deployment-gate", func() error {
    // Protected code
    return nil
}, opts...)

// Manual control
err := gate.Open(client, ctx, "deployment-gate")
err := gate.Close(client, ctx, "deployment-gate")
```

## Lease Operations

### Create
```go
// Create lease
err := lease.Create(client, ctx, "singleton-job", konductor.WithTTL(10*time.Minute))

// Via main package
err := konductor.LeaseCreate(client, ctx, "singleton-job")
```

### Read
```go
// Get specific lease
l, err := lease.Get(client, ctx, "singleton-job")

// List all leases
leases, err := lease.List(client, ctx)

// Check if lease is available
available, err := lease.IsAvailable(client, ctx, "singleton-job")

// Via main package
l, err := konductor.LeaseGet(client, ctx, "singleton-job")
leases, err := konductor.LeaseList(client, ctx)
available, err := konductor.LeaseIsAvailable(client, ctx, "singleton-job")
```

### Update
```go
// Update lease
l.Spec.TTL.Duration = 20 * time.Minute
err := lease.Update(client, ctx, l)

// Via main package
err := konductor.LeaseUpdate(client, ctx, l)
```

### Delete
```go
// Delete lease
err := lease.Delete(client, ctx, "singleton-job")

// Via main package
err := konductor.LeaseDelete(client, ctx, "singleton-job")
```

### Additional Operations
```go
// Acquire lease (blocking)
leaseHandle, err := lease.Acquire(client, ctx, "singleton-job", opts...)

// Try acquire lease (non-blocking)
leaseHandle, err := lease.TryAcquire(client, ctx, "singleton-job", opts...)

// Use with automatic cleanup
err := lease.With(client, ctx, "singleton-job", func() error {
    // Protected code
    return nil
}, opts...)

// Release lease
err := leaseHandle.Release()
```

## CLI Commands

All CRUD operations are also available via the CLI:

### Semaphore CLI
```bash
koncli semaphore create <name> --permits <n>
koncli semaphore list
koncli semaphore delete <name>
koncli semaphore acquire <name>
koncli semaphore release <name>
```

### Barrier CLI
```bash
koncli barrier create <name> --expected <n>
koncli barrier list
koncli barrier delete <name>
koncli barrier wait <name>
koncli barrier arrive <name>
```

### Gate CLI
```bash
koncli gate create <name>
koncli gate list
koncli gate delete <name>
koncli gate wait <name>
koncli gate open <name>
koncli gate close <name>
```

### Lease CLI
```bash
koncli lease create <name>
koncli lease list
koncli lease delete <name>
koncli lease acquire <name>
koncli lease release <name>
```

## Unit Tests

All CRUD operations are covered by unit tests:

- `sdk/go/semaphore/semaphore_test.go` - Tests Create, Get, List, Update, Delete
- `sdk/go/barrier/barrier_test.go` - Tests Create, Get, List, Update, Delete, GetStatus
- `sdk/go/gate/gate_test.go` - Tests Create, Get, List, Update, Delete, Check, GetStatus, GetConditions
- `sdk/go/lease/lease_test.go` - Tests Create, Get, List, Update, Delete, IsAvailable

## Options

All operations support various options:

```go
// Common options
konductor.WithTTL(duration)        // Set TTL for resources
konductor.WithTimeout(duration)    // Set operation timeout
konductor.WithHolder(string)       // Set holder identifier
konductor.WithPriority(int32)      // Set priority (for leases)
konductor.WithQuorum(int32)        // Set quorum (for barriers)
```

## Error Handling

All operations return appropriate errors:

```go
// Check for specific error types
if err != nil {
    if strings.Contains(err.Error(), "timeout") {
        // Handle timeout
    } else if strings.Contains(err.Error(), "not found") {
        // Handle not found
    } else {
        // Handle other errors
    }
}
```

## Best Practices

1. **Always use defer for cleanup**:
   ```go
   permit, err := semaphore.Acquire(client, ctx, "api-limit")
   if err != nil {
       return err
   }
   defer permit.Release()
   ```

2. **Use With functions for automatic cleanup**:
   ```go
   err := semaphore.With(client, ctx, "api-limit", func() error {
       // Your code here
       return nil
   })
   ```

3. **Set appropriate timeouts**:
   ```go
   permit, err := semaphore.Acquire(client, ctx, "api-limit",
       konductor.WithTimeout(30*time.Second))
   ```

4. **Handle errors appropriately**:
   ```go
   if err != nil {
       log.Printf("Operation failed: %v", err)
       return err
   }
   ```

This completes the CRUD operations for all Konductor coordination primitives. All objects now support Create, Read (Get/List), Update, and Delete operations through both the SDK and CLI interfaces.