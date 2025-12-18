# Konductor Product Specification

## Overview
Konductor is a Kubernetes-native operator providing synchronization primitives for coordinating jobs and workloads. It enables controlled access, ordering, and coordination between distributed workloads without breaking Kubernetes semantics.

## Problem Statement
Kubernetes lacks built-in coordination primitives for:
- Limiting concurrent access to shared resources (semaphores)
- Coordinating multi-stage batch pipelines (barriers)
- Ensuring singleton execution (leases)
- Managing dependencies between jobs (gates)

Current solutions either break Kubernetes patterns or require complex custom implementations.

## Core Design Principles

### 1. Kubernetes-Native
- Everything achievable via `kubectl apply`
- CRDs as primary interface
- GitOps-friendly YAML configuration
- Controller reconciliation driven

### 2. Self-Enforced Coordination
- Operator provides state and arbitration, not control
- Pods voluntarily gate their own progress
- No external blocking or pausing of containers
- Idempotent and crash-safe operations

### 3. Observable and Debuggable
- All coordination state visible via Kubernetes APIs
- Clear ownership and TTL semantics
- Human-readable status conditions

## Core Primitives

### 1. Semaphore (Priority 1)
**Purpose**: Limit concurrent access to shared resources

```yaml
apiVersion: sync.konductor.io/v1
kind: Semaphore
metadata:
  name: api-quota
spec:
  permits: 5
  ttl: 10m
status:
  inUse: 3
  available: 2
  phase: Ready
```

**Use Cases**:
- Throttle external API calls
- Limit database connections
- Control expensive operations

### 2. Barrier (Priority 1)
**Purpose**: Coordinate multi-stage workflows

```yaml
apiVersion: sync.konductor.io/v1
kind: Barrier
metadata:
  name: stage-2
spec:
  expected: 10
  timeout: 30m
  quorum: 8  # optional
status:
  arrived: 7
  phase: Waiting | Open | Failed
```

**Use Cases**:
- Fan-out/fan-in data processing
- Multi-stage batch pipelines
- Coordinated deployments

### 3. Lease (Priority 2)
**Purpose**: Singleton execution and leader election

```yaml
apiVersion: sync.konductor.io/v1
kind: Lease
metadata:
  name: db-migration
spec:
  ttl: 5m
  priority: 1
status:
  holder: pod-xyz-123
  acquired: "2024-01-15T10:30:00Z"
  phase: Held | Available
```

**Use Cases**:
- Database migrations
- Leader election
- Singleton services

### 4. Gate (Priority 3)
**Purpose**: Dependency coordination

```yaml
apiVersion: sync.konductor.io/v1
kind: Gate
metadata:
  name: processing-gate
spec:
  conditions:
    - job: data-loader
      state: Complete
    - semaphore: api-quota
      permits: 2
status:
  phase: Waiting | Open | Failed
```

## Pod Integration Patterns

### Pattern A: InitContainer Gate (Recommended)
```yaml
initContainers:
- name: acquire-semaphore
  image: konductor/koncli:latest
  command:
    - koncli
    - semaphore
    - acquire
    - api-quota
    - --wait
    - --ttl=10m
  # No namespace flag needed - auto-detected from pod!
```

**Benefits**:
- Clean startup gating
- Natural Kubernetes retry behavior
- No running pods blocked

### Pattern B: Fail-Fast + Job Retries
```bash
if ! koncli semaphore acquire api-quota --ttl=5m; then
  exit 1
fi
# Job retries handle contention
```

**Benefits**:
- Zero waiting pods
- Kubernetes handles backoff
- Clean failure semantics

### Pattern C: Sidecar Gate (Advanced)
For long-running services requiring dynamic coordination.

## CLI Design (koncli)

### Core Commands
```bash
# Semaphore operations
koncli semaphore acquire <name> [--wait] [--ttl=5m] [--timeout=30m]
koncli semaphore release <name>

# Barrier operations
koncli barrier wait <name> [--timeout=1h]
koncli barrier arrive <name>

# Lease operations
koncli lease acquire <name> [--ttl=5m] [--priority=1]
koncli lease renew <name>
koncli lease release <name>

# Status operations
koncli status semaphore <name>
koncli status barrier <name>
```

### CLI Responsibilities
- Automatic Pod UID detection
- TTL renewal in background
- Signal handling (SIGTERM cleanup)
- Retry logic with exponential backoff
- Crash-safe ownership management

## Architecture Decision: Minimal Operator vs CLI-Only

### Why We Need an Operator (Minimal)
While tempting to use CRDs + CLI only, an operator is essential for:

1. **Distributed Arbitration**: Resolving conflicts when multiple pods compete for permits
2. **TTL Enforcement**: CRDs are static - only controllers can expire permits automatically  
3. **Status Consistency**: Centralized reconciliation prevents split-brain scenarios
4. **Fairness**: Implementing ordering and priority requires coordination logic

### Minimal Operator Design
**Core Principle**: Operator handles coordination conflicts, CLI handles user interaction

#### Controller Responsibilities (Minimal)
- **Semaphore Controller**: Count valid permits, enforce limits, expire TTLs
- **Barrier Controller**: Track arrivals, evaluate quorum, update phase
- **Lease Controller**: Arbitrate ownership conflicts, handle expiration
- **Gate Controller**: Evaluate conditions, update gate status

#### What Controllers DON'T Do
- No complex business logic
- No direct pod interaction
- No workflow orchestration
- No external system integration

### Key Design Decisions
- **CLI-First Interface**: All user operations via koncli
- **Operator as Arbiter**: Only handles distributed coordination problems
- **TTL Everywhere**: All ownership has time bounds
- **Owner References**: Automatic cleanup on pod deletion
- **Status-Driven**: Pods poll status, don't block on API calls

## Implementation Phases

### Phase 1: MVP (Semaphore + Barrier)
- Core CRDs with minimal controllers
- Basic CLI with acquire/wait/release
- InitContainer integration pattern
- Documentation and examples

### Alternative: CLI-Only Approach (Not Recommended)
**Why this doesn't work:**
- Race conditions between competing CLIs
- No automatic TTL enforcement
- Complex distributed consensus in CLI
- Split-brain scenarios with permit counting
- Manual cleanup of dead permits

### Phase 2: Enhanced Features
- Lease primitive
- Priority and preemption
- Quorum support for barriers
- Advanced CLI features

### Phase 3: Advanced Coordination
- Gate primitive
- Cross-namespace coordination
- Metrics and observability
- Performance optimizations

## Success Metrics

### Technical Metrics
- Sub-second permit acquisition latency
- Zero coordination state loss during operator restarts
- 99.9% TTL accuracy
- Support for 1000+ concurrent permits per semaphore

### Adoption Metrics
- Integration with major batch processing frameworks
- Community contributions and extensions
- Production deployments across different industries

## Competitive Positioning

| Solution | Kubernetes Native | Declarative | Observable | Crash Safe |
|----------|-------------------|-------------|------------|------------|
| konductor | ✅ | ✅ | ✅ | ✅ |
| etcd locks | ❌ | ❌ | ❌ | ⚠️ |
| Redis locks | ❌ | ❌ | ❌ | ❌ |
| Argo Workflows | ✅ | ✅ | ✅ | ⚠️ |
| Native Lease | ✅ | ⚠️ | ⚠️ | ✅ |

## Risk Assessment

### Technical Risks
- **Controller Performance**: Mitigate with efficient reconciliation and caching
- **Split Brain**: Prevent with proper leader election and TTL enforcement
- **Resource Exhaustion**: Implement rate limiting and resource quotas

### Adoption Risks
- **Learning Curve**: Address with comprehensive documentation and examples
- **Integration Complexity**: Provide clear patterns and SDK
- **Ecosystem Fragmentation**: Focus on interoperability with existing tools

## Next Steps

1. **CRD Schema Design**: Finalize API specifications
2. **Controller Implementation**: Start with Semaphore controller
3. **CLI Development**: Basic acquire/release functionality
4. **Integration Testing**: Validate with real workloads
5. **Documentation**: Usage patterns and best practices

## Success Criteria for MVP

- [ ] Semaphore supports 100 concurrent permits
- [ ] Barrier coordinates 50 parallel jobs
- [ ] CLI handles pod crashes gracefully
- [ ] Integration with standard Job/CronJob patterns
- [ ] Complete documentation with examples
- [ ] Performance benchmarks published

---

*This document serves as the foundational specification for konductor development and should be updated as the product evolves.*