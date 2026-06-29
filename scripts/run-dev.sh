#!/usr/bin/env bash
# Starts the backend, the custom-engine adapter, and the frontend dev server
# concurrently. Ctrl-C tears all three down.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$SCRIPT_DIR/.."

cleanup() {
  echo "Shutting down..."
  kill 0
}
trap cleanup EXIT INT TERM

echo "→ Starting USIP backend (:8080)..."
(cd "$ROOT/backend" && go run ./cmd/server -addr :8080) &

# Give the backend a moment to come up before the adapter tries to register.
sleep 2

echo "→ Starting custom-engine adapter (:50051)..."
(cd "$ROOT/adapters/custom-engine" && go run . -addr :50051 -host localhost -port 50051 -backend http://localhost:8080) &

echo "→ Starting frontend dev server (:5173)..."
(cd "$ROOT/frontend" && npm run dev) &

wait
