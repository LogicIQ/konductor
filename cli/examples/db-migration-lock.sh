#!/bin/bash
# Database Migration Lock Example
# Prevent concurrent database migrations

set -e

LEASE_NAME="db-migration-lock"
HOLDER="${POD_NAME:-$HOSTNAME-$$}"
TTL="10m"

echo "Attempting to acquire database migration lock..."

# Try to acquire migration lock
if koncli lease acquire "$LEASE_NAME" --holder "$HOLDER" --ttl "$TTL"; then
    echo "✓ Migration lock acquired"
    
    # Cleanup on exit
    trap 'koncli lease release "$LEASE_NAME" --holder "$HOLDER" || echo "Warning: Failed to release lease" >&2' EXIT
    
    echo "Running database migrations..."
    
    # Simulate migrations
    echo "  → Creating tables..."
    sleep 2
    echo "  → Adding indexes..."
    sleep 2
    echo "  → Seeding data..."
    sleep 1
    
    echo "✓ Migrations completed successfully"
else
    echo "✗ Another pod is running migrations"
    echo "  Waiting for migrations to complete..."
    
    # Wait for migrations to finish
    MAX_WAIT=600  # 10 minutes
    ELAPSED=0
    
    while [ $ELAPSED -lt $MAX_WAIT ]; do
        # Try to acquire lease briefly to check if available
        if koncli lease acquire "$LEASE_NAME" --holder "$HOLDER" --ttl 1s 2>/dev/null; then
            if ! koncli lease release "$LEASE_NAME" --holder "$HOLDER"; then
                echo "✗ Failed to release temporary lease" >&2
                exit 1
            fi
            echo "✓ Migrations completed (by another pod)"
            exit 0
        fi
        
        sleep 5
        ELAPSED=$((ELAPSED + 5))
        echo "  Still waiting... (${ELAPSED}s)"
    done
    
    echo "✗ Timeout waiting for migrations"
    exit 1
fi
