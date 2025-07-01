
import threading
import time
from typing import List, Callable, Any, Optional


class EventBatcher:
    def log_batch(self, batch, logger=None):
        """
        Log batch analytics using structlog or loguru.
        """
        import structlog
        logger = logger or structlog.get_logger("EventBatcher")
        logger.info("batch_processed", batch_size=len(batch))

    def profile_batch(self, batch):
        """
        Profile batch processing using viztracer (if available).
        """
        try:
            from viztracer import VizTracer
            with VizTracer(output_file="batch_profile.json"):
                self._default_on_batch(batch)
        except ImportError:
            pass
    """
    Batches events for efficient processing and orchestrator integration.
    Supports time-based, size-based, and custom batch triggers.
    Thread-safe and extensible for real-time/federated event flows.
    """
    def __init__(
        self,
        batch_size: int = 32,
        batch_interval: float = 2.0,
        on_batch: Optional[Callable[[List[Any]], None]] = None,
        db=None,
    ):
        if db is None:
            import db as db_module
            self.db = db_module.AsyncEnrichmentDB()
        else:
            self.db = db
        self._batch_size = batch_size
        self._batch_interval = batch_interval
        self._on_batch = on_batch or self._default_on_batch
        self._lock = threading.RLock()
        self._events: List[Any] = []
        self._last_flush = time.time()
        self._stop_event = threading.Event()
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()

    async def _default_on_batch(self, batch: List[Any]):
        """
        Production-grade batch handler:
        - Converts batch to pandas DataFrame
        - Runs unsupervised anomaly detection (IsolationForest)
        - Persists AI-relevant batch metadata to DB
        - Prints summary stats and anomaly indices
        """
        import pandas as pd
        import numpy as np
        from sklearn.ensemble import IsolationForest
        if not batch:
            return
        try:
            df = pd.DataFrame(batch)
            print("[EventBatcher] Batch DataFrame summary:")
            print(df.describe(include='all'))
            # Select only numeric columns for anomaly detection
            num_df = df.select_dtypes(include=[np.number])
            if not num_df.empty and len(num_df) > 4:
                model = IsolationForest(n_estimators=50, contamination=0.1, random_state=42)
                preds = model.fit_predict(num_df)
                anomalies = np.where(preds == -1)[0]
                if len(anomalies) > 0:
                    print(f"[EventBatcher] Anomaly indices: {anomalies.tolist()}")
                    print(df.iloc[anomalies])
                else:
                    print("[EventBatcher] No anomalies detected in this batch.")
            else:
                print("[EventBatcher] Not enough numeric data for anomaly detection.")
            # Persist AI-relevant batch metadata to DB
            ai_metadatas = []
            for row in df.to_dict(orient='records'):
                ai_fields = {k: v for k, v in row.items() if k in [
                    'ai_confidence', 'embedding_id', 'categories', 'last_accessed', 'nexus_channel', 'source_uri', 'scheduler']}
                if ai_fields:
                    ai_metadatas.append({
                        'entity_type': row.get('entity_type', 'batch'),
                        'category': row.get('category', 'batch'),
                        'environment': row.get('environment', 'default'),
                        'role': row.get('role', 'default'),
                        'metadata': ai_fields
                    })
            if ai_metadatas:
                await self.db.batch_insert_metadata(ai_metadatas)
        except Exception as ex:
            print(f"[EventBatcher] Failed to process batch with pandas/sklearn/db: {ex}")

    def add_event(self, event: Any):
        with self._lock:
            self._events.append(event)
            if len(self._events) >= self._batch_size:
                self._flush()

    def _flush(self):
        if not self._events:
            return
        batch = self._events[:]
        self._events.clear()
        if self._on_batch:
            self._on_batch(batch)

    def _run(self):
        while not self._stop_event.is_set():
            time.sleep(self._batch_interval)
            with self._lock:
                now = time.time()
                if self._events and (now - self._last_flush >= self._batch_interval):
                    self._flush()
                    self._last_flush = now

    def stop(self):
        self._stop_event.set()
        self._thread.join()
        with self._lock:
            self._flush()

    def set_on_batch(self, callback: Callable[[List[Any]], None]):
        with self._lock:
            self._on_batch = callback
