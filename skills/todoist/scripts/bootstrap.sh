#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$SKILL_DIR"
mkdir -p bin
go build -o bin/todoist scripts/*.go
echo "todoist bootstrap done: $SKILL_DIR/bin/todoist"
