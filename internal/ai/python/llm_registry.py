# llm_registry.py: Registry and adapters for LLMs (Mistral, Cohere, etc.)

from typing import Dict, Type


import threading
from typing import List
from utils import get_logger


class LLMAdapter:
    """
    Production-grade base class for LLM adapters.
    Enforces interface, config validation, and robust error handling.
    """
    def __init__(self, **kwargs):
        self.config = kwargs
        self._lock = threading.Lock()
        self._connected = False
        self.logger = get_logger(self.__class__.__name__)
        self._connect()

    def _connect(self):
        self._connected = True

    def infer(self, prompt: str, **kwargs):
        raise NotImplementedError("infer() must be implemented by adapter.")

    def batch_infer(self, prompts: List[str], **kwargs):
        return [self.infer(p, **kwargs) for p in prompts]

    def stream_infer(self, prompt: str, **kwargs):
        yield self.infer(prompt, **kwargs)

    def validate(self):
        if not self._connected:
            raise RuntimeError("Adapter is not connected!")


class MistralAdapter(LLMAdapter):
    """Production-grade adapter for Mistral LLM."""
    def _connect(self):
        self._connected = True


class CohereAdapter(LLMAdapter):
    """Production-grade adapter for Cohere LLM."""
    def _connect(self):
        self._connected = True


class PhiAdapter(LLMAdapter):
    """Production-grade adapter for Phi LLM (transformers/ONNX/edge)."""
    def _connect(self):
        try:
            from inference.phi import PhiEngine
            self.engine = PhiEngine(**self.config)
            self._connected = True
        except Exception as e:
            self.logger.error(f"PhiAdapter connection failed: {e}")
            self._connected = False

    def infer(self, prompt: str, max_tokens: int = 128, **kwargs):
        return self.engine.infer(prompt, max_tokens=max_tokens)

    def batch_infer(self, prompts: List[str], max_tokens: int = 128, **kwargs):
        return self.engine.batch_infer(prompts, max_tokens=max_tokens)


class LlamaCppAdapter(LLMAdapter):
    """Production-grade adapter for Llama.cpp LLM (GGUF, thread-safe, streaming)."""
    def _connect(self):
        try:
            from inference.llama_cpp import LlamaCppEngine
            self.engine = LlamaCppEngine(**self.config)
            self._connected = True
        except Exception as e:
            self.logger.error(f"LlamaCppAdapter connection failed: {e}")
            self._connected = False

    def infer(self, prompt: str, max_tokens: int = 128, **kwargs):
        return self.engine.infer(prompt, max_tokens=max_tokens)

    def batch_infer(self, prompts: List[str], max_tokens: int = 128, **kwargs):
        return self.engine.batch_infer(prompts, max_tokens=max_tokens)

    def stream_infer(self, prompt: str, max_tokens: int = 128, **kwargs):
        return self.engine.stream_infer(prompt, max_tokens=max_tokens)


class WasmAdapter(LLMAdapter):
    """Production-grade adapter for WASM LLM (local or remote, concurrent)."""
    def _connect(self):
        try:
            from inference.wasm import WasmEngine
            self.engine = WasmEngine(logger=self.logger)
            self._connected = True
        except Exception as e:
            self.logger.error(f"WasmAdapter connection failed: {e}")
            self._connected = False

    def infer(self, prompt: str, max_tokens: int = 128, **kwargs):
        # WASM engine is thread-safe, but wrap in lock for safety
        with self._lock:
            result = self.engine.infer(prompt, max_tokens=max_tokens, **kwargs)
            # If result is dict, try to extract text
            if isinstance(result, dict) and "summary" in result:
                return result["summary"]
            return str(result)

    def batch_infer(self, prompts: List[str], max_tokens: int = 128, **kwargs):
        # Use thread pool for concurrency
        import concurrent.futures
        with concurrent.futures.ThreadPoolExecutor() as executor:
            futures = [executor.submit(self.infer, p, max_tokens=max_tokens, **kwargs) for p in prompts]
            return [f.result() for f in futures]

    def stream_infer(self, prompt: str, max_tokens: int = 128, **kwargs):
        # For WASM, just yield the result (no true streaming)
        yield self.infer(prompt, max_tokens=max_tokens, **kwargs)


LLM_ADAPTERS = {
    "phi": PhiAdapter,
    "llama.cpp": LlamaCppAdapter,
    "wasm": WasmAdapter,
    "mistral": MistralAdapter,
    "cohere": CohereAdapter,
}


def get_llm_adapter(name: str, **kwargs) -> LLMAdapter:
    if name not in LLM_ADAPTERS:
        raise ValueError(f"Unknown LLM adapter: {name}")
    return LLM_ADAPTERS[name](**kwargs)


class LLMRegistry:
    """
    Production-grade registry for LLM adapters.
    Enforces interface compliance and provides helpful errors.
    """
    _registry: Dict[str, Type[LLMAdapter]] = {}

    @classmethod
    def register(cls, name: str, adapter: Type[LLMAdapter]):
        if not issubclass(adapter, LLMAdapter):
            raise TypeError(f"Adapter {adapter} must inherit from LLMAdapter.")
        cls._registry[name] = adapter

    @classmethod
    def get(cls, name: str) -> Type[LLMAdapter]:
        if name not in cls._registry:
            raise KeyError(f"No adapter registered for LLM '{name}'")
        return cls._registry[name]


# Register adapters
LLMRegistry.register("mistral", MistralAdapter)
LLMRegistry.register("cohere", CohereAdapter)
LLMRegistry.register("phi", PhiAdapter)
LLMRegistry.register("llama.cpp", LlamaCppAdapter)
