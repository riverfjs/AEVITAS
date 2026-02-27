#!/bin/bash
# Start aevitas gateway in background with nohup

set -e

AEVITAS_BIN="${HOME}/.aevitas/bin/aevitas"
NOHUP_LOG="${HOME}/.aevitas/workspace/logs/nohup.out"
PID_FILE="${HOME}/.aevitas/aevitas.pid"

# Check if binary exists
if [ ! -f "$AEVITAS_BIN" ]; then
    echo "Error: aevitas binary not found at $AEVITAS_BIN"
    echo "Run 'make prod' to build and install"
    exit 1
fi

# Check if already running
if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        echo "aevitas gateway is already running (PID: $PID)"
        exit 1
    else
        # Stale PID file, remove it
        rm -f "$PID_FILE"
    fi
fi

# Ensure log directory exists
mkdir -p "$(dirname "$NOHUP_LOG")"

# Start in background with daemon mode
echo "Starting aevitas gateway..."
AEVITAS_DAEMON=1 nohup "$AEVITAS_BIN" gateway >> "$NOHUP_LOG" 2>&1 &
PID=$!

# Save PID
echo "$PID" > "$PID_FILE"

echo "aevitas gateway started (PID: $PID)"
echo "Main logs: ~/.aevitas/workspace/logs/aevitas.log"
echo "Startup logs: $NOHUP_LOG"

