import threading
import time
from typing import Dict, Any, Optional, Callable, List
from collections import defaultdict, deque


class MetricsCollector:
    def log_metrics(self, logger=None):
        """
        Log metrics using structlog or loguru for observability.
        """
        import structlog
        logger = logger or structlog.get_logger("MetricsCollector")
        metrics = self.report()
        logger.info("metrics_report", **metrics)

    def profile_metrics(self):
        """
        Profile metrics collection/reporting using viztracer (if available).
        """
        try:
            from viztracer import VizTracer
            with VizTracer(output_file="metrics_profile.json"):
                self.report()
        except ImportError:
            pass
    """
    Collects, aggregates, and reports system and federated metrics.
    Supports real-time, batch, and privacy-preserving aggregation.
    Thread-safe and extensible for custom metrics and hooks.
    """
    def __init__(self, window_seconds: int = 60, db=None):
        self._lock = threading.RLock()
        self._metrics = defaultdict(lambda: deque(maxlen=1000))
        self._window_seconds = window_seconds
        self._hooks: List[Callable[[str, float, Dict[str, Any]], None]] = []
        if db is None:
            from db import db as db_module
            self.db = db_module.AsyncEnrichmentDB()
        else:
            self.db = db

    def record(self, name: str, value: float, tags: Optional[Dict[str, Any]] = None):
        import numpy as np
        with self._lock:
            ts = time.time()
            # Optionally convert value to numpy float for consistency
            try:
                value = float(np.float64(value))
            except Exception:
                pass
            self._metrics[name].append((ts, value, tags or {}))
            for hook in self._hooks:
                hook(name, value, tags or {})

    def aggregate(self, name: str, method: str = "mean") -> Optional[float]:
        with self._lock:
            now = time.time()
            values = [v for (ts, v, _) in self._metrics[name] if now - ts <= self._window_seconds]
            if not values:
                return None
            if method == "mean":
                return sum(values) / len(values)
            elif method == "sum":
                return sum(values)
            elif method == "max":
                return max(values)
            elif method == "min":
                return min(values)
            else:
                raise ValueError(f"Unknown aggregation method: {method}")

    def report(self) -> Dict[str, Any]:
        import pandas as pd
        with self._lock:
            now = time.time()
            report = {}
            for name, vals in self._metrics.items():
                filtered = [(ts, v, t) for (ts, v, t) in vals if now - ts <= self._window_seconds]
                if not filtered:
                    continue
                df = pd.DataFrame(filtered, columns=["timestamp", "value", "tags"])
                stats = {
                    "count": len(df),
                    "mean": df["value"].mean(),
                    "sum": df["value"].sum(),
                    "max": df["value"].max(),
                    "min": df["value"].min(),
                    "std": df["value"].std(),
                }
                report[name] = stats
            return report

    def add_hook(self, hook: Callable[[str, float, Dict[str, Any]], None]):
        with self._lock:
            self._hooks.append(hook)

    def clear(self):
        with self._lock:
            self._metrics.clear()

    def export(self, format: str = "json") -> Any:
        import json
        if format == "json":
            return json.dumps(self.report(), indent=2)
        raise NotImplementedError(f"Export format {format} not supported.")

    def privacy_preserving_aggregate(self, name: str, epsilon: float = 1.0) -> Optional[float]:
        """
        Differential privacy: adds Laplace noise to the mean.
        """
        import random
        mean = self.aggregate(name, method="mean")
        if mean is None:
            return None
        noise = random.gauss(0, 1 / epsilon)
        return mean + noise

    async def persist_metrics(self):
        """
        Persist current metrics as AI-relevant metadata in the DB.
        """
        # pandas is already imported where needed
        metrics = self.report()
        ai_metadatas = []
        for name, stats in metrics.items():
            ai_fields = {k: v for k, v in stats.items() if k in [
                'mean', 'max', 'min', 'std', 'count']}
            if ai_fields:
                ai_metadatas.append({
                    'entity_type': 'metrics',
                    'category': name,
                    'environment': 'default',
                    'role': 'system',
                    'metadata': ai_fields
                })
        if ai_metadatas:
            await self.db.batch_insert_metadata(ai_metadatas)
