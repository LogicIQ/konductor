# Once CLI Commands

Detailed reference for once-related CLI commands.

## Commands

### check

Check if once has been executed.

```bash
koncli once check <name> [flags]
```

**Examples:**
```bash
# Check execution status
koncli once check app-init

# Check in specific namespace
koncli once check app-init -n production
```

### create

Create a new once.

```bash
koncli once create <name> [flags]
```

**Flags:**
- `--ttl duration` - Optional TTL for cleanup

**Examples:**
```bash
# Create once
koncli once create app-init

# With TTL
koncli once create app-init --ttl 1h
```

### delete

Delete a once.

```bash
koncli once delete <name>
```

**Examples:**
```bash
koncli once delete app-init
```

### list

List all onces.

```bash
koncli once list [flags]
```

**Examples:**
```bash
# List all
koncli once list

# With namespace
koncli once list -n production
```

## Usage Patterns

### Database Initialization
```bash
#!/bin/bash
ONCE_NAME="db-init"

if koncli once check $ONCE_NAME | grep -q "not been executed"; then
  echo "Running database initialization..."
  run-migrations
  echo "Initialization complete"
else
  echo "Database already initialized"
fi
```

### One-Time Setup
```bash
#!/bin/bash
ONCE_NAME="app-setup"

# Check if setup needed
if ! koncli once check $ONCE_NAME | grep -q "has been executed"; then
  echo "Running one-time setup..."
  setup-application
fi

# Start application
start-app
```

### Resource Provisioning
```bash
#!/bin/bash
ONCE_NAME="provision-resources"

koncli once create $ONCE_NAME --ttl 24h

if ! koncli once check $ONCE_NAME | grep -q "has been executed"; then
  echo "Provisioning cloud resources..."
  provision-s3-bucket
  provision-database
  echo "Provisioning complete"
fi
```

## Related Commands

- [Mutex CLI](./mutex.md) - Mutual exclusion
- [Lease CLI](./lease.md) - Singleton execution
- [CLI Overview](./overview.md) - Complete CLI reference
