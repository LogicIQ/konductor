# SDK Integration Guide

Complete guide for integrating Konductor synchronization primitives into your Go applications.

## Installation

```bash
go get github.com/LogicIQ/konductor/sdk/go
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "time"
    
    konductor "github.com/LogicIQ/konductor/sdk/go"
)

func main() {
    // Create client
    client, err := konductor.New(&konductor.Config{
        Namespace: "default",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    
    // Use semaphore for rate limiting
    permit, err := client.AcquireSemaphore(ctx, "api-quota", 
        konductor.WithTTL(5*time.Minute))
    if err != nil {
        log.Fatal(err)
    }
    defer permit.Release()
    
    // Your rate-limited work here
    callExternalAPI()
}
```

## Client Configuration

### Basic Configuration

```go
// Use default configuration (in-cluster or kubeconfig)
client, err := konductor.New(nil)

// Specify namespace
client, err := konductor.New(&konductor.Config{
    Namespace: "production",
})

// Specify kubeconfig path
client, err := konductor.New(&konductor.Config{
    Kubeconfig: "/path/to/kubeconfig",
    Namespace:  "staging",
})
```

## Semaphores

Control concurrent access to shared resources.

### Basic Usage

```go
// Acquire permit
permit, err := client.AcquireSemaphore(ctx, "api-quota")
if err != nil {
    return err
}
defer permit.Release()

// Do rate-limited work
callAPI()
```

### With Options

```go
permit, err := client.AcquireSemaphore(ctx, "db-connections",
    konductor.WithTTL(10*time.Minute),      // Auto-expire after 10m
    konductor.WithTimeout(30*time.Second),   // Wait up to 30s
    konductor.WithHolder("my-app-instance"), // Custom holder ID
)
```

### Helper Function

```go
// Automatically acquire and release
err := client.WithSemaphore(ctx, "api-quota", func() error {
    return callExternalAPI()
}, konductor.WithTTL(5*time.Minute))
```

## Barriers

Synchronize multiple processes at coordination points.

### Basic Usage

```go
// Signal arrival
err := client.ArriveBarrier(ctx, "stage-1-complete",
    konductor.WithHolder("worker-1"))

// Wait for barrier to open
err := client.WaitBarrier(ctx, "stage-1-complete",
    konductor.WithTimeout(10*time.Minute))
```

### Multi-Stage Pipeline

```go
func runPipeline(ctx context.Context, client *konductor.Client, workerID string) error {
    // Stage 1
    if err := processStage1(); err != nil {
        return err
    }
    
    // Signal stage 1 complete
    if err := client.ArriveBarrier(ctx, "stage-1-done", 
        konductor.WithHolder(workerID)); err != nil {
        return err
    }
    
    // Wait for all workers to complete stage 1
    if err := client.WaitBarrier(ctx, "stage-1-done",
        konductor.WithTimeout(30*time.Minute)); err != nil {
        return err
    }
    
    // Stage 2
    return processStage2()
}
```

## Leases

Singleton execution and leader election.

### Basic Usage

```go
// Acquire lease
lease, err := client.AcquireLease(ctx, "migration-lock",
    konductor.WithTTL(10*time.Minute))
if err != nil {
    return err
}
defer lease.Release()

// Run singleton task
runMigration()
```

### Leader Election

```go
func runWithLeaderElection(ctx context.Context, client *konductor.Client) error {
    for {
        lease, err := client.AcquireLease(ctx, "service-leader",
            konductor.WithTTL(30*time.Second),
            konductor.WithTimeout(0)) // Don't wait
        
        if err != nil {
            // Not the leader, wait and retry
            time.Sleep(10 * time.Second)
            continue
        }
        
        // We are the leader
        log.Println("Became leader")
        
        // Start leader work with renewal
        return runAsLeader(ctx, lease)
    }
}
```

## Error Handling

### Common Errors

```go
permit, err := client.AcquireSemaphore(ctx, "api-quota")
if err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        log.Println("Timeout acquiring permit")
        
    case errors.Is(err, konductor.ErrNotFound):
        log.Println("Semaphore not found")
        
    case errors.Is(err, konductor.ErrAlreadyHeld):
        log.Println("Already holding permit")
        
    default:
        log.Printf("Error: %v", err)
    }
    return err
}
```

## Best Practices

1. **Always Use Defer for Cleanup**
```go
permit, err := client.AcquireSemaphore(ctx, "resource")
if err != nil {
    return err
}
defer permit.Release() // Always release
```

2. **Set Appropriate TTLs**
```go
// TTL should be longer than expected work duration
expectedDuration := 5 * time.Minute
permit, err := client.AcquireSemaphore(ctx, "api-quota",
    konductor.WithTTL(2 * expectedDuration))
```

3. **Use Timeouts**
```go
// Always set timeouts for waiting operations
err := client.WaitBarrier(ctx, "stage-1",
    konductor.WithTimeout(10*time.Minute))
```

## Integration Patterns

### Service Startup Coordination

```go
type Service struct {
    client *konductor.Client
    lease  *konductor.Lease
}

func (s *Service) Start(ctx context.Context) error {
    // Wait for dependencies
    if err := s.client.WaitGate(ctx, "dependencies-ready"); err != nil {
        return err
    }
    
    // Acquire service lease for leader election
    lease, err := s.client.AcquireLease(ctx, "service-leader",
        konductor.WithTTL(30*time.Second))
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

## API Reference

### Client Methods

#### Semaphore Operations
- `AcquireSemaphore(ctx, name, ...opts) (*Permit, error)` - Acquire a semaphore permit
- `WithSemaphore(ctx, name, fn, ...opts) error` - Execute function with permit

#### Barrier Operations  
- `WaitBarrier(ctx, name, ...opts) error` - Wait for barrier to open
- `ArriveBarrier(ctx, name, ...opts) error` - Signal arrival at barrier
- `WithBarrier(ctx, name, fn, ...opts) error` - Arrive and wait

#### Lease Operations
- `AcquireLease(ctx, name, ...opts) (*Lease, error)` - Acquire a lease
- `TryAcquireLease(ctx, name, ...opts) (*Lease, error)` - Try to acquire (non-blocking)
- `WithLease(ctx, name, fn, ...opts) error` - Execute function with lease

### Options

- `WithTTL(duration)` - Set TTL for permits/leases
- `WithTimeout(duration)` - Set wait timeout
- `WithPriority(int)` - Set priority for leases
- `WithHolder(string)` - Set holder identifier

## Related Documentation

- [API Reference](../api/overview.md) - Complete API documentation
- [CLI Reference](../cli/overview.md) - Command-line usage
- [Examples](../examples/overview.md) - Real-world usage patterns