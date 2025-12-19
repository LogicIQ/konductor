# Konductor CLI (koncli)

Command-line interface for managing Konductor synchronization primitives.

## Installation

```bash
cd cli
go build -o koncli .
```

## Usage

```bash
koncli [command] [flags]
```

## Global Flags

- `--kubeconfig string` - Path to kubeconfig file
- `-n, --namespace string` - Kubernetes namespace (default: auto-detected or "default")
- `--log-level string` - Log level: debug, info, warn, error (default: "info")
- `-o, --output string` - Output format: text, json (default: "text")

## Output Formats

### Text Format (default)
Human-readable console output with colors:

```bash
koncli semaphore list
# Output:
# INFO  Semaphore  name=my-semaphore permits=5 in-use=2 available=3 phase=Ready
```

### JSON Format
Structured JSON output for scripting and automation:

```bash
koncli semaphore list -o json
# Output:
# {"level":"info","msg":"Semaphore","name":"my-semaphore","permits":5,"in-use":2,"available":3,"phase":"Ready"}
```

## Commands

### Semaphore

Manage semaphores for rate limiting and resource control.

```bash
# Create a semaphore
koncli semaphore create my-sem --permits 5

# Acquire a permit
koncli semaphore acquire my-sem --holder my-app

# Release a permit
koncli semaphore release my-sem --holder my-app

# List semaphores
koncli semaphore list

# Delete a semaphore
koncli semaphore delete my-sem
```

### Barrier

Manage barriers for coordinating multiple processes.

```bash
# Create a barrier
koncli barrier create my-barrier --expected 3

# Signal arrival
koncli barrier arrive my-barrier --holder worker-1

# Wait for barrier to open
koncli barrier wait my-barrier

# List barriers
koncli barrier list

# Delete a barrier
koncli barrier delete my-barrier
```

### Lease

Manage leases for leader election and singleton execution.

```bash
# Create a lease
koncli lease create my-lease --ttl 10m

# Acquire a lease
koncli lease acquire my-lease --holder my-app

# Release a lease
koncli lease release my-lease --holder my-app

# List leases
koncli lease list

# Delete a lease
koncli lease delete my-lease
```

### Gate

Manage gates for dependency-based coordination.

```bash
# Create a gate
koncli gate create my-gate

# Open a gate
koncli gate open my-gate

# Close a gate
koncli gate close my-gate

# Wait for gate to open
koncli gate wait my-gate

# List gates
koncli gate list

# Delete a gate
koncli gate delete my-gate
```

### Status

View detailed status of coordination primitives.

```bash
# Show all primitives
koncli status all

# Show specific semaphore
koncli status semaphore my-sem

# Show specific barrier
koncli status barrier my-barrier

# Show specific lease
koncli status lease my-lease

# Show specific gate
koncli status gate my-gate
```

### Operator

Check operator health and status.

```bash
koncli operator
```

## Examples

### Rate Limiting with Semaphore

```bash
# Create semaphore with 3 permits
koncli semaphore create api-limit --permits 3

# Acquire permit before making API call
koncli semaphore acquire api-limit --holder worker-1
# ... make API call ...
koncli semaphore release api-limit --holder worker-1
```

### Coordinating Multiple Workers with Barrier

```bash
# Create barrier expecting 3 workers
koncli barrier create sync-point --expected 3

# Each worker signals arrival
koncli barrier arrive sync-point --holder worker-1
koncli barrier arrive sync-point --holder worker-2
koncli barrier arrive sync-point --holder worker-3

# All workers wait for barrier to open
koncli barrier wait sync-point
```

### Leader Election with Lease

```bash
# Create lease
koncli lease create leader --ttl 30s

# Try to acquire lease
if koncli lease acquire leader --holder instance-1; then
  echo "I am the leader"
  # ... do leader work ...
  koncli lease release leader --holder instance-1
fi
```

### JSON Output for Automation

```bash
# Get semaphore status as JSON
koncli semaphore list -o json | jq '.name'

# Check if barrier is open
koncli status barrier my-barrier -o json | jq -r '.phase'
```

## Configuration

Configuration can be provided via:
1. Command-line flags
2. Environment variables (prefix: `KONCLI_`)
3. Config file: `~/.konductor/koncli.yaml`

Example config file:

```yaml
namespace: my-namespace
log-level: debug
output: json
```

## Namespace Detection

The CLI automatically detects the namespace in the following order:
1. Service account namespace (when running in a pod)
2. `POD_NAMESPACE` environment variable
3. `NAMESPACE` environment variable
4. Kubeconfig context namespace
5. Default namespace

## Exit Codes

- `0` - Success
- `1` - Error

## Logging

All output uses structured logging via zap:
- **Text format**: Human-readable console output with colors
- **JSON format**: Machine-readable JSON lines

Log levels:
- `debug` - Detailed debugging information
- `info` - General informational messages (default)
- `warn` - Warning messages
- `error` - Error messages
