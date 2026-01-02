# CLI Reference

The `koncli` command-line tool provides an interface for interacting with Konductor synchronization primitives from shell scripts, containers, and CI/CD pipelines.

## Installation

### Binary Download
```bash
# Download latest release
curl -LO https://github.com/LogicIQ/konductor/releases/latest/download/koncli-linux-amd64
chmod +x koncli-linux-amd64
sudo mv koncli-linux-amd64 /usr/local/bin/koncli
```

### Go Install
```bash
go install github.com/LogicIQ/konductor/cli@latest
```

### Container Image
```bash
docker pull logiciq/koncli:latest
```

## Global Options

| Flag | Description | Default |
|------|-------------|---------|
| `--namespace, -n` | Kubernetes namespace | Current context namespace |
| `--kubeconfig` | Path to kubeconfig file | `$KUBECONFIG` or `~/.kube/config` |
| `--context` | Kubernetes context to use | Current context |
| `--timeout` | Operation timeout | `30s` |
| `--verbose, -v` | Verbose output | `false` |
| `--help, -h` | Show help | - |

## Commands

### Semaphore Commands

```bash
# Acquire a permit
koncli semaphore acquire <name> [flags]

# Release a permit  
koncli semaphore release <name> [flags]

# Check semaphore status
koncli semaphore status <name> [flags]
```

**Flags:**
- `--ttl`: Time-to-live for the permit (default: 5m)
- `--wait`: Wait for permit if not immediately available
- `--holder`: Permit holder identifier (default: auto-detected)

### Barrier Commands

```bash
# Wait for barrier to open
koncli barrier wait <name> [flags]

# Signal arrival at barrier
koncli barrier arrive <name> [flags]

# Check barrier status
koncli barrier status <name> [flags]
```

**Flags:**
- `--timeout`: Maximum time to wait (default: 30m)
- `--arrival-id`: Arrival identifier (default: auto-detected)

### Lease Commands

```bash
# Acquire a lease
koncli lease acquire <name> [flags]

# Renew a lease
koncli lease renew <name> [flags]

# Release a lease
koncli lease release <name> [flags]

# Check lease status
koncli lease status <name> [flags]
```

**Flags:**
- `--ttl`: Time-to-live for the lease (default: 5m)
- `--holder`: Lease holder identifier (default: auto-detected)
- `--priority`: Priority for acquisition (default: 1)
- `--wait`: Wait for lease if not available

### Mutex Commands

```bash
# Lock a mutex
koncli mutex lock <name> [flags]

# Unlock a mutex
koncli mutex unlock <name> [flags]

# Check mutex status
koncli mutex status <name> [flags]
```

**Flags:**
- `--ttl`: Time-to-live for the lock (default: 5m)
- `--holder`: Lock holder identifier (default: auto-detected)
- `--wait`: Wait for lock if not available

### Gate Commands

```bash
# Wait for gate to open
koncli gate wait <name> [flags]

# Check gate status
koncli gate status <name> [flags]
```

**Flags:**
- `--timeout`: Maximum time to wait (default: 30m)

### General Commands

```bash
# Show version information
koncli version

# Show help for any command
koncli <command> --help
```

## Usage Patterns

### InitContainer Pattern
Use in initContainers to gate pod startup:

```yaml
initContainers:
- name: wait-dependencies
  image: logiciq/koncli:latest
  command:
  - koncli
  - barrier
  - wait
  - dependencies-ready
  - --timeout=10m
```

### Script Integration
Use in shell scripts:

```bash
#!/bin/bash
set -e

# Acquire semaphore permit
if koncli semaphore acquire batch-limit --ttl=30m --wait --timeout=5m; then
  echo "Processing batch..."
  process-batch-data
  
  # Release permit
  koncli semaphore release batch-limit
else
  echo "Failed to acquire permit"
  exit 1
fi
```

### CronJob Integration
Prevent overlapping executions:

```bash
#!/bin/bash
LEASE_NAME="daily-backup"
HOLDER_ID="$HOSTNAME-$$"

if koncli lease acquire $LEASE_NAME --holder $HOLDER_ID --ttl=2h --timeout=0; then
  echo "Running backup..."
  run-backup
  koncli lease release $LEASE_NAME --holder $HOLDER_ID
else
  echo "Backup already running, skipping"
fi
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `KUBECONFIG` | Path to kubeconfig file | `~/.kube/config` |
| `KONDUCTOR_NAMESPACE` | Default namespace | Current context |
| `KONDUCTOR_TIMEOUT` | Default timeout | `30s` |
| `KONDUCTOR_HOLDER_ID` | Default holder ID | Auto-detected |

## Auto-Detection

The CLI automatically detects several values when running in Kubernetes:

- **Namespace**: From pod's service account
- **Holder ID**: From pod name and UID
- **Kubeconfig**: From in-cluster service account

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Resource not found |
| 3 | Timeout |
| 4 | Permission denied |
| 5 | Resource unavailable |

## Examples

### Basic Semaphore Usage
```bash
# Acquire permit with 10 minute TTL
koncli semaphore acquire api-quota --ttl=10m

# Wait up to 5 minutes for permit
koncli semaphore acquire api-quota --wait --timeout=5m

# Check current status
koncli semaphore status api-quota

# Release permit
koncli semaphore release api-quota
```

### Barrier Coordination
```bash
# Wait for all extract jobs to complete
koncli barrier wait extract-complete --timeout=1h

# Signal that transform job is complete
koncli barrier arrive transform-complete
```

### Lease Management
```bash
# Try to acquire lease immediately
if koncli lease acquire db-migration --timeout=0; then
  run-migration
  koncli lease release db-migration
fi

# Acquire with wait and custom holder
koncli lease acquire leader-election --holder=$POD_NAME --wait
```

## Troubleshooting

### Connection Issues
```bash
# Test cluster connectivity
koncli version

# Use specific kubeconfig
koncli --kubeconfig=/path/to/config semaphore status my-semaphore

# Use specific context
koncli --context=production barrier wait my-barrier
```

### Permission Issues
```bash
# Check RBAC permissions
kubectl auth can-i get semaphores
kubectl auth can-i update semaphores

# Use service account with proper permissions
kubectl create serviceaccount konductor-user
kubectl create clusterrolebinding konductor-user --clusterrole=konductor-user --serviceaccount=default:konductor-user
```

### Debugging
```bash
# Enable verbose output
koncli -v semaphore acquire my-semaphore

# Check resource status directly
kubectl get semaphore my-semaphore -o yaml
kubectl describe barrier my-barrier
```

## Best Practices

1. **Use meaningful names**: Choose descriptive resource names
2. **Set appropriate timeouts**: Balance responsiveness with reliability
3. **Handle errors gracefully**: Always check exit codes
4. **Clean up resources**: Release permits and leases in error handlers
5. **Use holder IDs**: Provide unique identifiers for debugging
6. **Monitor operations**: Log important coordination events

## Related Documentation

- [Semaphore CLI](./semaphore.md) - Detailed semaphore commands
- [Barrier CLI](./barrier.md) - Detailed barrier commands  
- [Lease CLI](./lease.md) - Detailed lease commands
- [Mutex CLI](./mutex.md) - Detailed mutex commands
- [Gate CLI](./gate.md) - Detailed gate commands
- [Examples](../examples/overview.md) - Real-world usage examples