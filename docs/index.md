# Konductor

![Konductor](./images/konductor.webp)

**Konductor** is a Kubernetes operator that provides synchronization primitives for coordinating jobs and workloads. It enables controlled access, ordering, and coordination between distributed workloads without breaking Kubernetes semantics.

## What is Konductor?

Konductor solves coordination problems in Kubernetes by providing four core primitives:

- **[Semaphore](./api/semaphore.md)** - Limit concurrent access to shared resources
- **[Barrier](./api/barrier.md)** - Coordinate multi-stage workflows  
- **[Lease](./api/lease.md)** - Singleton execution and leader election
- **[Gate](./api/gate.md)** - Dependency coordination

## Why Konductor?

Kubernetes Jobs are powerful but lack built-in coordination. When you need to:

- Wait for multiple Jobs to complete before starting the next stage
- Prevent CronJobs from overlapping when they run longer than their schedule
- Ensure only one Job runs database migrations across multiple replicas
- Limit how many batch Jobs run concurrently to avoid overwhelming your cluster
- Coordinate Pods with each other using CLI or SDK

Konductor provides simple primitives to solve these problems natively in Kubernetes.

## Key Features

- **Kubernetes Native** - CRDs, kubectl, GitOps-friendly
- **Self-Enforced** - Pods voluntarily coordinate their own progress
- **Observable** - All state visible via Kubernetes APIs
- **Crash-Safe** - Automatic cleanup and TTL expiration
- **Lightweight** - Minimal operator with efficient reconciliation

## Quick Start

### Installation

```bash
# Add the LogicIQ Helm repository
helm repo add logiciq https://logiciq.github.io/helm-charts
helm repo update

# Install konductor
helm install my-konductor logiciq/konductor
```

### Basic Usage

```yaml
# Create a barrier for 3 Jobs
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: stage-1-complete
spec:
  expected: 3
```

```bash
# Wait for barrier in Job
koncli barrier wait stage-1-complete --timeout 30m
```

## Getting Started

- **[Installation Guide](./guides/installation.md)** - Install Konductor in your cluster
- **[Quick Start](./guides/quick-start.md)** - Get up and running in 5 minutes
- **[Basic Concepts](./introduction/concepts.md)** - Understand the core primitives
- **[CLI Usage](./guides/cli-usage.md)** - Command-line tool reference

## Use Cases

- **[ETL Pipelines](./examples/etl-pipeline.md)** - Multi-stage data processing
- **[Batch Processing](./examples/batch-processing.md)** - Controlled concurrent execution
- **[Database Migrations](./examples/database-migrations.md)** - Singleton execution patterns
- **[Job Dependencies](./examples/job-dependencies.md)** - Complex workflow coordination

## Community

- **GitHub**: [LogicIQ/konductor](https://github.com/LogicIQ/konductor)
- **Issues**: [Report bugs and request features](https://github.com/LogicIQ/konductor/issues)
- **Discussions**: [Community discussions](https://github.com/LogicIQ/konductor/discussions)