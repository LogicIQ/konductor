#!/bin/bash
# Gate Dependency Example
# Wait for multiple dependencies before starting

set -e

# Check if koncli is available
if ! command -v koncli &> /dev/null; then
    echo "Error: koncli command not found. Please install konductor CLI." >&2
    exit 1
fi

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
if ! koncli gate open "${SERVICE_NAME}-ready"; then
    echo "Error: Failed to signal service ready status. Other services may be waiting." >&2
    exit 1
fi

echo "✓ Service $SERVICE_NAME is running"
