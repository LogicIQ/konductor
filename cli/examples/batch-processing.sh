#!/bin/bash
# Batch Processing Example
# Process items with concurrency control

set -e

SEMAPHORE_NAME="batch-processor"
MAX_CONCURRENT=5
HOLDER="$HOSTNAME-$$"
ERROR_FILE="/tmp/batch-errors-$$"

# Create semaphore for concurrency control
if ! output=$(koncli semaphore create "$SEMAPHORE_NAME" --permits "$MAX_CONCURRENT" 2>&1); then
    if [[ ! "$output" =~ "already exists" ]]; then
        echo "✗ Failed to create semaphore: $output" >&2
        exit 1
    fi
fi

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
                echo "  ✗ Failed to release permit for $item" >&2
                echo "$item" >> "$ERROR_FILE"
                exit 1
            fi
        else
            echo "  ✗ Failed to acquire permit for $item" >&2
            if ! echo "$item" >> "$ERROR_FILE"; then
                echo "  ✗ Failed to write error for $item" >&2
            fi
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
    echo "✗ Processing failed for items: $(tr '\n' ' ' < "$ERROR_FILE")" >&2
    rm -f "$ERROR_FILE"
    exit 1
fi

echo "✓ All items processed"
