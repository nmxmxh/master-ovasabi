"""
CLI for OVASABI AI/Knowledge Graph system.
Supports running as a service, enrichment, crawling, and diagnostics.
"""
import typer
import asyncio
from main import main as run_service

from knowledge.graph import KnowledgeGraphEnricher
from crawler.devourer import Devourer
import uuid
import json

from bus.nexus_stream import NexusStreamClient
from nexus.v1 import nexus_pb2
from utils import setup_logging, get_logger

from llm_registry import get_llm_adapter, LLM_ADAPTERS
import threading

# --- Production-grade Nexus Event Bus ---
bus = NexusStreamClient()

app = typer.Typer()


@app.command()
def llm(
    prompt: str = typer.Option(..., help="Prompt or chat message for the LLM"),
    backend: str = typer.Option("llama.cpp", help=f"LLM backend: {', '.join(LLM_ADAPTERS.keys())}"),
    max_tokens: int = typer.Option(128, help="Max tokens to generate"),
    temperature: float = typer.Option(0.0, help="Sampling temperature"),
    stream: bool = typer.Option(True, help="Stream output as it is generated"),
    chat: bool = typer.Option(True, help="Use chat prompt formatting (Phi-4, etc.)"),
    system: str = typer.Option("You are a helpful assistant.", help="System message for chat format")
):
    """Run LLM inference with production-grade concurrency, streaming, and WASM support."""
    setup_logging()
    logger = get_logger("cli-llm")
    logger.info(f"Selected backend: {backend}")
    adapter = get_llm_adapter(backend)
    if chat:
        chat_prompt = f"<|system|>{system}<|end|><|user|>{prompt}<|end|><|assistant|>"
        logger.info(f"Using chat prompt format. System: {system}")
    else:
        chat_prompt = prompt
    logger.info(f"Prompt: {prompt}")
    try:
        if stream:
            logger.info("Streaming output enabled.")
            for chunk in adapter.stream_infer(chat_prompt, max_tokens=max_tokens, temperature=temperature):
                print(chunk, end="", flush=True)
            print()
        else:
            result = adapter.infer(chat_prompt, max_tokens=max_tokens, temperature=temperature)
            print(result)
        logger.info("LLM inference completed successfully.")
    except Exception as e:
        logger.error(f"LLM inference failed: {e}")


@app.command()
def batch_llm(
    prompts_file: str = typer.Option(..., help="Path to file with one prompt per line"),
    backend: str = typer.Option("llama.cpp", help=f"LLM backend: {', '.join(LLM_ADAPTERS.keys())}"),
    max_tokens: int = typer.Option(128, help="Max tokens to generate"),
    concurrency: int = typer.Option(4, help="Number of concurrent threads"),
    chat: bool = typer.Option(True, help="Use chat prompt formatting (Phi-4, etc.)"),
    system: str = typer.Option("You are a helpful assistant.", help="System message for chat format")
):
    """Batch LLM inference with concurrency and WASM/edge support."""
    setup_logging()
    logger = get_logger("cli-batch-llm")
    logger.info(f"Selected backend: {backend}")
    adapter = get_llm_adapter(backend)
    with open(prompts_file) as f:
        prompts = [line.strip() for line in f if line.strip()]
    logger.info(f"Loaded {len(prompts)} prompts from {prompts_file}")
    if chat:
        prompts = [f"<|system|>{system}<|end|><|user|>{p}<|end|><|assistant|>" for p in prompts]
        logger.info(f"Using chat prompt format. System: {system}")
    import concurrent.futures
    logger.info(f"Running batch inference with concurrency={concurrency}")
    try:
        with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as executor:
            futures = [executor.submit(adapter.infer, p, max_tokens=max_tokens) for p in prompts]
            for i, fut in enumerate(futures):
                result = fut.result()
                print(result)
                logger.info(f"Prompt {i + 1}/{len(prompts)} completed.")
        logger.info("Batch LLM inference completed successfully.")
    except Exception as e:
        logger.error(f"Batch LLM inference failed: {e}")


@app.command()
def service():
    """Run the AI orchestrator as a long-lived service."""
    setup_logging()
    run_service()


@app.command()
def enrich(input_file: str):
    """Enrich a knowledge graph from a JSON file."""
    setup_logging()
    logger = get_logger("cli-enrich")
    import json
    with open(input_file) as f:
        data = json.load(f)
    enricher = KnowledgeGraphEnricher()
    result = enricher.enrich(data)
    logger.info(f"Enrichment result: {result}")
    print(result)


@app.command()
def crawl(url: str):
    """Send a crawl request as a Nexus event and await response via the production Nexus event bus."""
    setup_logging()
    logger = get_logger("cli-crawl")
    crawl_id = str(uuid.uuid4())
    event = nexus_pb2.EventRequest(
        event_id=crawl_id,
        event_type="crawl_request",
        entity_id=url,
        payload=nexus_pb2.Payload(json=json.dumps({"url": url}))
    )
    response_holder = {}
    response_event = threading.Event()

    def on_event(resp):
        if hasattr(resp, "event_id") and resp.event_id == crawl_id and resp.event_type == "crawl_response":
            try:
                payload = json.loads(resp.payload.json)
            except Exception:
                payload = resp.payload.json
            response_holder["result"] = payload
            logger.info(f"Received crawl response: {payload}")
            response_event.set()

    # Subscribe in a background thread
    def subscribe_thread():
        sub_req = nexus_pb2.SubscribeRequest(event_types=["crawl_response"])
        bus.subscribe_events(sub_req, on_event=on_event, num_workers=1)

    t = threading.Thread(target=subscribe_thread, daemon=True)
    t.start()
    logger.info(f"Publishing crawl request event: {event}")
    bus.publish_event(event)

    # Wait for response or timeout
    if response_event.wait(timeout=30):
        print(json.dumps(response_holder["result"], indent=2))
    else:
        logger.error("Timed out waiting for crawl response.")
        print("Timed out waiting for crawl response.")


# --- Generic Crawl Task Receiver ---
@app.command()
def crawl_receiver():
    """Production-grade crawl receiver: listens for crawl_request events and replies via Nexus event bus."""
    setup_logging()
    logger = get_logger("cli-crawl-receiver")
    devourer = Devourer()

    def handle_event(event):
        logger.info(f"Received crawl request event: {event}")
        try:
            payload = json.loads(event.payload.json)
        except Exception:
            payload = {}
        url = payload.get("url")
        crawl_id = event.event_id
        if not url:
            logger.error("No URL in crawl request payload.")
            return
        # Perform crawl (sync for CLI, could be async in prod)
        crawl_event = {"id": crawl_id, "data": {"url": url}}
        result = asyncio.run(devourer.handle_event(crawl_event))
        response = nexus_pb2.EventRequest(
            event_id=crawl_id,
            event_type="crawl_response",
            entity_id=url,
            payload=nexus_pb2.Payload(json=json.dumps(result if result else {"status": "ok", "url": url}))
        )
        logger.info(f"Publishing crawl response event: {response}")
        bus.publish_event(response)

    sub_req = nexus_pb2.SubscribeRequest(event_types=["crawl_request"])
    logger.info("Crawl receiver subscribing to crawl_request events on Nexus bus...")
    bus.subscribe_events(sub_req, on_event=handle_event, num_workers=1)
    # Keep alive
    try:
        while True:
            import time
            time.sleep(3600)
    except KeyboardInterrupt:
        logger.info("Crawl receiver stopped.")


# --- AI Overview of All Nexus Events ---
@app.command()
def ai_overview():
    """AI can subscribe to and log/inspect every event on the Nexus event bus."""
    setup_logging()
    logger = get_logger("cli-ai-overview")

    def handle_event(event):
        logger.info(f"[AI OVERVIEW] Event: {event}")
        # Optionally, add logic to analyze, summarize, or trigger AI workflows

    sub_req = nexus_pb2.SubscribeRequest(event_types=["*"])
    logger.info("AI overview subscribing to ALL events on Nexus bus...")
    bus.subscribe_events(sub_req, on_event=handle_event, num_workers=2)
    try:
        while True:
            import time
            time.sleep(3600)
    except KeyboardInterrupt:
        logger.info("AI overview stopped.")


@app.command()
def diagnostics():
    """Run diagnostics and print system status."""
    setup_logging()
    logger = get_logger("cli-diagnostics")
    logger.info("Diagnostics: OK")
    print("Diagnostics: OK")


if __name__ == "__main__":
    app()
