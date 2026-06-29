#!/usr/bin/env bash
# Starts the backend server and frontend dev server concurrently.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$SCRIPT_DIR/.."

cleanup() {
  echo "Shutting down..."
  kill 0
}
trap cleanup EXIT INT TERM

echo "→ Starting USIP backend..."
(cd "$ROOT/backend" && go run ./cmd/server -addr :8080) &

echo "→ Starting frontend dev server..."
(cd "$ROOT/frontend" && npm run dev) &

wait
