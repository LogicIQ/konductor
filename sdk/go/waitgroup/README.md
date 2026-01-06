# WaitGroup SDK

Go SDK for coordinating dynamic number of workers.

## Overview

WaitGroup allows coordinating a dynamic number of goroutines/pods, similar to Go's sync.WaitGroup.

## Installation

```go
import "github.com/LogicIQ/konductor/sdk/go/waitgroup"
```

## Functions

### Add

Increment the counter by delta.

```go
func Add(c *konductor.Client, ctx context.Context, name string, delta int32) error
```

**Example:**
```go
// Add 3 to counter
err := waitgroup.Add(client, ctx, "workers", 3)
```

### Done

Decrement the counter by 1.

```go
func Done(c *konductor.Client, ctx context.Context, name string) error
```

**Example:**
```go
// Signal completion
err := waitgroup.Done(client, ctx, "workers")
```

### Wait

Block until counter reaches zero.

```go
func Wait(c *konductor.Client, ctx context.Context, name string, opts ...konductor.Option) error
```

**Example:**
```go
// Wait for all workers
err := waitgroup.Wait(client, ctx, "workers",
    konductor.WithTimeout(5*time.Minute))
```

### GetCounter

Get current counter value.

```go
func GetCounter(c *konductor.Client, ctx context.Context, name string) (int32, error)
```

## Usage Examples

### Parallel Workers

```go
func processInParallel(client *konductor.Client, items []string) error {
    ctx := context.Background()
    wgName := "batch-workers"
    
    // Add count for all workers
    err := waitgroup.Add(client, ctx, wgName, int32(len(items)))
    if err != nil {
        return err
    }
    
    // Start workers
    for _, item := range items {
        go func(item string) {
            defer waitgroup.Done(client, ctx, wgName)
            processItem(item)
        }(item)
    }
    
    // Wait for completion
    return waitgroup.Wait(client, ctx, wgName,
        konductor.WithTimeout(10*time.Minute))
}
```

### Dynamic Job Coordination

```go
func coordinateJobs(client *konductor.Client) error {
    ctx := context.Background()
    wgName := "job-group"
    
    // Create waitgroup
    err := waitgroup.Create(client, ctx, wgName)
    if err != nil {
        return err
    }
    
    // Add initial count
    waitgroup.Add(client, ctx, wgName, 5)
    
    // Jobs call Done() when complete
    // ...
    
    // Wait for all jobs
    return waitgroup.Wait(client, ctx, wgName)
}
```

### Batch Processing

```go
func processBatch(client *konductor.Client, batchSize int) error {
    ctx := context.Background()
    wgName := "batch-process"
    
    waitgroup.Add(client, ctx, wgName, int32(batchSize))
    
    for i := 0; i < batchSize; i++ {
        go func(id int) {
            defer waitgroup.Done(client, ctx, wgName)
            log.Printf("Processing item %d", id)
            time.Sleep(2 * time.Second)
        }(i)
    }
    
    return waitgroup.Wait(client, ctx, wgName)
}
```

## Best Practices

1. **Add before starting workers**: Call Add() before launching goroutines
2. **Always call Done()**: Use defer to ensure Done() is called
3. **Set timeouts**: Use WithTimeout() to avoid indefinite waits
4. **Check counter**: Use GetCounter() for debugging
5. **Use TTL**: Set TTL for automatic cleanup

## Related

- [Barrier SDK](../barrier/README.md) - Fixed-count synchronization
- [Semaphore SDK](../semaphore/README.md) - Concurrent access control
