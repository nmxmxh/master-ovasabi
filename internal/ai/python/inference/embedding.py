# embedding.py: Production-grade embedding engine for hybrid/vector search

from utils import get_logger
from typing import List
from inference.wasm import WasmEngine
from concurrent.futures import ThreadPoolExecutor
import numpy as np

try:
    from sentence_transformers import SentenceTransformer
except ImportError:
    SentenceTransformer = None

try:
    from transformers import AutoTokenizer, AutoModel
    import torch
    import tensorflow as tf
    import flax
except ImportError:
    AutoTokenizer = None
    AutoModel = None
    torch = None


class EmbeddingEngine:
    """
    Unified embedding engine for text/vector search (sentence-transformers, transformers, WASM/remote fallback).
    Supports batching, async, and WASM/edge/remote inference.
    """
    def __init__(self, model_name: str = "all-MiniLM-L6-v2", logger=None, wasm_path: str = None, wasm_service_url: str = None):
        self.logger = logger or get_logger("EmbeddingEngine")
        self.model_name = model_name
        self.model = None
        self.tokenizer = None
        self.wasm_path = wasm_path
        self.wasm_service_url = wasm_service_url
        self.executor = ThreadPoolExecutor(max_workers=2)
        if SentenceTransformer:
            self.model = SentenceTransformer(model_name)
            self.engine = "sentence-transformers"
        elif AutoTokenizer and AutoModel and torch:
            self.tokenizer = AutoTokenizer.from_pretrained(model_name)
            try:
                self.model = AutoModel.from_pretrained(model_name)
                self.engine = "transformers-pytorch"
            except Exception as pt_exc:
                try:
                    from transformers import TFAutoModel
                    self.model = TFAutoModel.from_pretrained(model_name)
                    self.engine = "transformers-tensorflow"
                except Exception as tf_exc:
                    try:
                        from transformers import FlaxAutoModel
                        self.model = FlaxAutoModel.from_pretrained(model_name)
                        self.engine = "transformers-flax"
                    except Exception as flax_exc:
                        raise RuntimeError(f"Failed to load model with any backend. PyTorch: {pt_exc}, TensorFlow: {tf_exc}, Flax: {flax_exc}")
        elif AutoTokenizer and torch and tf:
            # TensorFlow only (no AutoModel)
            self.tokenizer = AutoTokenizer.from_pretrained(model_name)
            try:
                from transformers import TFAutoModel
                self.model = TFAutoModel.from_pretrained(model_name)
                self.engine = "transformers-tensorflow"
            except Exception as tf_exc:
                self.logger.warning(f"TensorFlow backend failed: {tf_exc}")
        elif AutoTokenizer and flax:
            # Flax only (no AutoModel)
            self.tokenizer = AutoTokenizer.from_pretrained(model_name)
            try:
                from transformers import FlaxAutoModel
                self.model = FlaxAutoModel.from_pretrained(model_name)
                self.engine = "transformers-flax"
            except Exception as flax_exc:
                self.logger.warning(f"Flax backend failed: {flax_exc}")
        elif wasm_path or wasm_service_url:
            self.wasm_engine = WasmEngine(logger=self.logger)
            self.engine = "wasm"
        else:
            self.engine = None
            self.logger.warning("No embedding model available! Install sentence-transformers, transformers, or provide WASM config.")

    def embed(self, texts: List[str], batch_size: int = 32) -> np.ndarray:
        """
        Synchronous embedding with batching and WASM/remote fallback.
        """
        if self.engine == "sentence-transformers":
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
        elif getattr(self, "wasm_engine", None):
            # WASM/remote fallback (sync, one by one)
            all_embeds = []
            for text in texts:
                result = self.wasm_engine.infer(text, wasm_path=self.wasm_path, wasm_service_url=self.wasm_service_url)
                # Expect result to be a list/array
                all_embeds.append(np.array(result, dtype=np.float32))
            return np.vstack(all_embeds)
        else:
            self.logger.warning("Falling back to random embeddings!")
            return np.random.rand(len(texts), 384)

    async def aembed(self, texts: List[str], batch_size: int = 32) -> np.ndarray:
        """
        Async embedding (runs sync embed in thread pool for now).
        """
        import asyncio
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(self.executor, self.embed, texts, batch_size)
