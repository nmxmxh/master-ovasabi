#!/bin/bash
# generate-protos.sh
# This script orchestrates the generation of all protobuf files.

set -e

# Resolve project root relative to this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "Generating Go protobufs..."
"$PROJECT_ROOT/scripts/build-dynamic-tools.sh" # Assumes this script handles Go proto generation

echo "Generating TypeScript protobufs..."
"$PROJECT_ROOT/frontend/generate-ts-proto.sh"

echo "Generating Python protobufs..."
"$PROJECT_ROOT/scripts/generate-python-protos.sh"

echo "All protobuf generation complete."
