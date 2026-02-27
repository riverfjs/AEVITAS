#!/bin/bash
# Restart aevitas gateway

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Restarting aevitas gateway..."

# Stop if running
"$SCRIPT_DIR/stop.sh"

# Wait a moment
sleep 2

# Start again
"$SCRIPT_DIR/start.sh"

