#!/bin/bash
# Example: Cache coordination with RWMutex
# Multiple readers can access cache simultaneously, but writes are exclusive

set -e

RWMUTEX_NAME="cache-rwmutex"

echo "=== RWMutex Cache Coordination Example ==="

# Create RWMutex
echo "Creating RWMutex..."
koncli rwmutex create $RWMUTEX_NAME --ttl 5m

# Simulate multiple readers
echo ""
echo "Starting reader processes..."
for i in {1..3}; do
  (
    koncli rwmutex rlock $RWMUTEX_NAME --holder "reader-$i" --timeout 10s
    echo "Reader $i: Reading from cache..."
    sleep 2
    koncli rwmutex unlock $RWMUTEX_NAME --holder "reader-$i"
    echo "Reader $i: Released read lock"
  ) &
done

# Wait for readers
sleep 1

# Simulate writer (will wait for readers to finish)
echo ""
echo "Starting writer process..."
koncli rwmutex lock $RWMUTEX_NAME --holder "writer-1" --timeout 30s
echo "Writer: Updating cache..."
sleep 2
koncli rwmutex unlock $RWMUTEX_NAME --holder "writer-1"
echo "Writer: Released write lock"

# Wait for all background jobs
if ! wait; then
    echo "Warning: Some background jobs failed" >&2
fi

echo ""
if ! echo "=== Example Complete ==="; then
    exit 1
fi
echo "Cleanup: koncli rwmutex delete $RWMUTEX_NAME"
