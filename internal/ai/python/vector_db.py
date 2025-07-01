"""
vector_db.py: Unified interface for vector database integration (FAISS, Qdrant, Postgres/pgvector)
- Supports semantic search, RAG, and embedding storage for LLM workflows
- Pluggable for local, serverless, or production deployments
"""

import numpy as np
import os
import psycopg2
import psycopg2.extras
from typing import List, Dict, Any, Optional

try:
    import faiss
except ImportError:
    faiss = None

try:
    from qdrant_client import QdrantClient
except ImportError:
    QdrantClient = None


from utils import get_logger


class VectorDB:
    def __init__(self, backend: str = "faiss", dim: int = 384, **kwargs):
        self.backend = backend
        self.dim = dim
        self.logger = kwargs.get("logger") or get_logger("VectorDB")
        if backend == "faiss" and faiss:
            self.index = faiss.IndexFlatL2(dim)
        elif backend == "qdrant" and QdrantClient:
            self.client = QdrantClient(**kwargs)
            self.collection = kwargs.get("collection", "default")
            # Ensure collection exists
            if not self.client.collection_exists(self.collection):
                self.client.recreate_collection(collection_name=self.collection, vectors_config={"size": dim, "distance": "Cosine"})
        else:
            raise ValueError(f"Unsupported or missing vector DB backend: {backend}")

    def add(self, vectors: np.ndarray, payloads: Optional[List[Dict[str, Any]]] = None):
        if self.backend == "faiss":
            self.index.add(vectors)
        elif self.backend == "qdrant":
            ids = list(range(self.client.count(self.collection), self.client.count(self.collection) + len(vectors)))
            self.client.upload_collection(
                collection_name=self.collection,
                vectors=vectors.tolist(),
                payload=payloads or [{} for _ in vectors],
                ids=ids
            )

    def search(self, query: np.ndarray, top_k: int = 5) -> List[Any]:
        if self.backend == "faiss":
            D, idx = self.index.search(query, top_k)
            return idx.tolist()
        elif self.backend == "qdrant":
            hits = self.client.search(collection_name=self.collection, query_vector=query.tolist()[0], limit=top_k)
            return hits

    def count(self) -> int:
        if self.backend == "faiss":
            return self.index.ntotal
        elif self.backend == "qdrant":
            return self.client.count(self.collection)


class PgVectorDB(VectorDB):
    def __init__(self, dim: int = 384, **kwargs):
        self.dim = dim
        self.logger = kwargs.get("logger") or get_logger("PgVectorDB")
        self.table = kwargs.get("table", "service_embedding")
        self.conn = psycopg2.connect(os.getenv("AI_PG_URL"))
        self.conn.autocommit = True

    def add(self, vectors: np.ndarray, master_ids: list, campaign_ids: list, metadatas: Optional[list] = None):
        # Batch insert, parameterized, safe
        with self.conn.cursor() as cur:
            psycopg2.extras.execute_batch(
                cur,
                f"""
                INSERT INTO {self.table} (master_id, campaign_id, embedding, metadata)
                VALUES (%s, %s, %s, %s)
                """,
                [
                    (mid, cid, v.tolist(), md or {})
                    for v, mid, cid, md in zip(vectors, master_ids, campaign_ids, metadatas or [{}] * len(vectors))
                ]
            )

    def search(self, query: np.ndarray, campaign_id: int, top_k: int = 5) -> list:
        # ANN search using pgvector <-> operator, filtered by campaign
        with self.conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute(
                f"""
                SELECT id, master_id, embedding <-> %s AS distance, metadata
                FROM {self.table}
                WHERE campaign_id = %s
                ORDER BY distance ASC
                LIMIT %s
                """,
                (query.tolist(), campaign_id, top_k)
            )
            return cur.fetchall()

    def count(self, campaign_id: int = None) -> int:
        with self.conn.cursor() as cur:
            if campaign_id is not None:
                cur.execute(f"SELECT COUNT(*) FROM {self.table} WHERE campaign_id = %s", (campaign_id,))
            else:
                cur.execute(f"SELECT COUNT(*) FROM {self.table}")
            return cur.fetchone()[0]
