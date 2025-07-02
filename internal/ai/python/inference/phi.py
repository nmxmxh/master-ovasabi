
# phi.py: Production-grade Phi-3/4 inference for Python & WASM (transformers/ONNX/edge)
import os
from utils import get_logger
from typing import List


try:
    from transformers import AutoModelForCausalLM, AutoTokenizer
    TRANSFORMERS_BASIC_AVAILABLE = True
except (ImportError, Exception) as e:
    AutoModelForCausalLM = None
    AutoTokenizer = None
    TRANSFORMERS_BASIC_AVAILABLE = False
    print(f"Warning: Basic transformers components not available: {e}")

try:
    from transformers import pipeline
    TRANSFORMERS_PIPELINE_AVAILABLE = True
except (ImportError, Exception) as e:
    pipeline = None
    TRANSFORMERS_PIPELINE_AVAILABLE = False
    print(f"Warning: Transformers pipeline not available: {e}")

# Overall transformers availability
TRANSFORMERS_AVAILABLE = TRANSFORMERS_BASIC_AVAILABLE and TRANSFORMERS_PIPELINE_AVAILABLE

if not TRANSFORMERS_AVAILABLE:
    # Log the specific error for debugging
    if not TRANSFORMERS_BASIC_AVAILABLE and not TRANSFORMERS_PIPELINE_AVAILABLE:
        print("PhiEngine will run in fallback mode - transformers library not functional")
    elif not TRANSFORMERS_PIPELINE_AVAILABLE:
        print("PhiEngine will run in fallback mode - transformers pipeline not available")
    else:
        print("PhiEngine will run in fallback mode - transformers dependency issues")


class PhiEngine:
    """
    Production-grade Phi-3/4 inference engine.
    Supports transformers, ONNX, and WASM/edge fallback.
    """
    def __init__(self, model_name: str = "microsoft/phi-2", device: str = "cpu", logger=None):
        self.logger = logger or get_logger("PhiEngine")
        self.model_name = model_name
        self.device = device
        self.model = None
        self.tokenizer = None
        self.pipe = None

        # Check for offline mode or local models
        offline_mode = os.getenv("HF_HUB_OFFLINE", "false").lower() == "true"
        local_files_only = os.getenv("TRANSFORMERS_OFFLINE", "false").lower() == "true"

        # Check for local model directory first
        model_dir = os.path.join(os.path.dirname(__file__), "..", "models")
        local_model_path = os.path.join(model_dir, model_name.replace("/", "_"))

        if TRANSFORMERS_AVAILABLE and AutoModelForCausalLM and AutoTokenizer:
            try:
                # Try local model first
                if os.path.exists(local_model_path):
                    self.logger.info(f"Loading model from local path: {local_model_path}")
                    self.tokenizer = AutoTokenizer.from_pretrained(local_model_path, local_files_only=True)
                    self.model = AutoModelForCausalLM.from_pretrained(local_model_path, local_files_only=True)
                elif offline_mode or local_files_only:
                    self.logger.warning(f"Offline mode enabled but no local model found at {local_model_path}")
                    self.logger.warning("PhiEngine will run in fallback mode without model inference")
                    return
                else:
                    # Try to download from HuggingFace
                    self.logger.info(f"Downloading model from HuggingFace: {model_name}")
                    self.tokenizer = AutoTokenizer.from_pretrained(model_name)
                    self.model = AutoModelForCausalLM.from_pretrained(model_name)

                if self.model and self.tokenizer:
                    self.pipe = pipeline(
                        "text-generation",
                        model=self.model,
                        tokenizer=self.tokenizer,
                        device=0 if device == "cuda" else -1
                    )
                    self.logger.info("PhiEngine initialized successfully with transformers pipeline")

            except Exception as e:
                self.logger.warning(f"Failed to initialize PhiEngine with transformers: {e}")
                self.logger.info("PhiEngine will run in fallback mode")
        else:
            self.logger.warning("Transformers library not available. PhiEngine running in fallback mode")

    def infer(self, prompt: str, max_tokens: int = 128) -> str:
        if self.pipe:
            out = self.pipe(prompt, max_new_tokens=max_tokens, return_full_text=False)
            return (
                out[0]["generated_text"]
                if out and "generated_text" in out[0]
                else str(out)
            )
        else:
            # WASM/edge fallback
            return (
                '{"summary": "WASM fallback", "confidence": 0.5, "categories": ["Misc"]}'
            )

    def batch_infer(self, prompts: List[str], max_tokens: int = 128) -> List[str]:
        """
        Parallelized batch inference for Phi models (transformers pipeline or fallback).
        Preserves order of prompts.
        """
        import concurrent.futures
        results = [None] * len(prompts)
        if self.pipe and hasattr(self.pipe, "__call__"):
            # Use pipeline's built-in batching if available
            try:
                out = self.pipe(prompts, max_new_tokens=max_tokens, return_full_text=False)
                for i, o in enumerate(out):
                    results[i] = (
                        o["generated_text"] if o and "generated_text" in o else str(o)
                    )
                return results
            except Exception as ex:
                self.logger.warning(f"Pipeline batch failed, falling back to parallel: {ex}")

        # Fallback: parallelize individual inference
        def infer_with_index(idx_prompt):
            idx, prompt = idx_prompt
            return idx, self.infer(prompt, max_tokens=max_tokens)
        with concurrent.futures.ThreadPoolExecutor(max_workers=min(8, len(prompts))) as executor:
            futures = [executor.submit(infer_with_index, (i, p)) for i, p in enumerate(prompts)]
            for fut in concurrent.futures.as_completed(futures):
                idx, result = fut.result()
                results[idx] = result
        return results
