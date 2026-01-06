# WaitGroup CLI Commands

Detailed reference for waitgroup CLI commands.

## Commands

### add

Add to waitgroup counter.

```bash
koncli waitgroup add <name> [flags]
```

**Flags:**
- `--delta int32` - Amount to add (default: 1)

**Examples:**
```bash
# Add 1
koncli waitgroup add workers

# Add 5
koncli waitgroup add workers --delta 5
```

### done

Decrement waitgroup counter by 1.

```bash
koncli waitgroup done <name>
```

**Examples:**
```bash
koncli waitgroup done workers
```

### wait

Wait for counter to reach zero.

```bash
koncli waitgroup wait <name> [flags]
```

**Flags:**
- `--timeout duration` - Wait timeout

**Examples:**
```bash
# Wait indefinitely
koncli waitgroup wait workers

# Wait with timeout
koncli waitgroup wait workers --timeout 5m
```

### create

Create a waitgroup.

```bash
koncli waitgroup create <name> [flags]
```

**Flags:**
- `--ttl duration` - TTL for cleanup

**Examples:**
```bash
koncli waitgroup create workers --ttl 1h
```

### delete

Delete a waitgroup.

```bash
koncli waitgroup delete <name>
```

### list

List all waitgroups.

```bash
koncli waitgroup list
```

## Usage Patterns

### Parallel Job Processing
```bash
#!/bin/bash
WG="batch-jobs"
JOBS=10

# Create and initialize
koncli waitgroup create $WG
koncli waitgroup add $WG --delta $JOBS

# Start jobs
for i in $(seq 1 $JOBS); do
  (
    process-job $i
    koncli waitgroup done $WG
  ) &
done

# Wait for completion
koncli waitgroup wait $WG --timeout 30m
```

### Dynamic Worker Pool
```bash
#!/bin/bash
WG="workers"

koncli waitgroup create $WG

# Add workers dynamically
for item in $(cat items.txt); do
  koncli waitgroup add $WG --delta 1
  process-item $item &
done

# Wait for all
koncli waitgroup wait $WG
```

## Related Commands

- [Barrier CLI](./barrier.md) - Fixed-count coordination
- [CLI Overview](./overview.md) - Complete CLI reference
