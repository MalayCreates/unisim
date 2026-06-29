#!/usr/bin/env bash
# Seeds the running backend with the sample scenario (scripts/sample-scenario.json).
# Usage: ./scripts/seed.sh [backend-base-url]   (default http://localhost:8080)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE="${1:-http://localhost:8080}"

echo "→ Waiting for backend at $BASE ..."
for i in $(seq 1 30); do
  if curl -sf "$BASE/health" >/dev/null 2>&1; then
    break
  fi
  if [ "$i" -eq 30 ]; then
    echo "Error: backend not reachable at $BASE. Start it first (make backend or make dev)." >&2
    exit 1
  fi
  sleep 0.5
done

echo "→ Creating sample scenario ..."
resp=$(curl -sf -X POST "$BASE/api/v1/scenarios" \
  -H 'Content-Type: application/json' \
  --data @"$SCRIPT_DIR/sample-scenario.json")

echo "$resp" | python3 -c '
import sys, json
s = json.load(sys.stdin)
print("  created: " + s["name"])
print("  id:      " + s["id"])
print("  units:   " + str(len(s.get("entities", []))))
print("")
print("Open the frontend, pick it from the scenario dropdown, and click Run.")
'
