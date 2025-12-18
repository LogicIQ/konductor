# Konductor SDK Design

## Overview
SDKs extend konductor beyond CLI usage for **job-level coordination**, not per-request coordination. They enable applications to coordinate at appropriate granularity - startup, batch processing, and long-running operations.

## Performance Reality Check
**Konductor is NOT for per-request coordination.** K8s API calls have ~10-50ms latency and are expensive. Appropriate use cases:

✅ **Good**: Job startup, batch processing, service initialization
❌ **Bad**: HTTP request middleware, per-function calls, tight loops

## Core SDK Principles

### 1. Language-Native Patterns
Each SDK should feel natural in its target language:
- **Go**: Context-aware, error handling, defer patterns
- **Python**: Async/await, context managers, decorators  
- **JavaScript**: Promises, async/await, middleware patterns
- **Java**: CompletableFuture, try-with-resources
- **Rust**: Result types, RAII, async traits

### 2. Automatic Resource Management
```go
// Go: defer pattern
permit, err := konductor.AcquireSemaphore(ctx, "api-quota")
if err != nil {
    return err
}
defer permit.Release() // Automatic cleanup
```

```python
# Python: context manager
async with konductor.semaphore("api-quota") as permit:
    await call_external_api()
    # Automatic release on exit
```

### 3. Framework Integration
SDKs should integrate with popular frameworks:
- **HTTP middleware** for automatic rate limiting
- **Database connection pools** with semaphore integration
- **Message queue consumers** with coordination
- **Batch processing frameworks** with barriers

## SDK Feature Matrix

| Feature | Go SDK | Python SDK | Node.js SDK | Java SDK |
|---------|--------|------------|-------------|----------|
| **Semaphore** | ✅ | ✅ | ✅ | ✅ |
| **Barrier** | ✅ | ✅ | ✅ | ✅ |
| **Lease** | ✅ | ✅ | ✅ | ✅ |
| **Gate** | ✅ | ✅ | ✅ | ✅ |
| **Async Support** | ✅ | ✅ | ✅ | ✅ |
| **Context/Cancellation** | ✅ | ✅ | ✅ | ✅ |
| **Automatic Renewal** | ✅ | ✅ | ✅ | ✅ |
| **Framework Integration** | ⚠️ | ⚠️ | ⚠️ | ⚠️ |

## Go SDK Design

### Core Interface
```go
package konductor

import (
    "context"
    "time"
)

// Client provides coordination primitives
type Client struct {
    kubeClient kubernetes.Interface
    namespace  string
}

// Semaphore operations
func (c *Client) AcquireSemaphore(ctx context.Context, name string, opts ...SemaphoreOption) (*Permit, error)
func (c *Client) ReleaseSemaphore(ctx context.Context, name string, permitID string) error

// Barrier operations  
func (c *Client) WaitBarrier(ctx context.Context, name string, opts ...BarrierOption) error
func (c *Client) ArriveBarrier(ctx context.Context, name string) error

// Lease operations
func (c *Client) AcquireLease(ctx context.Context, name string, opts ...LeaseOption) (*Lease, error)

// Options pattern
type SemaphoreOption func(*SemaphoreConfig)

func WithTTL(ttl time.Duration) SemaphoreOption
func WithTimeout(timeout time.Duration) SemaphoreOption
func WithPriority(priority int) SemaphoreOption
```

### Usage Examples
```go
// Basic semaphore
func processWithRateLimit(ctx context.Context) error {
    client := konductor.NewClient()
    
    permit, err := client.AcquireSemaphore(ctx, "api-quota",
        konductor.WithTTL(5*time.Minute),
        konductor.WithTimeout(30*time.Second))
    if err != nil {
        return fmt.Errorf("failed to acquire permit: %w", err)
    }
    defer permit.Release()
    
    return callExternalAPI()
}

// Barrier coordination
func waitForUpstream(ctx context.Context) error {
    client := konductor.NewClient()
    
    // Wait for all upstream services
    if err := client.WaitBarrier(ctx, "upstream-ready",
        konductor.WithTimeout(10*time.Minute)); err != nil {
        return err
    }
    
    // Signal this service is ready
    return client.ArriveBarrier(ctx, "downstream-ready")
}

// Leader election
func runAsLeader(ctx context.Context) error {
    client := konductor.NewClient()
    
    lease, err := client.AcquireLease(ctx, "service-leader",
        konductor.WithTTL(30*time.Second))
    if err != nil {
        return err
    }
    defer lease.Release()
    
    // Run leader-only logic
    return runLeaderTasks(ctx)
}
```

## Python SDK Design

### Core Interface
```python
import asyncio
from contextlib import asynccontextmanager
from typing import Optional, AsyncGenerator

class KonductorClient:
    def __init__(self, namespace: str = "default"):
        self.namespace = namespace
    
    @asynccontextmanager
    async def semaphore(self, name: str, ttl: Optional[int] = None, 
                       timeout: Optional[int] = None) -> AsyncGenerator[Permit, None]:
        """Context manager for semaphore acquisition"""
        
    async def wait_barrier(self, name: str, timeout: Optional[int] = None) -> None:
        """Wait for barrier to open"""
        
    async def arrive_barrier(self, name: str) -> None:
        """Signal arrival at barrier"""
        
    @asynccontextmanager
    async def lease(self, name: str, ttl: Optional[int] = None) -> AsyncGenerator[Lease, None]:
        """Context manager for lease acquisition"""
```

### Usage Examples
```python
import konductor

# Semaphore with context manager
async def process_with_rate_limit():
    client = konductor.Client()
    
    async with client.semaphore("api-quota", ttl=300) as permit:
        await call_external_api()
        # Automatic release

# Decorator pattern
@konductor.with_semaphore("heavy-computation", permits=2)
async def heavy_task(data):
    return await compute_intensive_operation(data)

# Barrier coordination
async def coordinate_services():
    client = konductor.Client()
    
    # Wait for dependencies
    await client.wait_barrier("dependencies-ready", timeout=600)
    
    # Do work
    await process_data()
    
    # Signal completion
    await client.arrive_barrier("processing-complete")
```

## Node.js SDK Design

### Core Interface
```javascript
class KonductorClient {
    constructor(options = {}) {
        this.namespace = options.namespace || 'default';
    }
    
    async withSemaphore(name, options, fn) {
        // Acquire, execute, release pattern
    }
    
    async acquireSemaphore(name, options = {}) {
        // Manual acquisition
    }
    
    async waitBarrier(name, options = {}) {
        // Wait for barrier
    }
    
    async arriveBarrier(name) {
        // Signal barrier arrival
    }
    
    async withLease(name, options, fn) {
        // Lease with automatic release
    }
}
```

### Usage Examples
```javascript
const konductor = require('@konductor/sdk');

// Express middleware
app.use('/api/heavy', konductor.middleware.semaphore('api-heavy', {
    permits: 5,
    ttl: 300
}));

// Promise-based usage
async function processData() {
    const client = new konductor.Client();
    
    await client.withSemaphore('data-processing', { ttl: 600 }, async () => {
        return await heavyDataProcessing();
    });
}

// Barrier coordination
async function coordinatedWorkflow() {
    const client = new konductor.Client();
    
    await client.waitBarrier('stage-1-complete', { timeout: 1800 });
    await processStage2();
    await client.arriveBarrier('stage-2-complete');
}
```

## Framework Integrations

### HTTP Middleware
```go
// Go Gin middleware
func SemaphoreMiddleware(name string, opts ...konductor.SemaphoreOption) gin.HandlerFunc {
    return func(c *gin.Context) {
        client := konductor.NewClient()
        permit, err := client.AcquireSemaphore(c.Request.Context(), name, opts...)
        if err != nil {
            c.JSON(429, gin.H{"error": "rate limited"})
            return
        }
        defer permit.Release()
        c.Next()
    }
}

// Usage
r.GET("/api/heavy", SemaphoreMiddleware("api-heavy", konductor.WithTTL(5*time.Minute)), handleHeavyRequest)
```

### Database Integration
```python
# SQLAlchemy integration
class CoordinatedEngine:
    def __init__(self, engine, semaphore_name, max_connections=10):
        self.engine = engine
        self.semaphore_name = semaphore_name
        self.client = konductor.Client()
    
    @asynccontextmanager
    async def connect(self):
        async with self.client.semaphore(self.semaphore_name):
            async with self.engine.connect() as conn:
                yield conn
```

### Message Queue Integration
```javascript
// Bull queue with coordination
const Queue = require('bull');
const konductor = require('@konductor/sdk');

const processQueue = new Queue('process data');

processQueue.process(async (job) => {
    const client = new konductor.Client();
    
    await client.withSemaphore('external-api', { permits: 3 }, async () => {
        return await processJobData(job.data);
    });
});
```

## Implementation Phases

### Phase 1: Core SDKs
- **Go SDK**: Full feature set (primary language)
- **Python SDK**: Async support, context managers
- **CLI**: Enhanced with SDK-like features

### Phase 2: Framework Integration
- HTTP middleware for major frameworks
- Database connection pool integration
- Message queue coordination

### Phase 3: Advanced Features
- **Node.js SDK**: Full feature parity
- **Java SDK**: Spring Boot integration
- **Rust SDK**: High-performance use cases

### Phase 4: Ecosystem
- **Observability**: Metrics, tracing integration
- **Testing**: Mock implementations for unit tests
- **Documentation**: Comprehensive guides and examples

## SDK Benefits Over CLI-Only

| Aspect | CLI Only | With SDKs |
|--------|----------|-----------|
| **Integration** | Shell scripts only | Native application code |
| **Performance** | Process overhead | In-process coordination |
| **Error Handling** | Exit codes | Native exceptions |
| **Resource Management** | Manual cleanup | Automatic (defer/context managers) |
| **Framework Integration** | Impossible | Native middleware |
| **Development Experience** | Awkward | Natural language patterns |
| **Testing** | Complex mocking | Easy unit testing |

## Competitive Advantage

**With SDKs, konductor becomes:**
- **Job coordination platform**: Applications coordinate at appropriate granularity
- **Language agnostic**: Works across technology stacks for batch/job workloads
- **Batch processing friendly**: Integrates with job processing patterns
- **Developer focused**: Natural APIs for job-level coordination

**This positions konductor as infrastructure for job and batch coordination, not real-time request processing.**

## Success Metrics for SDKs

### Adoption Metrics
- Downloads per language SDK
- GitHub stars and contributions
- Integration examples in the wild

### Technical Metrics
- SDK performance vs CLI overhead
- Memory usage and resource efficiency
- Error rates and reliability

### Developer Experience
- Time to first successful integration
- Documentation completeness
- Community feedback and issues

---

**Bottom Line**: SDKs transform konductor from a niche coordination tool into a **coordination platform** that any application can use, dramatically expanding its addressable market and value proposition.