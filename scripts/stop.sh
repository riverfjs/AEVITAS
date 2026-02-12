#!/bin/bash
# Stop myclaw gateway gracefully

PID_FILE="${HOME}/.myclaw/myclaw.pid"

if [ ! -f "$PID_FILE" ]; then
    echo "myclaw gateway is not running (no PID file)"
    exit 0
fi

PID=$(cat "$PID_FILE")

if ! ps -p "$PID" > /dev/null 2>&1; then
    echo "myclaw gateway is not running (stale PID file)"
    rm -f "$PID_FILE"
    exit 0
fi

echo "Stopping myclaw gateway (PID: $PID)..."
kill -TERM "$PID"

# Wait for process to exit (max 10 seconds)
for i in {1..10}; do
    if ! ps -p "$PID" > /dev/null 2>&1; then
        echo "myclaw gateway stopped successfully"
        rm -f "$PID_FILE"
        exit 0
    fi
    sleep 1
done

# Force kill if still running
if ps -p "$PID" > /dev/null 2>&1; then
    echo "Force killing myclaw gateway..."
    kill -9 "$PID"
    sleep 1
fi

rm -f "$PID_FILE"
echo "myclaw gateway stopped"

