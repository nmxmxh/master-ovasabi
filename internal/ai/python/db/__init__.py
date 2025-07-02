"""
Database modules for AI system.
Includes PostgreSQL, vector databases, and web knowledge graph interfaces.
"""

from .db import ai_table, metadata_master_table
from .vector_db import VectorDB
from .vector_db_registry import VectorDBAdapter, ChromaDBAdapter, MilvusAdapter
from .ai_web import web_table

__all__ = [
    'ai_table',
    'metadata_master_table',
    'VectorDB',
    'VectorDBAdapter',
    'ChromaDBAdapter',
    'MilvusAdapter',
    'web_table'
]
