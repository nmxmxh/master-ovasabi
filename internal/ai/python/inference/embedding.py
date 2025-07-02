# embedding.py: Production-grade embedding engine for hybrid/vector search

import os
from utils import get_logger
from typing import List
from inference.wasm import WasmEngine
from concurrent.futures import ThreadPoolExecutor
import numpy as np

try:
    from llama_cpp import Llama
    LLAMA_CPP_AVAILABLE = True
except ImportError:
    Llama = None
    LLAMA_CPP_AVAILABLE = False

try:
    from sentence_transformers import SentenceTransformer
    SENTENCE_TRANSFORMERS_AVAILABLE = True
except (ImportError, Exception) as e:
    SentenceTransformer = None
    SENTENCE_TRANSFORMERS_AVAILABLE = False

    # Handle all import errors including NumPy compatibility issues
    error_msg = str(e)
    if "numpy" in error_msg.lower() or "jax" in error_msg.lower() or "_ARRAY_API" in error_msg or "ml_dtypes" in error_msg.lower():
        print(f"Warning: SentenceTransformers dependency issue: {e}")
        print("EmbeddingEngine will run in fallback mode due to NumPy/JAX/ML-dtypes version conflicts")
    elif "huggingface" in error_msg.lower() or "No module named" in error_msg:
        print(f"Warning: SentenceTransformers library not available: {e}")
        print("EmbeddingEngine will run in fallback mode")
    else:
        print(f"Warning: Unexpected error importing sentence_transformers: {e}")
        print("EmbeddingEngine will run in fallback mode")

try:
    from transformers import AutoTokenizer, AutoModel
    import torch
    import tensorflow as tf
    import flax
    TRANSFORMERS_TORCH_AVAILABLE = True
except (ImportError, Exception) as e:
    AutoTokenizer = None
    AutoModel = None
    torch = None
    tf = None
    flax = None
    TRANSFORMERS_TORCH_AVAILABLE = False

    # Handle all import errors including NumPy compatibility issues
    error_msg = str(e)
    if "numpy" in error_msg.lower() or "jax" in error_msg.lower() or "_ARRAY_API" in error_msg or "ml_dtypes" in error_msg.lower():
        print(f"Warning: Transformers/Torch dependency issue: {e}")
        print("Advanced embedding features will run in fallback mode due to NumPy/JAX/ML-dtypes version conflicts")
    elif "huggingface" in error_msg.lower() or "No module named" in error_msg:
        print(f"Warning: Transformers/Torch libraries not available: {e}")
        print("Advanced embedding features will run in fallback mode")
    else:
        print(f"Warning: Unexpected error importing transformers dependencies: {e}")
        print("Advanced embedding features will run in fallback mode")
    tf = None
    flax = None


class EmbeddingEngine:
    """
    Unified embedding engine for text/vector search (sentence-transformers, transformers, WASM/remote fallback).
    Supports batching, async, and WASM/edge/remote inference.
    """
    def __init__(self, model_name: str = "all-MiniLM-L6-v2", logger=None, wasm_path: str = None, wasm_service_url: str = None, use_llama_cpp: bool = True):
        self.logger = logger or get_logger("EmbeddingEngine")
        self.model_name = model_name
        self.model = None
        self.tokenizer = None
        self.wasm_path = wasm_path
        self.wasm_service_url = wasm_service_url
        self.executor = ThreadPoolExecutor(max_workers=2)
        self.engine = None
        self.llama_model = None

        # Check for offline mode
        offline_mode = os.getenv("HF_HUB_OFFLINE", "false").lower() == "true"
        local_files_only = os.getenv("TRANSFORMERS_OFFLINE", "false").lower() == "true"

        # Check for local model directory first
        model_dir = os.path.join(os.path.dirname(__file__), "..", "models")
        local_model_path = os.path.join(model_dir, model_name.replace("/", "_"))
        
        # Check for GGUF models in the models directory
        gguf_models = []
        if os.path.exists(model_dir):
            gguf_models = [f for f in os.listdir(model_dir) if f.endswith('.gguf')]
        
        try:
            # First priority: Try llama.cpp with local GGUF models
            if use_llama_cpp and LLAMA_CPP_AVAILABLE and Llama and gguf_models:
                gguf_path = os.path.join(model_dir, gguf_models[0])  # Use first GGUF model found
                self.logger.info(f"Loading GGUF model for embeddings: {gguf_path}")
                try:
                    self.llama_model = Llama(
                        model_path=gguf_path,
                        embedding=True,  # Enable embedding mode
                        verbose=False,
                        n_ctx=512,  # Context size for embeddings
                        n_threads=os.cpu_count() or 2
                    )
                    self.engine = "llama-cpp"
                    self.logger.info(f"Successfully loaded GGUF model for embeddings: {gguf_models[0]}")
                except Exception as gguf_exc:
                    self.logger.warning(f"Failed to load GGUF model: {gguf_exc}")
                    import traceback
                    self.logger.warning(f"Traceback: {traceback.format_exc()}")
                    self.llama_model = None
            
            # Second priority: Try sentence-transformers
            if not self.engine and SENTENCE_TRANSFORMERS_AVAILABLE and SentenceTransformer:
                # Try local model first
                if os.path.exists(local_model_path):
                    self.logger.info(f"Loading embedding model from local path: {local_model_path}")
                    self.model = SentenceTransformer(local_model_path)
                    self.engine = "sentence-transformers"
                    self.logger.info(f"Loaded local embedding model from {local_model_path}")
                elif offline_mode or local_files_only:
                    self.logger.warning("Offline mode enabled - trying to load cached embedding model")
                    # Try to load from cache
                    cache_dir = os.path.expanduser("~/.cache/torch/sentence_transformers")
                    model_cache_path = os.path.join(cache_dir, model_name.replace("/", "_"))
                    if os.path.exists(model_cache_path):
                        self.model = SentenceTransformer(model_cache_path)
                        self.engine = "sentence-transformers"
                        self.logger.info(f"Loaded cached embedding model from {model_cache_path}")
                    else:
                        self.logger.warning(f"No cached model found at {model_cache_path}")
                        self.logger.warning("Falling back to basic embedding engine")
                        self.engine = None
                else:
                    # Try to download from HuggingFace
                    self.logger.info(f"Downloading embedding model from HuggingFace: {model_name}")
                    self.model = SentenceTransformer(model_name)
                    self.engine = "sentence-transformers"
            
            # Third priority: Try transformers with PyTorch
            elif not self.engine and TRANSFORMERS_TORCH_AVAILABLE and AutoTokenizer and AutoModel and torch:
                # Try transformers with local models first
                if os.path.exists(local_model_path):
                    self.logger.info(f"Loading embedding model from local transformers path: {local_model_path}")
                    try:
                        self.tokenizer = AutoTokenizer.from_pretrained(local_model_path, local_files_only=True)
                        self.model = AutoModel.from_pretrained(local_model_path, local_files_only=True)
                        self.engine = "transformers-pytorch"
                        self.logger.info(f"Loaded local transformers model from {local_model_path}")
                    except Exception as local_exc:
                        self.logger.warning(f"Failed to load local transformers model: {local_exc}")
                        self.engine = None
                elif offline_mode or local_files_only:
                    self.logger.warning("Offline mode enabled but no local transformers model found")
                    self.engine = None
                else:
                    # Try to download from HuggingFace
                    try:
                        self.logger.info(f"Downloading transformers model from HuggingFace: {model_name}")
                        self.tokenizer = AutoTokenizer.from_pretrained(model_name)
                        self.model = AutoModel.from_pretrained(model_name)
                        self.engine = "transformers-pytorch"
                    except Exception as download_exc:
                        self.logger.warning(f"Failed to download transformers model: {download_exc}")
                        self.engine = None
            elif wasm_path or wasm_service_url:
                self.logger.info("Using WASM engine for embeddings")
                self.wasm_engine = WasmEngine(logger=self.logger)
                self.engine = "wasm"
            elif not self.engine:
                self.logger.warning("No embedding model available! Using random embeddings fallback")
                self.engine = "random"
        except Exception as e:
            self.logger.warning(f"Failed to initialize embedding model: {e}")
            import traceback
            self.logger.warning(f"Traceback: {traceback.format_exc()}")
            self.logger.info("Falling back to random embeddings")
            self.engine = "random"

    def embed(self, texts: List[str], batch_size: int = 32) -> np.ndarray:
        """
        Synchronous embedding with batching and WASM/remote fallback.
        """
        if self.engine == "llama-cpp":
            # Use llama.cpp for embeddings
            all_embeds = []
            for text in texts:
                try:
                    # Get embedding from llama.cpp
                    embedding = self.llama_model.embed(text)
                    
                    # Handle different return types
                    if isinstance(embedding, list):
                        if len(embedding) > 0:
                            # Take the mean if multiple embeddings returned
                            if isinstance(embedding[0], (list, np.ndarray)):
                                embedding = np.mean(embedding, axis=0)
                            else:
                                embedding = np.array(embedding[0] if len(embedding) == 1 else embedding)
                        else:
                            # Fallback if empty
                            embedding = np.zeros(3072)  # Default embedding size for phi-4
                    elif isinstance(embedding, np.ndarray):
                        if embedding.ndim > 1:
                            embedding = np.mean(embedding, axis=0)
                    else:
                        embedding = np.array(embedding)
                    
                    all_embeds.append(embedding.flatten())
                except Exception as e:
                    self.logger.warning(f"Failed to embed text with llama.cpp: {e}")
                    # Fallback to random embedding
                    all_embeds.append(np.random.rand(3072))
            
            return np.vstack(all_embeds)
        elif self.engine == "sentence-transformers":
            # Dynamic batching
            all_embeds = []
            for i in range(0, len(texts), batch_size):
                batch = texts[i : i + batch_size]
                embeds = self.model.encode(batch, show_progress_bar=False, convert_to_numpy=True)
                all_embeds.append(embeds)
            return np.vstack(all_embeds)
        elif self.engine == "transformers-pytorch":
            all_embeds = []
            for i in range(0, len(texts), batch_size):
                batch = texts[i : i + batch_size]
                inputs = self.tokenizer(batch, padding=True, truncation=True, return_tensors="pt")
                with torch.no_grad():
                    outputs = self.model(**inputs)
                    embeddings = outputs.last_hidden_state.mean(dim=1).cpu().numpy()
                all_embeds.append(embeddings)
            return np.vstack(all_embeds)
        elif self.engine == "transformers-tensorflow":
            all_embeds = []
            for i in range(0, len(texts), batch_size):
                batch = texts[i : i + batch_size]
                inputs = self.tokenizer(batch, padding=True, truncation=True, return_tensors="tf")
                outputs = self.model(**inputs)
                embeddings = tf.reduce_mean(outputs.last_hidden_state, axis=1).numpy()
                all_embeds.append(embeddings)
            return np.vstack(all_embeds)
        elif self.engine == "transformers-flax":
            all_embeds = []
            for i in range(0, len(texts), batch_size):
                batch = texts[i : i + batch_size]
                inputs = self.tokenizer(batch, padding=True, truncation=True, return_tensors="np")
                outputs = self.model(**inputs)
                # Flax returns a dict, get last_hidden_state
                embeddings = np.mean(outputs["last_hidden_state"], axis=1)
                all_embeds.append(embeddings)
            return np.vstack(all_embeds)
        elif self.engine == "wasm" and getattr(self, "wasm_engine", None):
            # WASM/remote fallback (sync, one by one)
            all_embeds = []
            for text in texts:
                result = self.wasm_engine.infer(text, wasm_path=self.wasm_path, wasm_service_url=self.wasm_service_url)
                # Expect result to be a list/array
                all_embeds.append(np.array(result, dtype=np.float32))
            return np.vstack(all_embeds)
        elif self.engine == "random":
            # Random embeddings fallback
            self.logger.debug(f"Using random embeddings for {len(texts)} texts")
            return np.random.rand(len(texts), 384)
        else:
            # Final fallback
            self.logger.warning("No embedding engine available, using random embeddings!")
            return np.random.rand(len(texts), 384)

    async def aembed(self, texts: List[str], batch_size: int = 32) -> np.ndarray:
        """
        Async embedding (runs sync embed in thread pool for now).
        """
        import asyncio
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(self.executor, self.embed, texts, batch_size)
