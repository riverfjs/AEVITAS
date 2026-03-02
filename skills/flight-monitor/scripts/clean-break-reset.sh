#!/usr/bin/env bash
set -euo pipefail

SKILL_DIR="$HOME/.aevitas/workspace/.claude/skills/flight-monitor"
DATA_DIR="$SKILL_DIR/data"
MONITORS_FILE="$DATA_DIR/monitors.json"

mkdir -p "$DATA_DIR"

if [ -f "$MONITORS_FILE" ]; then
  ts="$(date +%Y%m%d-%H%M%S)"
  backup="$DATA_DIR/monitors.backup-$ts.json"
  cp "$MONITORS_FILE" "$backup"
  echo "Backup created: $backup"
else
  echo "No existing monitors file found. Creating a new one."
fi

echo "[]" > "$MONITORS_FILE"
echo "Reset complete: $MONITORS_FILE"
