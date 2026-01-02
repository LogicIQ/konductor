# Lease CLI Commands

Detailed reference for lease-related CLI commands.

## Commands

### acquire

Acquire a lease for singleton execution.

```bash
koncli lease acquire <name> [flags]
```

**Flags:**
- `--holder string` - Holder identifier (default: auto-detected)
- `--timeout duration` - Wait timeout (default: 30s)
- `--ttl duration` - Lease TTL (default: 5m)
- `--wait` - Wait for lease if not available
- `--priority int` - Priority for acquisition (default: 1)

**Examples:**
```bash
# Basic acquisition
koncli lease acquire db-migration

# With custom TTL
koncli lease acquire db-migration --holder $HOSTNAME --ttl 15m

# Wait for lease
koncli lease acquire db-migration --wait --timeout 5m
```

### renew

Renew an existing lease.

```bash
koncli lease renew <name> [flags]
```

**Flags:**
- `--holder string` - Holder identifier (default: auto-detected)

**Examples:**
```bash
# Renew lease
koncli lease renew db-migration --holder $HOSTNAME
```

### release

Release a lease.

```bash
koncli lease release <name> [flags]
```

**Flags:**
- `--holder string` - Holder identifier (default: auto-detected)

**Examples:**
```bash
# Release lease
koncli lease release db-migration --holder $HOSTNAME
```

### status

Check lease status.

```bash
koncli lease status <name> [flags]
```

**Examples:**
```bash
# Check status
koncli lease status db-migration

# JSON output
koncli lease status db-migration -o json
```

## Usage Patterns

### Singleton CronJob
```bash
#!/bin/bash
LEASE="daily-report"
HOLDER="$HOSTNAME-$$"

if koncli lease acquire $LEASE --holder $HOLDER --ttl 2h --timeout 0; then
  trap "koncli lease release $LEASE --holder $HOLDER" EXIT
  generate-report
else
  echo "Report already running"
fi
```

### Database Migration
```bash
#!/bin/bash
if koncli lease acquire db-migration --holder $HOSTNAME --wait --timeout 5m; then
  trap "koncli lease release db-migration --holder $HOSTNAME" EXIT
  run-migrations
fi
```

### Leader Election
```bash
#!/bin/bash
LEASE="service-leader"
HOLDER="$HOSTNAME"

if koncli lease acquire $LEASE --holder $HOLDER --ttl 30s; then
  while true; do
    do-leader-work
    koncli lease renew $LEASE --holder $HOLDER || exit 1
    sleep 15
  done
fi
```

## Related Commands

- [Barrier CLI](./barrier.md) - Barrier coordination
- [Mutex CLI](./mutex.md) - Mutex locking
- [CLI Overview](./overview.md) - Complete CLI reference
