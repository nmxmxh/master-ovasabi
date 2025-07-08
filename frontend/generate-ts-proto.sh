#!/bin/bash
# generate-ts-proto.sh
# Script to generate TypeScript code from protobufs using ts-proto

set -e

# Always resolve project root relative to this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
# Use project-root-relative path for proto import (for local and docker compatibility)
PROTO_DIR="../api/protos"
OUT_DIR="./protos"

if [ ! -d "$PROTO_DIR" ]; then
  echo "Error: Could not find api/protos at $PROTO_DIR"
  exit 1
fi

# Find all .proto files in all v* subdirs of each service (not just latest)
echo "Locating all .proto files in v* subdirs..."
PROTO_FILES=$(find "$PROTO_DIR" -type d -name 'v*' | xargs -I {} find {} -name '*.proto')

if [ -z "$PROTO_FILES" ]; then
  echo "No .proto files found in $PROTO_DIR."
  exit 1
fi

echo "Generating TypeScript code using ts-proto..."
npx protoc \
  --plugin=protoc-gen-ts_proto=../node_modules/.bin/protoc-gen-ts_proto \
  -I "$PROTO_DIR" \
  --ts_proto_out="$OUT_DIR" \
  --ts_proto_opt=esModuleInterop=true,forceLong=string,useOptionals=messages \
  $PROTO_FILES

echo "TypeScript proto generation complete. Output in $OUT_DIR"
