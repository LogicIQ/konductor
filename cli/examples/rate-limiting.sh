#!/bin/bash
# Rate Limiting Example
# Limit concurrent API calls to external service

set -e

SEMAPHORE_NAME="external-api-limit"
PERMITS=10
HOLDER="$HOSTNAME-$$"

# Create semaphore (idempotent)
if ! output=$(koncli semaphore create "$SEMAPHORE_NAME" --permits "$PERMITS" 2>&1); then
    if echo "$output" | grep -q "already exists"; then
        echo "Semaphore already exists, continuing..."
    else
        echo "Error: Failed to create semaphore: $output" >&2
        exit 1
    fi
fi

echo "Acquiring permit from $SEMAPHORE_NAME..."

# Acquire permit with timeout
if koncli semaphore acquire "$SEMAPHORE_NAME" --holder "$HOLDER" --timeout 30s; then
    echo "✓ Permit acquired"
    
    # Cleanup on exit
    trap 'koncli semaphore release "$SEMAPHORE_NAME" --holder "$HOLDER" || echo "Warning: Failed to release semaphore" >&2' EXIT
    
    # Simulate API call
    echo "Calling external API..."
    sleep 2
    
    echo "✓ API call completed"
else
    echo "✗ Failed to acquire permit (timeout or error)"
    exit 1
fi
