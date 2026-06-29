#!/usr/bin/env bash
# Generates Go, Python, and TypeScript stubs from proto/ using buf.
# Requires: buf CLI — install via: brew install bufbuild/buf/buf
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$SCRIPT_DIR/.."

check_buf() {
  if ! command -v buf &>/dev/null; then
    echo "Error: buf not found."
    echo "Install with: brew install bufbuild/buf/buf"
    echo "  or: curl -sSL https://github.com/bufbuild/buf/releases/latest/download/buf-Darwin-arm64 -o /usr/local/bin/buf && chmod +x /usr/local/bin/buf"
    exit 1
  fi
}

check_buf

cd "$ROOT"

echo "→ Updating buf dependencies..."
buf dep update proto

echo "→ Generating code..."
buf generate proto

echo "→ Running go mod tidy..."
(cd backend && go mod tidy)

echo "Done. Generated:"
echo "  backend/internal/schema/  (Go)"
echo "  adapters/panopticon/proto/ (Python)"
echo "  frontend/src/proto/        (TypeScript)"
