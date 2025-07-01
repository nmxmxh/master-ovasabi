#!/bin/sh
set -e
MODEL_DIR="$(dirname "$0")/../internal/ai/python/models"
MODEL_FILE="$MODEL_DIR/phi-4-mini-q4_k_s.gguf"

if [ ! -f "$MODEL_FILE" ]; then
  echo "[download_models.sh] Model not found, downloading..."
  mkdir -p "$MODEL_DIR"
  huggingface-cli download Mungert/Phi-4-mini-instruct.gguf \
    --include "phi-4-mini-q4_k_s.gguf" --local-dir "$MODEL_DIR"
else
  echo "[download_models.sh] Model already present: $MODEL_FILE"
fi
