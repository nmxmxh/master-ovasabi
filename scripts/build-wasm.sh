#!/bin/sh
set -e

echo "--- Building WASM modules ---"

# Find wasm_exec.js in the Go root, supporting both old and new locations.
GOROOT=$(go env GOROOT)
# Location for Go 1.21+ (required in 1.23+)
WASM_EXEC_PATH_NEW="$GOROOT/lib/wasm/wasm_exec.js"
# Location for Go < 1.21 (deprecated in 1.21, removed in 1.23)
WASM_EXEC_PATH_OLD="$GOROOT/misc/wasm/wasm_exec.js"

if [ -f "$WASM_EXEC_PATH_NEW" ]; then
    WASM_EXEC_PATH="$WASM_EXEC_PATH_NEW"
elif [ -f "$WASM_EXEC_PATH_OLD" ]; then
    WASM_EXEC_PATH="$WASM_EXEC_PATH_OLD"
else
    echo "Error: wasm_exec.js not found in GOROOT ($GOROOT)." >&2
    echo "Looked in:" >&2
    echo "  - $WASM_EXEC_PATH_NEW (for Go 1.21+)" >&2
    echo "  - $WASM_EXEC_PATH_OLD (for older Go versions)" >&2
    echo "Please ensure your Go installation is correct and complete." >&2
    exit 1
fi

# Navigate to wasm directory to build
cd "$(dirname "$0")/../wasm"

# Build the main WASM binary (single-threaded fallback)
echo "Building main.wasm (single-threaded)..."
GOOS=js GOARCH=wasm go build -ldflags="-X main.enableThreading=false" -o ../frontend/public/main.wasm

# Build the threaded WASM binary with proper threading optimizations
echo "Building main.threads.wasm (multi-threaded)..."
GOOS=js GOARCH=wasm go build -ldflags="-X main.enableThreading=true" -tags=threads -o ../frontend/public/main.threads.wasm

# Copy the required JS support file
echo "Copying $WASM_EXEC_PATH to ../frontend/public/"
cp "$WASM_EXEC_PATH" ../frontend/public/

echo "--- WASM build complete ---"