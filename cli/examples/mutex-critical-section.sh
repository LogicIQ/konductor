#!/bin/bash
# Example: Using Mutex for Critical Section Protection
# This demonstrates how to use mutex to ensure only one process executes critical code

set -e

MUTEX_NAME="db-migration"
HOLDER="${HOSTNAME:-worker-$$}"

echo "=== Mutex Critical Section Example ==="
echo "Mutex: $MUTEX_NAME"
echo "Holder: $HOLDER"
echo ""

# Create mutex if it doesn't exist
echo "Creating mutex..."
koncli mutex create "$MUTEX_NAME" --ttl 5m || echo "Mutex already exists"

# Try to acquire the lock
echo "Attempting to acquire lock..."
if koncli mutex lock "$MUTEX_NAME" --holder "$HOLDER" --timeout 30s; then
    echo "✓ Lock acquired!"
    trap '
        for i in {1..3}; do
            if koncli mutex unlock "$MUTEX_NAME" --holder "$HOLDER"; then
                exit 0
            fi
            echo "Retry $i: Failed to release lock, retrying..." >&2
            sleep 1
        done
        echo "Error: Failed to release lock after 3 attempts" >&2
        exit 1
    ' EXIT INT TERM
    
    # Critical section - only one process can be here at a time
    echo "Executing critical section..."
    echo "  - Running database migration"
    sleep 3
    echo "  - Migration complete"
    
    echo "✓ Critical section complete"
else
    echo "✗ Failed to acquire lock (another process holds it)"
    exit 1
fi

echo ""
echo "=== Example Complete ==="
