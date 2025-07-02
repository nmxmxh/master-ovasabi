"""
Core application modules for AI system.
Includes main entrypoint, CLI interface, and LLM registry.
"""

from .main import main
from .llm_registry import LLMAdapter, get_llm_adapter, LLM_ADAPTERS

__all__ = [
    'main',
    'LLMAdapter',
    'get_llm_adapter',
    'LLM_ADAPTERS'
]
