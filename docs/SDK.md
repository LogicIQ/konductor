# Konductor SDK Documentation

Complete guide for integrating Konductor synchronization primitives into your Go applications.

## Table of Contents

- [Roadmap](#roadmap)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Client Configuration](#client-configuration)
- [Semaphores](#semaphores)
- [Barriers](#barriers)
- [Leases](#leases)
- [Gates](#gates)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)
- [Integration Patterns](#integration-patterns)

## Roadmap

Konductor SDK development follows a phased approach aligned with the core product roadmap:

### Phase 1: MVP (Current) ✅
- **Semaphores**: Rate limiting and resource throttling
- **Barriers**: Multi-stage workflow coordination
- **Basic Client**: Core configuration and connection handling
- **Error Handling**: Comprehensive error types and retry patterns

### Phase 2: Enhanced Features (In Progress) 🚧
- **Leases**: Singleton execution and leader election
- **Priority Support**: Preemption and priority-based acquisition
- **Advanced TTL**: Automatic renewal and background management
- **Metrics Integration**: Built-in observability hooks

### Phase 3: Advanced Coordination (Planned) 📋
- **Gates**: Complex dependency coordination
- **Cross-Namespace**: Multi-tenant coordination primitives
- **Performance Optimizations**: Connection pooling and caching
- **SDK Extensions**: Framework-specific integrations

### Language Support Roadmap
- **Go SDK**: ✅ Available (current)
- **Python SDK**: 🚧 In development
- **Java SDK**: 📋 Planned
- **Node.js SDK**: 📋 Planned
- **Rust SDK**: 📋 Community-driven

### Integration Targets
- **Argo Workflows**: Native step coordination
- **Tekton Pipelines**: Task synchronization
- **Kubeflow**: ML pipeline coordination
- **Spark on K8s**: Job-level resource management

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

### In-Cluster vs Out-of-Cluster

The SDK automatically detects whether it's running inside a Kubernetes cluster:

```go
// In-cluster (uses service account)
client, err := konductor.New(&konductor.Config{
    Namespace: "default",
})

// Out-of-cluster (uses kubeconfig)
client, err := konductor.New(&konductor.Config{
    Kubeconfig: os.Getenv("KUBECONFIG"),
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

### Advanced: Manual Renewal

```go
permit, err := client.AcquireSemaphore(ctx, "long-task",
    konductor.WithTTL(1*time.Minute))
if err != nil {
    return err
}
defer permit.Release()

// Start renewal goroutine
go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if err := permit.Renew(ctx); err != nil {
                log.Printf("Failed to renew: %v", err)
                return
            }
        case <-ctx.Done():
            return
        }
    }
}()

// Do long-running work
processLargeDataset()
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
    if err := processStage2(); err != nil {
        return err
    }
    
    return nil
}
```

### Helper Function

```go
// Arrive and wait in one call
err := client.WithBarrier(ctx, "sync-point", func() error {
    return doWork()
}, konductor.WithHolder("worker-1"))
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
        
        // Start leader work
        leaderCtx, cancel := context.WithCancel(ctx)
        go runLeaderTasks(leaderCtx)
        
        // Keep renewing lease
        ticker := time.NewTicker(20 * time.Second)
        for {
            select {
            case <-ticker.C:
                if err := lease.Renew(ctx); err != nil {
                    log.Println("Lost leadership")
                    cancel()
                    break
                }
            case <-ctx.Done():
                cancel()
                lease.Release()
                return ctx.Err()
            }
        }
    }
}
```

### Try Acquire (Non-blocking)

```go
// Try to acquire without waiting
lease, err := client.TryAcquireLease(ctx, "singleton-task")
if err != nil {
    if errors.Is(err, konductor.ErrLeaseHeld) {
        log.Println("Another instance is running")
        return nil
    }
    return err
}
defer lease.Release()

// Run singleton task
runTask()
```

## Gates

Wait for multiple conditions before proceeding.

### Basic Usage

```go
// Wait for gate to open
err := client.WaitGate(ctx, "dependencies-ready",
    konductor.WithTimeout(5*time.Minute))
if err != nil {
    return err
}

// All dependencies are ready
startService()
```

### Check Gate Status

```go
// Check if gate is open without waiting
isOpen, err := client.CheckGate(ctx, "processing-gate")
if err != nil {
    return err
}

if isOpen {
    log.Println("Gate is open")
} else {
    log.Println("Gate is closed")
}
```

### Get Gate Conditions

```go
// Get detailed condition status
conditions, err := client.GetGateConditions(ctx, "workflow-gate")
if err != nil {
    return err
}

for _, cond := range conditions {
    status := "NOT MET"
    if cond.Met {
        status = "MET"
    }
    log.Printf("%s: %s/%s - %s", status, cond.Type, cond.Name, cond.Message)
}
```

## Error Handling

### Common Errors

```go
permit, err := client.AcquireSemaphore(ctx, "api-quota")
if err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        // Timeout waiting for permit
        log.Println("Timeout acquiring permit")
        
    case errors.Is(err, context.Canceled):
        // Context was canceled
        log.Println("Operation canceled")
        
    case errors.Is(err, konductor.ErrNotFound):
        // Resource doesn't exist
        log.Println("Semaphore not found")
        
    case errors.Is(err, konductor.ErrAlreadyHeld):
        // Already holding the resource
        log.Println("Already holding permit")
        
    default:
        // Other errors (network, API, etc.)
        log.Printf("Error: %v", err)
    }
    return err
}
```

### Retry Logic

```go
func acquireWithRetry(ctx context.Context, client *konductor.Client, name string) (*konductor.Permit, error) {
    maxRetries := 3
    backoff := time.Second
    
    for i := 0; i < maxRetries; i++ {
        permit, err := client.AcquireSemaphore(ctx, name,
            konductor.WithTimeout(10*time.Second))
        
        if err == nil {
            return permit, nil
        }
        
        if !errors.Is(err, context.DeadlineExceeded) {
            return nil, err // Don't retry non-timeout errors
        }
        
        log.Printf("Retry %d/%d after %v", i+1, maxRetries, backoff)
        time.Sleep(backoff)
        backoff *= 2
    }
    
    return nil, fmt.Errorf("failed after %d retries", maxRetries)
}
```

## Best Practices

### 1. Always Use Defer for Cleanup

```go
permit, err := client.AcquireSemaphore(ctx, "resource")
if err != nil {
    return err
}
defer permit.Release() // Always release
```

### 2. Set Appropriate TTLs

```go
// TTL should be longer than expected work duration
expectedDuration := 5 * time.Minute
permit, err := client.AcquireSemaphore(ctx, "api-quota",
    konductor.WithTTL(2 * expectedDuration))
```

### 3. Use Timeouts

```go
// Always set timeouts for waiting operations
err := client.WaitBarrier(ctx, "stage-1",
    konductor.WithTimeout(10*time.Minute))
```

### 4. Handle Context Cancellation

```go
select {
case <-ctx.Done():
    return ctx.Err()
default:
    // Continue with coordination
}
```

### 5. Use Unique Holder IDs

```go
holder := fmt.Sprintf("%s-%s", os.Getenv("POD_NAME"), uuid.New())
permit, err := client.AcquireSemaphore(ctx, "resource",
    konductor.WithHolder(holder))
```

## Integration Patterns

### InitContainer Pattern

```go
// wait-for-dependencies.go
func main() {
    client, _ := konductor.New(nil)
    ctx := context.Background()
    
    // Wait for dependencies
    if err := client.WaitGate(ctx, "dependencies-ready",
        konductor.WithTimeout(5*time.Minute)); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Dependencies ready, starting main container")
}
```

```yaml
apiVersion: v1
kind: Pod
spec:
  initContainers:
  - name: wait-dependencies
    image: my-app:latest
    command: ["/app/wait-for-dependencies"]
  containers:
  - name: main
    image: my-app:latest
```

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

func (s *Service) Stop() error {
    if s.lease != nil {
        return s.lease.Release()
    }
    return nil
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

### Graceful Shutdown

```go
func main() {
    client, _ := konductor.New(nil)
    ctx, cancel := context.WithCancel(context.Background())
    
    // Handle shutdown signals
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-sigCh
        log.Println("Shutting down...")
        cancel()
    }()
    
    // Acquire lease
    lease, err := client.AcquireLease(ctx, "service-leader")
    if err != nil {
        log.Fatal(err)
    }
    defer lease.Release()
    
    // Run service
    runService(ctx)
}
```

## Examples

See the [examples directory](./examples/) for complete working examples:

- [basic_usage.go](./examples/basic_usage.go) - Basic usage of all primitives
- [sdk_usage.go](./examples/sdk_usage.go) - Advanced SDK patterns

## API Reference

### Client Methods

#### Semaphore Operations
- `AcquireSemaphore(ctx, name, ...opts) (*Permit, error)` - Acquire a semaphore permit
- `WithSemaphore(ctx, name, fn, ...opts) error` - Execute function with permit
- `ListSemaphores(ctx) ([]Semaphore, error)` - List all semaphores
- `GetSemaphore(ctx, name) (*Semaphore, error)` - Get semaphore details

#### Barrier Operations  
- `WaitBarrier(ctx, name, ...opts) error` - Wait for barrier to open
- `ArriveBarrier(ctx, name, ...opts) error` - Signal arrival at barrier
- `WithBarrier(ctx, name, fn, ...opts) error` - Arrive and wait
- `ListBarriers(ctx) ([]Barrier, error)` - List all barriers
- `GetBarrier(ctx, name) (*Barrier, error)` - Get barrier details

#### Lease Operations
- `AcquireLease(ctx, name, ...opts) (*Lease, error)` - Acquire a lease
- `TryAcquireLease(ctx, name, ...opts) (*Lease, error)` - Try to acquire (non-blocking)
- `WithLease(ctx, name, fn, ...opts) error` - Execute function with lease
- `ListLeases(ctx) ([]Lease, error)` - List all leases
- `GetLease(ctx, name) (*Lease, error)` - Get lease details

#### Gate Operations
- `WaitGate(ctx, name, ...opts) error` - Wait for gate to open
- `CheckGate(ctx, name) (bool, error)` - Check if gate is open
- `GetGateConditions(ctx, name) ([]GateConditionStatus, error)` - Get condition status
- `ListGates(ctx) ([]Gate, error)` - List all gates
- `GetGate(ctx, name) (*Gate, error)` - Get gate details

### Options

- `WithTTL(duration)` - Set TTL for permits/leases
- `WithTimeout(duration)` - Set wait timeout
- `WithPriority(int)` - Set priority for leases
- `WithHolder(string)` - Set holder identifier

## Contributing

Contributions welcome! Please see the main repository for guidelines.

## License

Apache 2.0
