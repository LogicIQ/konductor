#!/bin/bash
# Barrier Coordination Example
# Synchronize multiple workers at a coordination point

set -e

BARRIER_NAME="worker-sync-point"
WORKER_ID="${1:-worker-$RANDOM}"

echo "Worker: $WORKER_ID"

# Phase 1: Do initial work
echo "Phase 1: Processing data..."
sleep $((RANDOM % 5 + 1))
echo "✓ Phase 1 complete"

# Signal arrival at barrier
echo "Arriving at barrier: $BARRIER_NAME"
koncli barrier arrive "$BARRIER_NAME" --holder "$WORKER_ID"

# Wait for all workers
echo "Waiting for all workers at barrier..."
if koncli barrier wait "$BARRIER_NAME" --timeout 2m; then
    echo "✓ All workers arrived, barrier is open"
else
    echo "✗ Timeout waiting for barrier"
    exit 1
fi

# Phase 2: Continue with synchronized work
echo "Phase 2: Processing synchronized data..."
sleep 2
echo "✓ Phase 2 complete"

echo "✓ Worker $WORKER_ID finished"
