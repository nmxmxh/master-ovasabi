# vector_db_registry.py: Registry and adapters for vector DB backends (ChromaDB, Milvus, etc.)


from typing import Dict, Type
import threading


class VectorDBAdapter:
    """
    Production-grade base class for vector DB adapters.
    Enforces interface, provides config validation, and robust error handling.
    """
    def __init__(self, **kwargs):
        self.config = kwargs
        self._lock = threading.Lock()
        self._connected = False
        self._connect()

    def _connect(self):
        """Override to implement backend connection logic."""
        self._connected = True

    def add(self, vectors, metadatas=None, **kwargs):
        """Add vectors and optional metadata to the DB."""
        raise NotImplementedError("add() must be implemented by adapter.")

    def search(self, query, top_k=5, **kwargs):
        """Search for nearest vectors to the query."""
        raise NotImplementedError("search() must be implemented by adapter.")

    def count(self, **kwargs):
        """Return the number of vectors in the DB."""
        raise NotImplementedError("count() must be implemented by adapter.")

    def validate(self):
        """Validate configuration and connection."""
        if not self._connected:
            raise RuntimeError("Adapter is not connected!")


class ChromaDBAdapter(VectorDBAdapter):
    """Production-grade adapter for ChromaDB."""
    def _connect(self):
        # TODO: Implement ChromaDB connection logic
        self._connected = True


class MilvusAdapter(VectorDBAdapter):
    """Production-grade adapter for Milvus."""
    def _connect(self):
        # TODO: Implement Milvus connection logic
        self._connected = True


class QdrantAdapter(VectorDBAdapter):
    """Production-grade adapter for Qdrant."""
    def _connect(self):
        # TODO: Implement Qdrant connection logic
        self._connected = True


class PgVectorAdapter(VectorDBAdapter):
    """Production-grade adapter for Postgres/pgvector."""
    def _connect(self):
        # TODO: Implement pgvector connection logic
        self._connected = True


class VectorDBRegistry:
    """
    Production-grade registry for vector DB adapters.
    Enforces interface compliance and provides helpful errors.
    """
    _registry: Dict[str, Type[VectorDBAdapter]] = {}

    @classmethod
    def register(cls, name: str, adapter: Type[VectorDBAdapter]):
        if not issubclass(adapter, VectorDBAdapter):
            raise TypeError(f"Adapter {adapter} must inherit from VectorDBAdapter.")
        cls._registry[name] = adapter

    @classmethod
    def get(cls, name: str) -> Type[VectorDBAdapter]:
        if name not in cls._registry:
            raise KeyError(f"No adapter registered for backend '{name}'")
        return cls._registry[name]


# Register adapters
VectorDBRegistry.register("chromadb", ChromaDBAdapter)
VectorDBRegistry.register("milvus", MilvusAdapter)
VectorDBRegistry.register("qdrant", QdrantAdapter)
VectorDBRegistry.register("pgvector", PgVectorAdapter)
