# AI System Offline Configuration

This document explains how to run the AI system in offline mode when there's no internet connectivity or when Hugging Face is not accessible.

## Quick Start - Offline Mode

To run the AI system in offline mode:

```bash
./scripts/run_offline_ai.sh
```

Or manually set environment variables:

```bash
export HF_HUB_OFFLINE=true
export TRANSFORMERS_OFFLINE=true
cd internal/ai/python
python3 main.py
```

## How It Works

The system now includes graceful fallback mechanisms:

1. **PhiEngine**: 
   - First tries to load from local model directory (`internal/ai/python/models/`)
   - Falls back to WASM/edge mode if no models are available
   - Logs warnings but continues running

2. **EmbeddingEngine**:
   - Tries to load cached models from `~/.cache/torch/sentence_transformers/`
   - Falls back to WASM mode if no cached models found
   - Continues operation with degraded functionality

3. **Other Components**:
   - All other AI components (VectorDB, Devourer, etc.) work without internet

## Environment Variables

- `HF_HUB_OFFLINE=true` - Prevents Hugging Face Hub downloads
- `TRANSFORMERS_OFFLINE=true` - Forces transformers library to use local files only
- `HF_HUB_DISABLE_PROGRESS_BARS=true` - Reduces console noise
- `HF_HUB_DISABLE_SYMLINKS_WARNING=true` - Suppresses warnings

## Pre-downloading Models (Optional)

To prepare for offline operation, you can pre-download models:

```bash
# Download Phi model (if internet is available)
./scripts/download_models.sh

# Download embedding models to cache
python3 -c "from sentence_transformers import SentenceTransformer; SentenceTransformer('all-MiniLM-L6-v2')"
```

## Fallback Behavior

When running offline:

- **PhiEngine**: Returns JSON fallback responses for text generation
- **EmbeddingEngine**: Uses WASM engine or returns dummy embeddings
- **System**: Continues to function with knowledge graph and other services
- **Logging**: Clear warnings about missing models, but no crashes

## Troubleshooting

If you see `"System bootstrap failed"` with network errors:
1. Use the offline script: `./scripts/run_offline_ai.sh`
2. Or set the environment variables manually
3. Check that Python dependencies are installed locally

The system is designed to degrade gracefully rather than fail completely when models are unavailable.
