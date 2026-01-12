#!/bin/bash
# Batch Processing Example
# Process items with concurrency control

set -e

SEMAPHORE_NAME="batch-processor"
MAX_CONCURRENT=5
HOLDER="$HOSTNAME-$$"
ERROR_FILE="/tmp/batch-errors-$$"

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
            if ! koncli semaphore release "$SEMAPHORE_NAME" --holder "$HOLDER-$item"; then
                echo "  Warning: Failed to release permit for $item" >&2
            fi
        else
            echo "  ✗ Failed to acquire permit for $item" >&2
            echo "$item" >> "$ERROR_FILE"
            exit 1
        fi
    ) &
done

# Wait for all background jobs and check exit codes
failed_jobs=0
for job in $(jobs -p); do
    if ! wait "$job"; then
        ((failed_jobs++))
    fi
done

if [[ $failed_jobs -gt 0 ]]; then
    echo "✗ $failed_jobs background jobs failed" >&2
    exit 1
fi

# Check for errors
if [[ -f "$ERROR_FILE" ]]; then
    echo "✗ Processing failed for items: $(cat "$ERROR_FILE" | tr '\n' ' ')" >&2
    rm -f "$ERROR_FILE"
    exit 1
fi

echo "✓ All items processed"
