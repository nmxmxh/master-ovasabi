
# Main entrypoint for the AI enrichment module
# Handles event loop, enrichment orchestration, and graceful shutdown

import sys
from utils import setup_logging, get_logger, log_exception


def log_and_import(logger, import_path, import_func):
    logger.info(f"Importing {import_path} ...")
    try:
        result = import_func()
        logger.info(f"Successfully imported {import_path}.")
        return result
    except ModuleNotFoundError as exc:
        logger.error(f"Module import failed: {exc}")
        logger.error("Check that all proto files are generated and PYTHONPATH is set correctly.")
        raise
    except Exception:
        logger.exception(f"Unexpected error during import of {import_path}.")
        raise


def main():
    setup_logging()
    logger = get_logger("ai-enrichment")
    logger.info("Starting AI enrichment module...")

    NexusEventStream = log_and_import(
        logger,
        "bus.nexus_stream.NexusEventStream",
        lambda: __import__('bus.nexus_stream', fromlist=['NexusEventStream']).NexusEventStream
    )
    KnowledgeGraphEnricher = log_and_import(
        logger,
        "knowledge.graph.KnowledgeGraphEnricher",
        lambda: __import__('knowledge.graph', fromlist=['KnowledgeGraphEnricher']).KnowledgeGraphEnricher
    )

    event_stream = NexusEventStream()
    enricher = KnowledgeGraphEnricher()

    try:
        for event in event_stream.listen():
            logger.info(f"Received event: {event}")
            try:
                enriched = enricher.enrich(event)
                event_stream.emit_enriched(enriched)
            except Exception as exc:
                log_exception(logger, "Error enriching event", exc)
    except KeyboardInterrupt:
        logger.info("Shutting down AI enrichment module.")


if __name__ == "__main__":
    logger = get_logger("ai-enrichment-bootstrap")
    logger.info("Bootstrapping OVASABI AI/Knowledge Graph system...")
    logger.info(f"Python version: {sys.version}")
    logger.info("Testing core modules and service connectivity...")
    try:
        Devourer = log_and_import(
            logger,
            "crawler.devourer.Devourer",
            lambda: __import__('crawler.devourer', fromlist=['Devourer']).Devourer
        )
        FederatedOrchestrator = log_and_import(
            logger,
            "orchestrator.federated_orchestrator.FederatedOrchestrator",
            lambda: __import__('orchestrator.federated_orchestrator', fromlist=['FederatedOrchestrator']).FederatedOrchestrator
        )
        PhiEngine = log_and_import(
            logger,
            "inference.phi.PhiEngine",
            lambda: __import__('inference.phi', fromlist=['PhiEngine']).PhiEngine
        )
        WasmEngine = log_and_import(
            logger,
            "inference.wasm.WasmEngine",
            lambda: __import__('inference.wasm', fromlist=['WasmEngine']).WasmEngine
        )
        VectorDB = log_and_import(
            logger,
            "vector_db.VectorDB",
            lambda: __import__('vector_db', fromlist=['VectorDB']).VectorDB
        )
        NexusEventStream = log_and_import(
            logger,
            "bus.nexus_stream.NexusEventStream",
            lambda: __import__('bus.nexus_stream', fromlist=['NexusEventStream']).NexusEventStream
        )
        KnowledgeGraphEnricher = log_and_import(
            logger,
            "knowledge.graph.KnowledgeGraphEnricher",
            lambda: __import__('knowledge.graph', fromlist=['KnowledgeGraphEnricher']).KnowledgeGraphEnricher
        )
        logger.info("All core modules imported successfully.")
        # Test instantiation
        logger.info(f"NexusEventStream: {NexusEventStream()}")
        logger.info(f"KnowledgeGraphEnricher: {KnowledgeGraphEnricher()}")
        logger.info(f"Devourer: {Devourer()}")
        logger.info(f"FederatedOrchestrator: {FederatedOrchestrator()}")
        logger.info(f"PhiEngine: {PhiEngine()}")
        logger.info(f"WasmEngine: {WasmEngine()}")
        logger.info(f"VectorDB: {VectorDB()}")
        logger.info("System bootstrap and smoke test complete. Ready for enrichment.")
    except Exception as exc:
        logger.error(f"System bootstrap failed: {exc}", exc_info=True)
