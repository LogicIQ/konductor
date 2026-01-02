# Mutex CLI Commands

Detailed reference for mutex-related CLI commands.

## Commands

### lock

Acquire a mutex lock.

```bash
koncli mutex lock <name> [flags]
```

**Flags:**
- `--holder string` - Holder identifier (default: auto-detected)
- `--timeout duration` - Wait timeout (default: 30s)
- `--ttl duration` - Lock TTL (default: 5m)
- `--wait` - Wait for lock if not available

**Examples:**
```bash
# Basic lock
koncli mutex lock db-migration

# With custom holder and TTL
koncli mutex lock db-migration --holder $HOSTNAME --ttl 10m

# Wait for lock
koncli mutex lock db-migration --wait --timeout 1m
```

### unlock

Release a mutex lock.

```bash
koncli mutex unlock <name> [flags]
```

**Flags:**
- `--holder string` - Holder identifier (default: auto-detected)

**Examples:**
```bash
# Unlock
koncli mutex unlock db-migration --holder $HOSTNAME
```

### status

Check mutex status.

```bash
koncli mutex status <name> [flags]
```

**Examples:**
```bash
# Check status
koncli mutex status db-migration

# JSON output
koncli mutex status db-migration -o json
```

## Usage Patterns

### Critical Section
```bash
#!/bin/bash
MUTEX="file-writer"
HOLDER="$HOSTNAME-$$"

if koncli mutex lock $MUTEX --holder $HOLDER --wait --timeout 30s; then
  trap "koncli mutex unlock $MUTEX --holder $HOLDER" EXIT
  write-to-shared-file
fi
```

### Database Migration
```bash
#!/bin/bash
if koncli mutex lock db-migration --holder $HOSTNAME --timeout 30s; then
  trap "koncli mutex unlock db-migration --holder $HOSTNAME" EXIT
  run-migrations
else
  echo "Migration already running"
fi
```

### Try-Lock Pattern
```bash
#!/bin/bash
if koncli mutex lock my-mutex --holder $HOSTNAME --timeout 0; then
  trap "koncli mutex unlock my-mutex --holder $HOSTNAME" EXIT
  do-critical-work
else
  echo "Lock busy, skipping"
fi
```

## Related Commands

- [Lease CLI](./lease.md) - Lease management
- [Semaphore CLI](./semaphore.md) - Semaphore commands
- [CLI Overview](./overview.md) - Complete CLI reference
