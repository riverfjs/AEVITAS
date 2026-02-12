#!/bin/bash
# Start myclaw gateway in background with nohup

set -e

MYCLAW_BIN="${HOME}/.myclaw/bin/myclaw"
NOHUP_LOG="${HOME}/.myclaw/workspace/logs/nohup.out"
PID_FILE="${HOME}/.myclaw/myclaw.pid"

# Check if binary exists
if [ ! -f "$MYCLAW_BIN" ]; then
    echo "Error: myclaw binary not found at $MYCLAW_BIN"
    echo "Run 'make prod' to build and install"
    exit 1
fi

# Check if already running
if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
        echo "myclaw gateway is already running (PID: $PID)"
        exit 1
    else
        # Stale PID file, remove it
        rm -f "$PID_FILE"
    fi
fi

# Ensure log directory exists
mkdir -p "$(dirname "$NOHUP_LOG")"

# Start in background with daemon mode
echo "Starting myclaw gateway..."
MYCLAW_DAEMON=1 nohup "$MYCLAW_BIN" gateway >> "$NOHUP_LOG" 2>&1 &
PID=$!

# Save PID
echo "$PID" > "$PID_FILE"

echo "myclaw gateway started (PID: $PID)"
echo "Main logs: ~/.myclaw/workspace/logs/myclaw.log"
echo "Startup logs: $NOHUP_LOG"

