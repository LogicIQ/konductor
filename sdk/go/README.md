# Konductor Go SDK

A Go SDK for interacting with Konductor coordination primitives in Kubernetes.

## Installation

```bash
go get github.com/LogicIQ/konductor/sdk/go
```

## Quick Start

```go
package main

import (
    "context"
    "time"
    
    konductor "github.com/LogicIQ/konductor/sdk/go"
)

func main() {
    // Create client
    client, err := konductor.New(&konductor.Config{
        Namespace: "default",
    })
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    
    // Use semaphore for rate limiting
    permit, err := client.AcquireSemaphore(ctx, "api-quota", 
        konductor.WithTTL(5*time.Minute))
    if err != nil {
        panic(err)
    }
    defer permit.Release()
    
    // Your rate-limited work here
    callExternalAPI()
}
```

## Features

### Semaphores
Control concurrent access to shared resources:

```go
// Acquire permit with automatic TTL renewal
permit, err := client.AcquireSemaphore(ctx, "db-connections",
    konductor.WithTTL(10*time.Minute),
    konductor.WithTimeout(30*time.Second))
if err != nil {
    return err
}
defer permit.Release()

// Or use helper for automatic management
err := client.WithSemaphore(ctx, "api-quota", func() error {
    return callExternalAPI()
}, konductor.WithTTL(5*time.Minute))
```

### Barriers
Coordinate multi-stage workflows:

```go
// Wait for barrier to open
err := client.WaitBarrier(ctx, "stage-1-complete",
    konductor.WithTimeout(10*time.Minute))
if err != nil {
    return err
}

// Do work
processData()

// Signal completion
err = client.ArriveBarrier(ctx, "stage-2-ready")
```

### Leases
Singleton execution and leader election:

```go
// Acquire lease with priority
lease, err := client.AcquireLease(ctx, "migration-lock",
    konductor.WithPriority(10),
    konductor.WithTimeout(1*time.Minute))
if err != nil {
    return err
}
defer lease.Release()

// Run singleton task
runMigration()
```

### Gates
Wait for multiple conditions:

```go
// Wait for all conditions to be met
err := client.WaitGate(ctx, "processing-gate",
    konductor.WithTimeout(30*time.Minute))
if err != nil {
    return err
}

// All dependencies are ready
startProcessing()
```

## Configuration Options

### Client Configuration
```go
client, err := konductor.New(&konductor.Config{
    Namespace:  "production",
    Kubeconfig: "/path/to/kubeconfig", // optional
})
```

### Operation Options
```go
// Common options for all operations
konductor.WithTTL(5*time.Minute)        // Set TTL for permits/leases
konductor.WithTimeout(30*time.Second)   // Set wait timeout
konductor.WithPriority(5)               // Set priority for leases
konductor.WithHolder("my-app-instance") // Set holder identifier
```

## Integration Patterns

### InitContainer Pattern
```yaml
apiVersion: batch/v1
kind: Job
spec:
  template:
    spec:
      initContainers:
      - name: wait-dependencies
        image: my-app:latest
        command:
        - /app/wait-for-dependencies
        # Uses SDK to wait for gates/barriers
      containers:
      - name: main
        image: my-app:latest
```

### Service Startup Coordination
```go
func (s *Service) Start(ctx context.Context) error {
    // Wait for dependencies
    if err := s.client.WaitGate(ctx, "dependencies-ready"); err != nil {
        return err
    }
    
    // Acquire service lease
    lease, err := s.client.AcquireLease(ctx, "service-leader")
    if err != nil {
        return err
    }
    s.lease = lease
    
    // Start service
    return s.startHTTPServer()
}
```

### Batch Processing
```go
func processBatch(ctx context.Context, items []Item) error {
    client, _ := konductor.New(nil)
    
    // Rate limit batch processing
    return client.WithSemaphore(ctx, "batch-processor", func() error {
        for _, item := range items {
            if err := processItem(item); err != nil {
                return err
            }
        }
        return nil
    }, konductor.WithTTL(30*time.Minute))
}
```

## Error Handling

The SDK returns standard Go errors. Common error scenarios:

```go
permit, err := client.AcquireSemaphore(ctx, "api-quota")
if err != nil {
    // Handle specific error types
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        // Timeout waiting for permit
    case errors.Is(err, context.Canceled):
        // Context was canceled
    default:
        // Other errors (network, API, etc.)
    }
}
```

## Best Practices

1. **Always use defer for cleanup**:
   ```go
   permit, err := client.AcquireSemaphore(ctx, "resource")
   if err != nil {
       return err
   }
   defer permit.Release() // Always release
   ```

2. **Set appropriate TTLs**:
   ```go
   // TTL should be longer than expected work duration
   permit, err := client.AcquireSemaphore(ctx, "api-quota",
       konductor.WithTTL(2*expectedWorkDuration))
   ```

3. **Use timeouts for waiting operations**:
   ```go
   err := client.WaitBarrier(ctx, "stage-1",
       konductor.WithTimeout(10*time.Minute))
   ```

4. **Handle context cancellation**:
   ```go
   select {
   case <-ctx.Done():
       return ctx.Err()
   default:
       // Continue with coordination
   }
   ```

## Examples

See the [examples](./examples/) directory for complete usage examples.

## API Reference

### Client Methods

#### Semaphore Operations
- `AcquireSemaphore(ctx, name, ...opts) (*Permit, error)`
- `WithSemaphore(ctx, name, fn, ...opts) error`
- `ListSemaphores(ctx) ([]Semaphore, error)`
- `GetSemaphore(ctx, name) (*Semaphore, error)`

#### Barrier Operations  
- `WaitBarrier(ctx, name, ...opts) error`
- `ArriveBarrier(ctx, name, ...opts) error`
- `WithBarrier(ctx, name, fn, ...opts) error`
- `ListBarriers(ctx) ([]Barrier, error)`
- `GetBarrier(ctx, name) (*Barrier, error)`

#### Lease Operations
- `AcquireLease(ctx, name, ...opts) (*Lease, error)`
- `WithLease(ctx, name, fn, ...opts) error`
- `TryAcquireLease(ctx, name, ...opts) (*Lease, error)`
- `ListLeases(ctx) ([]Lease, error)`
- `GetLease(ctx, name) (*Lease, error)`

#### Gate Operations
- `WaitGate(ctx, name, ...opts) error`
- `CheckGate(ctx, name) (bool, error)`
- `GetGateConditions(ctx, name) ([]GateConditionStatus, error)`
- `ListGates(ctx) ([]Gate, error)`
- `GetGate(ctx, name) (*Gate, error)`