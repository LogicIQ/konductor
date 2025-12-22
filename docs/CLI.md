# Konductor CLI Documentation

Complete reference for the Konductor command-line interface (koncli).

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Global Flags](#global-flags)
- [Commands](#commands)
  - [Semaphore](#semaphore)
  - [Barrier](#barrier)
  - [Lease](#lease)
  - [Gate](#gate)
  - [Status](#status)
  - [Operator](#operator)
- [Output Formats](#output-formats)
- [Examples](#examples)
- [Scripting](#scripting)

## Installation

### From Release

```bash
# Linux AMD64
curl -L https://github.com/LogicIQ/konductor/releases/latest/download/koncli-linux-amd64 -o koncli
chmod +x koncli
sudo mv koncli /usr/local/bin/

# macOS ARM64
curl -L https://github.com/LogicIQ/konductor/releases/latest/download/koncli-darwin-arm64 -o koncli
chmod +x koncli
sudo mv koncli /usr/local/bin/
```

### From Source

```bash
go install github.com/LogicIQ/konductor/cli@latest
```

### Docker

```bash
docker run logiciq/koncli:latest semaphore list
```

## Configuration

Configuration priority (highest to lowest):

1. Command-line flags
2. Environment variables (prefix: `KONCLI_`)
3. Config file: `~/.konductor/koncli.yaml`
4. Defaults

### Config File

```yaml
# ~/.konductor/koncli.yaml
namespace: production
log-level: info
output: text
kubeconfig: /path/to/kubeconfig
```

### Environment Variables

```bash
export KONCLI_NAMESPACE=production
export KONCLI_LOG_LEVEL=debug
export KONCLI_OUTPUT=json
```

## Global Flags

```
--kubeconfig string      Path to kubeconfig file
-n, --namespace string   Kubernetes namespace (default: auto-detected)
--log-level string       Log level: debug, info, warn, error (default: "info")
-o, --output string      Output format: text, json (default: "text")
-h, --help              Help for any command
```

## Commands

### Semaphore

Manage semaphores for rate limiting and resource control.

#### Create

```bash
koncli semaphore create NAME --permits COUNT [flags]

Flags:
  --permits int     Number of permits (required)
  --ttl duration    Default TTL for permits (e.g., 5m, 1h)

Examples:
  koncli semaphore create api-limit --permits 10
  koncli semaphore create db-pool --permits 20 --ttl 10m
```

#### Acquire

```bash
koncli semaphore acquire NAME [flags]

Flags:
  --holder string     Holder identifier (default: hostname-pid)
  --timeout duration  Wait timeout (default: 30s)
  --ttl duration      Permit TTL (default: 5m)

Examples:
  koncli semaphore acquire api-limit
  koncli semaphore acquire api-limit --holder my-app --timeout 1m
```

#### Release

```bash
koncli semaphore release NAME [flags]

Flags:
  --holder string  Holder identifier (default: hostname-pid)

Examples:
  koncli semaphore release api-limit
  koncli semaphore release api-limit --holder my-app
```

#### List

```bash
koncli semaphore list [flags]

Examples:
  koncli semaphore list
  koncli semaphore list -o json
```

#### Delete

```bash
koncli semaphore delete NAME

Examples:
  koncli semaphore delete api-limit
```

### Barrier

Manage barriers for coordinating multiple processes.

#### Create

```bash
koncli barrier create NAME --expected COUNT [flags]

Flags:
  --expected int      Number of expected arrivals (required)
  --timeout duration  Barrier timeout (e.g., 10m, 1h)
  --quorum int        Minimum arrivals to open (optional)

Examples:
  koncli barrier create stage-1 --expected 5
  koncli barrier create stage-1 --expected 5 --timeout 30m
  koncli barrier create stage-1 --expected 5 --quorum 3
```

#### Arrive

```bash
koncli barrier arrive NAME [flags]

Flags:
  --holder string  Holder identifier (default: hostname-pid)

Examples:
  koncli barrier arrive stage-1
  koncli barrier arrive stage-1 --holder worker-1
```

#### Wait

```bash
koncli barrier wait NAME [flags]

Flags:
  --timeout duration  Wait timeout (default: 5m)

Examples:
  koncli barrier wait stage-1
  koncli barrier wait stage-1 --timeout 10m
```

#### List

```bash
koncli barrier list [flags]

Examples:
  koncli barrier list
  koncli barrier list -o json
```

#### Delete

```bash
koncli barrier delete NAME

Examples:
  koncli barrier delete stage-1
```

### Lease

Manage leases for leader election and singleton execution.

#### Create

```bash
koncli lease create NAME --ttl DURATION [flags]

Flags:
  --ttl duration      Lease TTL (required, e.g., 30s, 5m)
  --priority int      Priority for acquisition (default: 0)

Examples:
  koncli lease create leader --ttl 30s
  koncli lease create migration --ttl 10m --priority 10
```

#### Acquire

```bash
koncli lease acquire NAME [flags]

Flags:
  --holder string     Holder identifier (default: hostname-pid)
  --timeout duration  Wait timeout (default: 30s)
  --priority int      Priority for acquisition (default: 0)

Examples:
  koncli lease acquire leader
  koncli lease acquire leader --holder instance-1 --timeout 1m
```

#### Release

```bash
koncli lease release NAME [flags]

Flags:
  --holder string  Holder identifier (default: hostname-pid)

Examples:
  koncli lease release leader
  koncli lease release leader --holder instance-1
```

#### List

```bash
koncli lease list [flags]

Examples:
  koncli lease list
  koncli lease list -o json
```

#### Delete

```bash
koncli lease delete NAME

Examples:
  koncli lease delete leader
```

### Gate

Manage gates for dependency-based coordination.

#### Create

```bash
koncli gate create NAME [flags]

Flags:
  --timeout duration  Gate timeout (e.g., 10m, 1h)

Examples:
  koncli gate create dependencies
  koncli gate create dependencies --timeout 30m
```

#### Open

```bash
koncli gate open NAME

Examples:
  koncli gate open dependencies
```

#### Close

```bash
koncli gate close NAME

Examples:
  koncli gate close dependencies
```

#### Wait

```bash
koncli gate wait NAME [flags]

Flags:
  --timeout duration  Wait timeout (default: 5m)

Examples:
  koncli gate wait dependencies
  koncli gate wait dependencies --timeout 10m
```

#### List

```bash
koncli gate list [flags]

Examples:
  koncli gate list
  koncli gate list -o json
```

#### Delete

```bash
koncli gate delete NAME

Examples:
  koncli gate delete dependencies
```

### Status

View detailed status of coordination primitives.

#### All

```bash
koncli status all [flags]

Examples:
  koncli status all
  koncli status all -o json
```

#### Specific Resource

```bash
koncli status TYPE NAME [flags]

Types: semaphore, barrier, lease, gate

Examples:
  koncli status semaphore api-limit
  koncli status barrier stage-1
  koncli status lease leader
  koncli status gate dependencies
```

### Operator

Check operator health and status.

```bash
koncli operator [flags]

Examples:
  koncli operator
  koncli operator -o json
```

## Output Formats

### Text Format (default)

Human-readable console output with colors:

```bash
$ koncli semaphore list
INFO  Semaphore  name=api-limit permits=10 in-use=3 available=7 phase=Ready
INFO  Semaphore  name=db-pool permits=20 in-use=15 available=5 phase=Ready
```

### JSON Format

Structured JSON output for scripting:

```bash
$ koncli semaphore list -o json
{"level":"info","msg":"Semaphore","name":"api-limit","permits":10,"in-use":3,"available":7,"phase":"Ready"}
{"level":"info","msg":"Semaphore","name":"db-pool","permits":20,"in-use":15,"available":5,"phase":"Ready"}
```

Parse with `jq`:

```bash
koncli semaphore list -o json | jq -r '.name'
```

## Examples

### Rate Limiting

```bash
# Create semaphore
koncli semaphore create api-limit --permits 10

# Acquire permit
koncli semaphore acquire api-limit --holder my-app

# Do work
curl https://api.example.com/data

# Release permit
koncli semaphore release api-limit --holder my-app
```

### Multi-Stage Pipeline

```bash
# Worker 1
koncli barrier arrive stage-1 --holder worker-1
koncli barrier wait stage-1

# Worker 2
koncli barrier arrive stage-1 --holder worker-2
koncli barrier wait stage-1

# Worker 3
koncli barrier arrive stage-1 --holder worker-3
koncli barrier wait stage-1
```

### Leader Election

```bash
# Try to become leader
if koncli lease acquire leader --holder $HOSTNAME; then
  echo "I am the leader"
  run_leader_tasks
  koncli lease release leader --holder $HOSTNAME
else
  echo "Another instance is the leader"
fi
```

### Dependency Coordination

```bash
# Wait for dependencies
koncli gate wait dependencies --timeout 5m

# Start service
start_service

# Signal ready
koncli gate open service-ready
```

## Scripting

### Exit Codes

- `0` - Success
- `1` - Error

### Error Handling

```bash
#!/bin/bash
set -e

if koncli semaphore acquire api-limit --holder $HOSTNAME; then
  trap "koncli semaphore release api-limit --holder $HOSTNAME" EXIT
  
  # Do work
  call_api
else
  echo "Failed to acquire permit"
  exit 1
fi
```

### JSON Parsing

```bash
# Get available permits
AVAILABLE=$(koncli status semaphore api-limit -o json | jq -r '.available')

if [ "$AVAILABLE" -gt 0 ]; then
  echo "Permits available: $AVAILABLE"
else
  echo "No permits available"
fi
```

### Retry Logic

```bash
#!/bin/bash

MAX_RETRIES=5
RETRY_DELAY=5

for i in $(seq 1 $MAX_RETRIES); do
  if koncli semaphore acquire api-limit --holder $HOSTNAME; then
    echo "Acquired permit"
    break
  fi
  
  if [ $i -eq $MAX_RETRIES ]; then
    echo "Failed after $MAX_RETRIES retries"
    exit 1
  fi
  
  echo "Retry $i/$MAX_RETRIES"
  sleep $RETRY_DELAY
done
```

### Monitoring

```bash
#!/bin/bash

# Monitor semaphore utilization
while true; do
  koncli semaphore list -o json | jq -r '
    select(.inUse / .permits > 0.8) |
    "WARNING: \(.name) is \(.inUse)/\(.permits) (\(.inUse / .permits * 100)%)"
  '
  sleep 60
done
```

## Namespace Detection

The CLI automatically detects the namespace in the following order:

1. `--namespace` flag
2. `KONCLI_NAMESPACE` environment variable
3. Service account namespace (when running in a pod)
4. `POD_NAMESPACE` environment variable
5. `NAMESPACE` environment variable
6. Kubeconfig context namespace
7. `default` namespace

## Troubleshooting

### Debug Mode

```bash
koncli --log-level debug semaphore list
```

### Check Operator

```bash
koncli operator
```

### View Resource Details

```bash
# Detailed status
koncli status semaphore api-limit

# Raw Kubernetes resource
kubectl get semaphore api-limit -o yaml
```

### Common Issues

#### "Failed to connect to cluster"

Check kubeconfig:
```bash
kubectl cluster-info
koncli --kubeconfig ~/.kube/config semaphore list
```

#### "Resource not found"

Check namespace:
```bash
koncli -n production semaphore list
```

#### "Timeout waiting for permit"

Check semaphore status:
```bash
koncli status semaphore api-limit
```

## Advanced Usage

### Custom Holder IDs

```bash
HOLDER="$POD_NAME-$RANDOM"
koncli semaphore acquire api-limit --holder "$HOLDER"
```

### Conditional Execution

```bash
if koncli lease acquire leader --holder $HOSTNAME --timeout 0; then
  # Non-blocking acquire succeeded
  run_as_leader
  koncli lease release leader --holder $HOSTNAME
fi
```

### Pipeline Integration

```bash
# GitHub Actions
- name: Acquire deployment lock
  run: koncli lease acquire deploy-lock --holder $GITHUB_RUN_ID

# GitLab CI
script:
  - koncli semaphore acquire deploy-quota --holder $CI_JOB_ID
```

## See Also

- [CLI Examples](./examples/README.md) - Real-world usage scenarios
- [SDK Documentation](../docs/SDK.md) - Go SDK integration
- [Main README](../README.md) - Project overview

## Contributing

Contributions welcome! Please see the main repository for guidelines.

## License

Apache 2.0
