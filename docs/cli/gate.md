# Gate CLI Commands

Detailed reference for gate-related CLI commands.

## Commands

### wait

Wait for a gate to open (all conditions met).

```bash
koncli gate wait <name> [flags]
```

**Flags:**
- `--timeout duration` - Maximum wait time (default: 30m)

**Examples:**
```bash
# Wait for gate
koncli gate wait deployment-gate

# Wait with custom timeout
koncli gate wait deployment-gate --timeout 1h
```

### status

Check gate status and conditions.

```bash
koncli gate status <name> [flags]
```

**Examples:**
```bash
# Check status
koncli gate status deployment-gate

# JSON output
koncli gate status deployment-gate -o json
```

## Usage Patterns

### Job Dependency
```bash
#!/bin/bash
# Wait for dependencies
koncli gate wait processing-gate --timeout 1h

# Proceed with work
process-data
```

### Deployment Gate
```bash
#!/bin/bash
# Wait for all checks
koncli gate wait prod-deployment-gate --timeout 30m

# Deploy
deploy-application
```

### Multi-Stage Pipeline
```bash
#!/bin/bash
# Wait for stage 1 gate
koncli gate wait stage-1-gate --timeout 30m

# Do stage 2 work
process-stage-2

# Wait for stage 2 gate
koncli gate wait stage-2-gate --timeout 30m

# Do stage 3 work
process-stage-3
```

## Related Commands

- [Barrier CLI](./barrier.md) - Barrier coordination
- [CLI Overview](./overview.md) - Complete CLI reference
