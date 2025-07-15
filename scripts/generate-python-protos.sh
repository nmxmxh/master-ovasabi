#!/bin/bash
# generate-python-protos.sh
# This script generates Python protobuf files and places them in the correct service directories.

set -e

# Resolve project root relative to this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROTO_DIR="$PROJECT_ROOT/api/protos"
PYTHON_DIR="$PROJECT_ROOT/internal/ai/python"

if [ ! -d "$PROTO_DIR" ]; then
  echo "Error: Could not find api/protos at $PROTO_DIR"
  exit 1
fi

echo "Locating all .proto files..."
PROTO_FILES=$(find "$PROTO_DIR" -type f -name '*.proto')

if [ -z "$PROTO_FILES" ]; then
  echo "No .proto files found in $PROTO_DIR."
  exit 1
fi

echo "Generating Python code using grpc_tools.protoc..."
python3 -m grpc_tools.protoc \
  -I "$PROTO_DIR" \
  --python_out="$PYTHON_DIR" \
  --grpc_python_out="$PYTHON_DIR" \
  $PROTO_FILES

echo "Python proto generation complete. Output in $PYTHON_DIR"
