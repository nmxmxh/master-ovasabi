#!/bin/sh
set -e
cd "$(dirname "$0")/../wasm"
GOOS=js GOARCH=wasm go build -o ../frontend/public/main.wasm
# Copy wasm_exec.js to frontend/public/
cp $(go env GOROOT)/misc/wasm/wasm_exec.js ../frontend/public/ 