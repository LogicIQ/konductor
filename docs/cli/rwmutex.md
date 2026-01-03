# RWMutex CLI Commands

Detailed reference for read-write mutex CLI commands.

## Commands

### rlock

Acquire a read lock (multiple readers allowed).

```bash
koncli rwmutex rlock <name> [flags]
```

**Flags:**
- `--holder string` - Holder identifier (default: auto-detected)
- `--timeout duration` - Wait timeout (default: 30s)

**Examples:**
```bash
# Basic read lock
koncli rwmutex rlock cache-lock

# With custom holder
koncli rwmutex rlock cache-lock --holder reader-1

# Wait for lock
koncli rwmutex rlock cache-lock --timeout 1m
```

### lock

Acquire a write lock (exclusive access).

```bash
koncli rwmutex lock <name> [flags]
```

**Flags:**
- `--holder string` - Holder identifier (default: auto-detected)
- `--timeout duration` - Wait timeout (default: 30s)

**Examples:**
```bash
# Basic write lock
koncli rwmutex lock cache-lock

# With custom holder and timeout
koncli rwmutex lock cache-lock --holder writer-1 --timeout 30s
```

### unlock

Release a read or write lock.

```bash
koncli rwmutex unlock <name> [flags]
```

**Flags:**
- `--holder string` - Holder identifier (default: auto-detected)

**Examples:**
```bash
# Unlock
koncli rwmutex unlock cache-lock --holder $HOSTNAME
```

### create

Create a new rwmutex.

```bash
koncli rwmutex create <name> [flags]
```

**Flags:**
- `--ttl duration` - Optional TTL for automatic unlock

**Examples:**
```bash
# Create rwmutex
koncli rwmutex create cache-lock

# With TTL
koncli rwmutex create cache-lock --ttl 5m
```

### delete

Delete a rwmutex.

```bash
koncli rwmutex delete <name>
```

**Examples:**
```bash
koncli rwmutex delete cache-lock
```

### list

List all rwmutexes.

```bash
koncli rwmutex list [flags]
```

**Examples:**
```bash
# List all
koncli rwmutex list

# With namespace
koncli rwmutex list -n production
```

## Usage Patterns

### Cache Read Pattern
```bash
#!/bin/bash
RWMUTEX="cache-lock"
HOLDER="reader-$HOSTNAME"

if koncli rwmutex rlock $RWMUTEX --holder $HOLDER --timeout 10s; then
  trap "koncli rwmutex unlock $RWMUTEX --holder $HOLDER" EXIT
  read-from-cache
fi
```

### Cache Write Pattern
```bash
#!/bin/bash
RWMUTEX="cache-lock"
HOLDER="writer-$HOSTNAME"

if koncli rwmutex lock $RWMUTEX --holder $HOLDER --timeout 30s; then
  trap "koncli rwmutex unlock $RWMUTEX --holder $HOLDER" EXIT
  update-cache
fi
```

### Multiple Readers
```bash
#!/bin/bash
# Multiple readers can acquire lock simultaneously
for i in {1..5}; do
  (
    koncli rwmutex rlock data-lock --holder "reader-$i"
    echo "Reader $i: reading data"
    sleep 2
    koncli rwmutex unlock data-lock --holder "reader-$i"
  ) &
done
wait
```

### Read-Write Coordination
```bash
#!/bin/bash
RWMUTEX="config-lock"

# Reader process
read_config() {
  koncli rwmutex rlock $RWMUTEX --holder "reader-$HOSTNAME"
  cat /shared/config.json
  koncli rwmutex unlock $RWMUTEX --holder "reader-$HOSTNAME"
}

# Writer process
update_config() {
  koncli rwmutex lock $RWMUTEX --holder "writer-$HOSTNAME" --timeout 1m
  echo '{"updated": true}' > /shared/config.json
  koncli rwmutex unlock $RWMUTEX --holder "writer-$HOSTNAME"
}
```

## Related Commands

- [Mutex CLI](./mutex.md) - Simple mutex commands
- [Lease CLI](./lease.md) - Lease management
- [CLI Overview](./overview.md) - Complete CLI reference
