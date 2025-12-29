# Semaphore CLI Commands

Detailed reference for semaphore-related CLI commands.

## Commands

### acquire

Acquire a permit from a semaphore.

```bash
koncli semaphore acquire <name> [flags]
```

**Flags:**
- `--holder string` - Holder identifier (default: auto-detected)
- `--timeout duration` - Wait timeout (default: 30s)
- `--ttl duration` - Permit TTL (default: 5m)
- `--wait` - Wait for permit if not immediately available

**Examples:**
```bash
# Basic acquisition
koncli semaphore acquire api-limit

# With custom holder and TTL
koncli semaphore acquire api-limit --holder my-app --ttl 10m

# Wait up to 1 minute for permit
koncli semaphore acquire api-limit --wait --timeout 1m
```

### release

Release a permit back to the semaphore.

```bash
koncli semaphore release <name> [flags]
```

**Flags:**
- `--holder string` - Holder identifier (default: auto-detected)

**Examples:**
```bash
# Release permit
koncli semaphore release api-limit

# Release with specific holder
koncli semaphore release api-limit --holder my-app
```

### status

Check semaphore status and usage.

```bash
koncli semaphore status <name> [flags]
```

**Examples:**
```bash
# Check status
koncli semaphore status api-limit

# JSON output
koncli semaphore status api-limit -o json
```

## Usage Patterns

### Rate Limiting Script
```bash
#!/bin/bash
SEMAPHORE="api-quota"
HOLDER="script-$$"

if koncli semaphore acquire $SEMAPHORE --holder $HOLDER --ttl 5m; then
    trap "koncli semaphore release $SEMAPHORE --holder $HOLDER" EXIT
    
    # Do rate-limited work
    call-external-api
else
    echo "Failed to acquire permit"
    exit 1
fi
```

### Batch Processing
```bash
#!/bin/bash
for item in $(cat batch-items.txt); do
    if koncli semaphore acquire batch-limit --wait --timeout 30s; then
        process-item $item &
    else
        echo "Skipping $item - no permits available"
    fi
done
wait
```

## Related Commands

- [Barrier CLI](./barrier.md) - Barrier coordination commands
- [Lease CLI](./lease.md) - Lease management commands
- [CLI Overview](./overview.md) - Complete CLI reference