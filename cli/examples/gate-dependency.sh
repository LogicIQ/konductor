#!/bin/bash
# Gate Dependency Example
# Wait for multiple dependencies before starting

set -e

GATE_NAME="service-dependencies"
SERVICE_NAME="${1:-my-service}"

echo "Service: $SERVICE_NAME"
echo "Waiting for dependencies via gate: $GATE_NAME"

# Wait for gate to open (all dependencies ready)
if koncli gate wait "$GATE_NAME" --timeout 5m; then
    echo "✓ All dependencies are ready"
else
    echo "✗ Timeout waiting for dependencies"
    exit 1
fi

# Start service
echo "Starting $SERVICE_NAME..."
sleep 2

echo "✓ Service started successfully"

# Signal service is ready
koncli gate open "${SERVICE_NAME}-ready" 2>/dev/null || true

echo "✓ Service $SERVICE_NAME is running"
