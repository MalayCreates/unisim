#!/usr/bin/env bash
# Generates Go stubs from proto/ using buf + locally-installed protoc plugins.
#
# Requires:
#   buf               -> https://buf.build/docs/installation
#   protoc-gen-go     -> go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#   protoc-gen-go-grpc-> go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
#
# (buf bundles its own protobuf compiler, so protoc itself is not needed.)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$SCRIPT_DIR/.."

require() {
  if ! command -v "$1" &>/dev/null; then
    echo "Error: '$1' not found on PATH."
    echo "  $2"
    exit 1
  fi
}

require buf            "Install: https://buf.build/docs/installation"
require protoc-gen-go  "Install: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
require protoc-gen-go-grpc "Install: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"

cd "$ROOT"

echo "→ Linting protos..."
buf lint proto

echo "→ Generating Go code..."
buf generate proto

echo "→ Tidying backend module..."
(cd backend && go mod tidy)

echo "Done. Generated Go stubs in backend/schema/"
