#!/bin/bash
# Batch Processing Example
# Process items with concurrency control

set -e

SEMAPHORE_NAME="batch-processor"
MAX_CONCURRENT=5
HOLDER="$HOSTNAME-$$"

# Create semaphore for concurrency control
koncli semaphore create "$SEMAPHORE_NAME" --permits "$MAX_CONCURRENT" 2>/dev/null || true

# Sample items to process
ITEMS=(item1 item2 item3 item4 item5 item6 item7 item8 item9 item10)

echo "Processing ${#ITEMS[@]} items with max $MAX_CONCURRENT concurrent workers"

# Process items in parallel with concurrency control
for item in "${ITEMS[@]}"; do
    (
        # Acquire permit
        if koncli semaphore acquire "$SEMAPHORE_NAME" --holder "$HOLDER-$item"; then
            echo "  → Processing $item..."
            
            # Simulate processing
            sleep $((RANDOM % 3 + 1))
            
            echo "  ✓ Completed $item"
            
            # Release permit
            koncli semaphore release "$SEMAPHORE_NAME" --holder "$HOLDER-$item"
        else
            echo "  ✗ Failed to acquire permit for $item"
        fi
    ) &
done

# Wait for all background jobs
wait

echo "✓ All items processed"
