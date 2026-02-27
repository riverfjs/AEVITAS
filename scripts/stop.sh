#!/bin/bash
# Stop aevitas gateway gracefully

PID_FILE="${HOME}/.aevitas/aevitas.pid"

if [ ! -f "$PID_FILE" ]; then
    echo "aevitas gateway is not running (no PID file)"
    exit 0
fi

PID=$(cat "$PID_FILE")

if ! ps -p "$PID" > /dev/null 2>&1; then
    echo "aevitas gateway is not running (stale PID file)"
    rm -f "$PID_FILE"
    exit 0
fi

echo "Stopping aevitas gateway (PID: $PID)..."
kill -TERM "$PID"

# Wait for process to exit (max 10 seconds)
for i in {1..10}; do
    if ! ps -p "$PID" > /dev/null 2>&1; then
        echo "aevitas gateway stopped successfully"
        rm -f "$PID_FILE"
        exit 0
    fi
    sleep 1
done

# Force kill if still running
if ps -p "$PID" > /dev/null 2>&1; then
    echo "Force killing aevitas gateway..."
    kill -9 "$PID"
    sleep 1
fi

rm -f "$PID_FILE"
echo "aevitas gateway stopped"

