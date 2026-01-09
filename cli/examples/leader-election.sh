#!/bin/bash
# Leader Election Example
# Ensure only one instance runs the singleton task

set -e

LEASE_NAME="singleton-task"
HOLDER="$HOSTNAME-$$"
TTL="5m"

echo "Attempting to acquire leadership for $LEASE_NAME..."

# Try to acquire lease
if koncli lease acquire "$LEASE_NAME" --holder "$HOLDER" --ttl "$TTL"; then
    echo "✓ Leadership acquired"
    
    # Cleanup on exit
    trap 'koncli lease release "$LEASE_NAME" --holder "$HOLDER"' EXIT
    
    # Run singleton task
    echo "Running singleton task as leader..."
    
    # Simulate work
    for i in {1..10}; do
        echo "  Working... ($i/10)"
        sleep 1
    done
    
    echo "✓ Task completed"
else
    echo "✗ Another instance is the leader"
    echo "  Exiting gracefully..."
    exit 0
fi
