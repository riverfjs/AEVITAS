#!/bin/bash
# Restart myclaw gateway

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "Restarting myclaw gateway..."

# Stop if running
"$SCRIPT_DIR/stop.sh"

# Wait a moment
sleep 2

# Start again
"$SCRIPT_DIR/start.sh"

