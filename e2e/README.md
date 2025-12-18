# E2E Tests

End-to-end tests for Konductor synchronization primitives using Kind and the CLI.

## Quick Start

Run the complete e2e test suite:
```bash
task e2e:run
```

## Commands

- `task e2e:setup` - Setup Kind cluster and deploy operator
- `task e2e:test` - Run tests (assumes cluster exists)
- `task e2e:cleanup` - Delete Kind cluster
- `task e2e:run` - Complete test cycle (setup + test + cleanup)

## Test Coverage

The e2e tests cover:

- **Semaphore**: Create, acquire permits, release permits, delete
- **Barrier**: Create, arrive at barrier, wait for completion, delete  
- **Lease**: Create, acquire exclusive lease, release lease, delete
- **Gate**: Create, open/close gate, verify state, delete

All tests use the `koncli` CLI tool to interact with the Kubernetes API, simulating real-world usage patterns.

## Requirements

- Go 1.21+
- Docker
- kubectl
- Kind (installed automatically)

## Architecture

Tests run against a local Kind cluster with the Konductor operator deployed. The CLI binary is built and used to create/manage sync primitives, while the Go test client verifies the expected Kubernetes resource states.