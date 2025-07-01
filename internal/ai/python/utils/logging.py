# Logging and tracing utilities for AI enrichment, orchestration, and event-driven modules
import logging
import sys
from typing import Optional


def setup_logging(level: int = logging.INFO, log_to_file: Optional[str] = None):
    """
    Set up logging with optional file output and standard format.
    """
    handlers = [logging.StreamHandler(sys.stdout)]
    if log_to_file:
        handlers.append(logging.FileHandler(log_to_file))
    logging.basicConfig(
        level=level,
        format="[%(asctime)s] %(levelname)s %(name)s: %(message)s",
        handlers=handlers
    )


def get_logger(name: str) -> logging.Logger:
    """
    Get a logger with the given name.
    """
    return logging.getLogger(name)


def log_exception(logger: logging.Logger, msg: str, exc: Exception):
    logger.error(f"{msg}: {exc}", exc_info=True)
