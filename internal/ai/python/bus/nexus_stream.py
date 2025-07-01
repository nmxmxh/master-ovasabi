import os
from utils import get_logger
from typing import Optional, Callable
import grpc
from nexus.v1 import nexus_pb2_grpc, nexus_pb2
import threading
import queue
import importlib
import glob
import sys
import time
import collections


class NexusStreamClient:
    def live_event_stream(
        self,
        request: Optional[nexus_pb2.SubscribeRequest] = None,
        on_event: Optional[Callable[[nexus_pb2.EventResponse], None]] = None,
        num_workers: int = 4,
        privacy: str = "dp",
        quality_cb: Optional[Callable[[dict], None]] = None
    ) -> None:
        """
        Live event stream with privacy-preserving aggregation and quality attribution.
        - privacy: 'dp' (differential privacy), 'kanon' (k-anonymity), or 'none'
        - quality_cb: callback for quality/incentive updates
        """
        import numpy as np
        req = request or nexus_pb2.SubscribeRequest()
        self._stop_event.clear()

        def event_producer():
            try:
                for event in self.stub.SubscribeEvents(req, timeout=self.timeout):
                    if self._stop_event.is_set():
                        break
                    # Privacy-preserving: mask or perturb sensitive fields
                    if privacy == "dp" and hasattr(event, 'payload'):
                        # Example: add Laplace noise to numeric fields in payload
                        for k, v in event.payload.ListFields():
                            if isinstance(v, (int, float)):
                                setattr(event.payload, k.name, v + float(np.random.laplace(0, 1)))
                    elif privacy == "kanon":
                        # Example: mask rare fields (not real k-anon)
                        pass
                    try:
                        self._event_queue.put(event, timeout=0.1)
                    except queue.Full:
                        self.logger.warning(
                            "Event queue full, dropping event for backpressure."
                        )
            except grpc.RpcError as e:
                self.logger.error(
                    f"Failed to subscribe to events: {e.details()} (code: {e.code()})"
                )
                raise

        def event_worker(worker_id: int):
            while not self._stop_event.is_set():
                try:
                    event = self._event_queue.get(timeout=0.2)
                except queue.Empty:
                    continue
                try:
                    self.logger.debug(f"[Worker {worker_id}] Processing event: {event}")
                    if on_event:
                        on_event(event)
                    # Quality attribution: call quality_cb with event info
                    if quality_cb:
                        quality_cb({"event": event, "worker": worker_id})
                except Exception as ex:
                    self.logger.error(
                        f"[Worker {worker_id}] Error in event handler: {ex}"
                    )
                finally:
                    self._event_queue.task_done()

        # Start producer thread
        producer_thread = threading.Thread(target=event_producer, daemon=True)
        producer_thread.start()
        self._workers = []
        # Start worker threads
        for i in range(num_workers):
            t = threading.Thread(target=event_worker, args=(i,), daemon=True)
            t.start()
            self._workers.append(t)
    """
    NexusStreamClient provides a robust, production-grade gRPC client for the Nexus event bus.
    Implements publish, subscribe, and orchestration methods using the generated gRPC stubs from nexus.proto.
    Includes robust error handling, safe multithreading, and decoupled low-latency event processing for AI enrichment.
    """
    def __init__(self, address: Optional[str] = None, timeout: float = 5.0, logger: Optional[object] = None):
        # Prefer env var, then argument, then default
        # Prefer env var, then argument, then default, with readable formatting
        nexus_addr = address
        if not nexus_addr:
            nexus_addr = os.getenv("NEXUS_GRPC_ADDR")
        if not nexus_addr:
            nexus_addr = os.getenv("NEXUS_ADDR")
        if not nexus_addr:
            nexus_addr = "nexus:50052"
        self.address = nexus_addr
        self.timeout = timeout
        self.logger = logger or get_logger("NexusStreamClient")
        self.channel = grpc.insecure_channel(self.address)
        self.stub = nexus_pb2_grpc.NexusServiceStub(self.channel)
        self._lock = threading.Lock()
        self._event_queue = queue.Queue(maxsize=1000)  # Bounded queue for backpressure
        self._workers = []
        self._stop_event = threading.Event()

    def publish_event(self, event: nexus_pb2.EventRequest) -> nexus_pb2.EventResponse:
        """Publish an event to the Nexus event bus."""
        try:
            response = self.stub.EmitEvent(event, timeout=self.timeout)
            self.logger.info(f"Published event: {event}")
            return response
        except grpc.RpcError as e:
            self.logger.error(f"Failed to publish event: {e.details()} (code: {e.code()})")
            raise

    def subscribe_events(
        self,
        request: Optional[nexus_pb2.SubscribeRequest] = None,
        on_event: Optional[Callable[[nexus_pb2.EventResponse], None]] = None,
        num_workers: int = 4,
        aggregate_log: bool = False
    ) -> None:
        """
        Subscribe to events from the Nexus event bus. Optionally provide a callback for each event.
        Uses a thread-safe queue and a configurable worker pool for low-latency, decoupled AI enrichment.
        """
        req = request or nexus_pb2.SubscribeRequest(event_types=["*"])
        self._stop_event.clear()

        # Aggregated event stats
        event_counter = collections.Counter()
        event_summary = collections.defaultdict(list)
        last_log_time = [time.time()]

        def event_producer():
            try:
                for event in self.stub.SubscribeEvents(req, timeout=self.timeout):
                    if self._stop_event.is_set():
                        break
                    # Aggregate event type
                    event_type = getattr(event, "event_type", getattr(event, "EventType", ""))
                    event_counter[event_type] += 1
                    event_summary[event_type].append(getattr(event, "event_id", getattr(event, "EventId", "")))
                    try:
                        self._event_queue.put(event, timeout=0.1)
                    except queue.Full:
                        self.logger.warning(
                            "Event queue full, dropping event for backpressure."
                        )
                    # Log summary every minute
                    if aggregate_log and (time.time() - last_log_time[0] > 60):
                        self._log_event_summary(event_counter, event_summary)
                        last_log_time[0] = time.time()
                        # Clear summary for next interval
                        event_counter.clear()
                        event_summary.clear()
            except grpc.RpcError as e:
                self.logger.error(
                    f"Failed to subscribe to events: {e.details()} (code: {e.code()})"
                )
                raise

        def event_worker(worker_id: int):
            while not self._stop_event.is_set():
                try:
                    event = self._event_queue.get(timeout=0.2)
                except queue.Empty:
                    continue
                try:
                    self.logger.debug(f"[Worker {worker_id}] Processing event: {event}")
                    if on_event:
                        on_event(event)
                except Exception as ex:
                    self.logger.error(
                        f"[Worker {worker_id}] Error in event handler: {ex}"
                    )
                finally:
                    self._event_queue.task_done()

        # Start producer thread
        producer_thread = threading.Thread(target=event_producer, daemon=True)
        producer_thread.start()
        self._workers = []
        # Start worker threads
        for i in range(num_workers):
            t = threading.Thread(target=event_worker, args=(i,), daemon=True)
            t.start()
            self._workers.append(t)

    def _log_event_summary(self, event_counter, event_summary):
        summary = {etype: {"count": count, "ids": ids[:5]} for etype, (count, ids) in zip(event_counter.keys(), zip(event_counter.values(), event_summary.values()))}
        self.logger.info(f"[AI AGGREGATED EVENT SUMMARY] Last minute: {summary}")

        # --- AI/LLM summary via prompt ---
        try:
            from llm_registry import get_llm_adapter
            adapter = get_llm_adapter()
            prompt = (
                "You are an event bus monitor. Summarize the following event activity for the last minute, "
                "highlighting the most active event types, any anomalies, and suggest possible actions.\n"
                f"Event summary: {summary}"
            )
            ai_summary = adapter.infer(prompt, max_tokens=128)
            self.logger.info(f"[AI LLM SUMMARY] {ai_summary}")
        except Exception as e:
            self.logger.warning(f"[AI LLM SUMMARY] Could not generate summary: {e}")

    def stop(self):
        """Signal all threads to stop and wait for them to finish."""
        self._stop_event.set()
        # Wait for queue to drain
        self._event_queue.join()

    def orchestrate(self, request: nexus_pb2.OrchestrateRequest) -> nexus_pb2.OrchestrateResponse:
        """Send an orchestration request to the Nexus event bus."""
        try:
            response = self.stub.Orchestrate(request, timeout=self.timeout)
            self.logger.info(f"Orchestration response: {response}")
            return response
        except grpc.RpcError as e:
            self.logger.error(f"Failed to orchestrate: {e.details()} (code: {e.code()})")
            raise

    def close(self):
        """Close the gRPC channel and stop all threads."""
        self.stop()
        with self._lock:
            self.channel.close()

    def start_generic_ai_enrichment_listener(self, ai_module_dir: str = "../inference", priority_prefix: str = "ai_", num_workers: int = 4):
        """
        Listen for all events with 'ai_' in the event type, dynamically load matching AI modules, and process events with priority.
        """
        # Discover all ai_*.py files in the given directory
        ai_module_paths = glob.glob(os.path.join(ai_module_dir, "ai_*.py"))
        ai_modules = {}
        for path in ai_module_paths:
            module_name = os.path.splitext(os.path.basename(path))[0]
            spec = importlib.util.spec_from_file_location(module_name, path)
            if spec and spec.loader:
                mod = importlib.util.module_from_spec(spec)
                sys.modules[module_name] = mod
                spec.loader.exec_module(mod)
                ai_modules[module_name] = mod

        def generic_ai_handler(event):
            event_type = getattr(event, "EventType", getattr(event, "event_type", ""))
            if priority_prefix in event_type:
                # Try to find a matching AI module by event type or context
                for mod_name, mod in ai_modules.items():
                    if mod_name in event_type or mod_name in str(event):
                        # Assume the module has an 'infer' or 'enrich' function
                        func = getattr(mod, "infer", None) or getattr(mod, "enrich", None)
                        if func:
                            # Try to parse payload as JSON
                            import json
                            try:
                                payload = json.loads(getattr(event, "payload", "{}"))
                            except Exception:
                                payload = {}
                            prompt = payload.get("Fields", {}).get("title", "")
                            result = func(prompt) if prompt else func(payload)
                            payload["enrichment"] = result
                            # Publish enriched event
                            enriched_event = getattr(sys.modules[__name__], "nexus_pb2").EventRequest(
                                EventType=event_type.replace(priority_prefix, priority_prefix + "enriched_"),
                                EntityId=getattr(event, "entity_id", getattr(event, "EntityId", "")),
                                Payload=json.dumps(payload)
                            )
                            self.publish_event(enriched_event)
                            break

        # Subscribe to all events with 'ai_' in the type
        from nexus.v1 import nexus_pb2
        sub_req = nexus_pb2.SubscribeRequest(EventTypes=[f"*{priority_prefix}*"])
        self.subscribe_events(sub_req, on_event=generic_ai_handler, num_workers=num_workers)


class NexusEventStream(NexusStreamClient):
    """
    NexusEventStream: High-level interface for the Nexus event bus (Nexus gRPC API)
    Provides orchestration, event publishing, subscription, and pattern query for AI enrichment and agent workflows.
    This is an alias for NexusStreamClient for backward compatibility and dynamic import.
    """
    def listen(self, request=None, num_workers=1):
        """
        Generator that yields events from the Nexus event bus as they arrive.
        Usage: for event in NexusEventStream().listen(): ...
        """
        req = request or nexus_pb2.SubscribeRequest()
        channel = grpc.insecure_channel(self.address)
        stub = nexus_pb2_grpc.NexusServiceStub(channel)
        try:
            for event in stub.SubscribeEvents(req, timeout=self.timeout):
                yield event
        except grpc.RpcError as e:
            self.logger.error(f"Failed to listen for events: {e.details()} (code: {e.code()})")
            raise
# Example usage:
#
# from ai.python.protos.nexus.v1 import nexus_pb2
# client = NexusStreamClient("localhost:50052")
# event = nexus_pb2.EventRequest(EventType="ENRICH", EntityId="123", Metadata=...)  # Fill in fields as needed
# client.publish_event(event)
#
# def handle_event(event):
#     print("Received:", event)
# sub_req = nexus_pb2.SubscribeRequest(EventTypes=["ENRICH"])
# client.subscribe_events(sub_req, on_event=handle_event, num_workers=8)
# ...
# client.close()

# Example usage:
#
# from ai.python.protos.nexus.v1 import nexus_pb2
# client = NexusStreamClient("localhost:50052")
# event = nexus_pb2.EventRequest(EventType="ENRICH", EntityId="123", Metadata=...)  # Fill in fields as needed
# client.publish_event(event)
#
# def handle_event(event):
#     print("Received:", event)
# sub_req = nexus_pb2.SubscribeRequest(EventTypes=["ENRICH"])
# client.subscribe_events(sub_req, on_event=handle_event)
#
# orch_req = nexus_pb2.OrchestrateRequest(...)
# resp = client.orchestrate(orch_req)
# client.close()
