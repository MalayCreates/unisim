#!/usr/bin/env bash
# Generates Python + gRPC stubs from proto/ into adapters/_proto/, shared by all
# Python adapters (Panopticon today; CMO/ArmA later).
#
# Uses grpcio-tools (bundles its own protoc and the google well-known types), so
# neither protoc nor buf's remote plugins are required.
#
# Requires:
#   python3 -m pip install grpcio grpcio-tools
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$SCRIPT_DIR/.."
PROTO_DIR="$ROOT/proto"
OUT="$ROOT/adapters/_proto"

if ! python3 -c 'import grpc_tools.protoc' 2>/dev/null; then
  echo "Error: grpc_tools not found. Install with:" >&2
  echo "  python3 -m pip install grpcio grpcio-tools" >&2
  exit 1
fi

echo "→ Generating Python stubs into adapters/_proto/ ..."
rm -rf "$OUT"
mkdir -p "$OUT"

python3 -m grpc_tools.protoc \
  -I "$PROTO_DIR" \
  --python_out="$OUT" \
  --grpc_python_out="$OUT" \
  entity.proto mission.proto results.proto scenario.proto adapter.proto

# Make the directory importable as a package and expose it on sys.path via a
# tiny loader the adapters import first.
touch "$OUT/__init__.py"

echo "Done. Generated:"
ls -1 "$OUT" | sed 's/^/  /'
