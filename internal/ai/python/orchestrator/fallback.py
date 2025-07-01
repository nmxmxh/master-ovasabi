
from utils import get_logger, log_exception
import threading
from typing import Callable, Any, Optional


class FallbackHandler:
    """
    Handles errors, retries, and fallback logic for orchestrator modules.
    Supports custom recovery, exponential backoff, and circuit breaker patterns.
    Thread-safe and production-grade for agentic/federated systems.
    """
    def __init__(self, max_retries: int = 3, backoff_factor: float = 2.0, on_fallback: Optional[Callable[[Exception, Any], None]] = None, logger=None):
        self._max_retries = max_retries
        self._backoff_factor = backoff_factor
        self._on_fallback = on_fallback
        self._lock = threading.RLock()
        self._circuit_open = False
        self._failure_count = 0
        self.logger = logger or get_logger("FallbackHandler")

    def run_with_fallback(self, func: Callable, *args, **kwargs) -> Any:
        retries = 0
        delay = 1.0
        while retries <= self._max_retries:
            try:
                result = func(*args, **kwargs)
                with self._lock:
                    self._failure_count = 0
                    self._circuit_open = False
                return result
            except Exception as e:
                log_exception(self.logger, f"[FallbackHandler] Exception. Retry {retries + 1}/{self._max_retries}", e)
                retries += 1
                with self._lock:
                    self._failure_count += 1
                    if self._failure_count >= self._max_retries:
                        self._circuit_open = True
                if self._on_fallback:
                    self._on_fallback(e, args)
                if retries > self._max_retries:
                    raise
                import time
                time.sleep(delay)
                delay *= self._backoff_factor

    def is_circuit_open(self) -> bool:
        with self._lock:
            return self._circuit_open

    def reset_circuit(self):
        with self._lock:
            self._failure_count = 0
            self._circuit_open = False
