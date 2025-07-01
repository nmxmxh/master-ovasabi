# Devourer: Knowledge Crawler for Local and External Knowledge Graphs
# Crawls codebase, docs, web, APIs, and external sources to feed the knowledge graph


import os
from utils import get_logger
import requests
from typing import List, Dict, Any, Optional
from common.v1 import metadata_pb2
import asyncio
import ai_web
from bus import nexus_stream


class Devourer:
    """
    Devourer is a universal knowledge orchestrator/enricher for crawl results:
    - Orchestrates crawl requests via the Go service (using Nexus event bus)
    - Enriches and stores crawl results in the _web table
    - Listens to crawl events/results from the Nexus event stream
    """
    def __init__(self, logger=None, webdb=None, nexus_client=None):
        self.logger = logger or get_logger("Devourer")
        self.webdb = webdb or ai_web.WebDB()
        self.nexus_client = nexus_client or nexus_stream.NexusStreamClient()

    async def store_crawl_result(self, result: dict, parent_ids=None, model_hash=None, metadata_id=None):
        """
        Store a crawl result as a node in the _web table.
        Performs schema validation and sanitization before storage.
        """
        from pydantic import BaseModel, ValidationError

        class CrawlResultSchema(BaseModel):
            uuid: str
            status: str
            extracted_links: list = []
            error_message: str = None
            # Add more fields as needed

        try:
            # Sanitize and validate
            clean = {k: v for k, v in result.items() if k in CrawlResultSchema.__fields__}
            validated = CrawlResultSchema(**clean).dict()
            await self.webdb.insert_node(
                node_type='crawl_result',
                node_data=validated,
                parent_ids=parent_ids,
                model_hash=model_hash,
                metadata_id=metadata_id
            )
        except ValidationError as ex:
            self.logger.error(f"Invalid crawl result: {ex}")
        except Exception as ex:
            self.logger.error(f"Failed to store crawl result: {ex}")

    async def orchestrate_crawl(self, crawl_task: dict):
        """
        Orchestrate a crawl by publishing a crawl task event to the Nexus event bus (Go handles crawling).
        """
        await self.nexus_client.publish_event("crawl_task", crawl_task)
        self.logger.info(
            f"Published crawl task to Nexus: {crawl_task.get('uuid')}"
        )

    async def listen_to_crawl_results(self, stop_event: asyncio.Event = None):
        """
        Listen to crawl result events from the Nexus event bus and store/enrich in _web.
        Handles graceful shutdown and retries with FallbackHandler.
        """
        from orchestrator.fallback import FallbackHandler

        handler = FallbackHandler()
        try:
            async for event in self.nexus_client.live_event_stream(event_type="crawl_result"):
                if stop_event and stop_event.is_set():
                    self.logger.info("Graceful shutdown of crawl result listener.")
                    break
                result = event.payload if hasattr(event, 'payload') else event
                await handler.run_with_fallback(self.store_crawl_result, result)
                self.logger.info(
                    f"Stored crawl result in _web: {getattr(result, 'uuid', None)}"
                )
        except Exception as ex:
            self.logger.error(f"Listener error: {ex}")

    def crawl_codebase(self, root_dir: str) -> List[Dict[str, Any]]:
        """
        Walk the codebase, extract file/module/class/function metadata, and run static analysis.
        All extracted metadata is sanitized and validated before use.
        """
        import ast
        import pandas as pd
        import numpy as np
        from pydantic import BaseModel, ValidationError

        try:
            import tree_sitter  # noqa: F401
        except ImportError:
            pass

        class FileMetaSchema(BaseModel):
            path: str
            size: int
            sample: str
            type: str
            function_count: int = 0
            class_count: int = 0
            line_count: int = 0
            static_analysis_error: str = None

        results = []
        for dirpath, _, filenames in os.walk(root_dir):
            for fname in filenames:
                if fname.endswith((".py", ".go", ".js", ".ts", ".md")):
                    fpath = os.path.join(dirpath, fname)
                    meta = self.extract_file_metadata(fpath)
                    if meta and fname.endswith(".py"):
                        try:
                            with open(fpath, "r", encoding="utf-8", errors="ignore") as f:
                                source = f.read()
                            tree = ast.parse(source)
                            meta["function_count"] = len([
                                n for n in ast.walk(tree) if isinstance(n, ast.FunctionDef)
                            ])
                            meta["class_count"] = len([
                                n for n in ast.walk(tree) if isinstance(n, ast.ClassDef)
                            ])
                            meta["line_count"] = source.count("\n")
                        except Exception as ex:
                            meta["static_analysis_error"] = str(ex)
                    # Sanitize and validate
                    try:
                        validated = FileMetaSchema(**meta).dict()
                        results.append(validated)
                    except ValidationError as ex:
                        self.logger.warning(f"Invalid file metadata: {ex}")
        # Example: batch analysis with pandas
        if results:
            df = pd.DataFrame(results)
            print("[Devourer] Codebase static analysis summary:")
            print(df.describe(include="all"))
            num_df = df.select_dtypes(include=[np.number])
            if not num_df.empty and len(num_df) > 4:
                from sklearn.ensemble import IsolationForest
                from sklearn.cluster import KMeans
                # Anomaly detection
                model = IsolationForest(n_estimators=50, contamination=0.1, random_state=42)
                preds = model.fit_predict(num_df)
                anomaly_indices = np.where(preds == -1)[0]
                if len(anomaly_indices) > 0:
                    print(f"[Devourer] Anomalous files: {df.iloc[anomaly_indices]['path'].tolist()}")
                else:
                    print("[Devourer] No anomalies detected in code metrics.")
                # Clustering
                n_clusters = min(3, len(num_df))
                kmeans = KMeans(n_clusters=n_clusters, random_state=42, n_init=10)
                cluster_labels = kmeans.fit_predict(num_df)
                df['cluster'] = cluster_labels
                print("[Devourer] Codebase clustering summary:")
                print(df.groupby('cluster').size())
            else:
                print("[Devourer] Not enough numeric data for code anomaly detection or clustering.")
        return results

    def extract_file_metadata(self, fpath: str) -> Optional[Dict[str, Any]]:
        try:
            with open(fpath, "r", encoding="utf-8", errors="ignore") as f:
                content = f.read(4096)  # Only sample for speed
            return {
                "path": fpath,
                "size": os.path.getsize(fpath),
                "sample": content[:256],
                "type": os.path.splitext(fpath)[-1],
            }
        except Exception as ex:
            self.logger.warning(f"Failed to extract metadata for {fpath}: {ex}")
            return None

    def crawl_web(self, url: str) -> Dict[str, Any]:
        """
        Fetch and extract metadata from a web page. Sanitizes and validates output.
        """
        from pydantic import BaseModel, ValidationError

        class WebMetaSchema(BaseModel):
            url: str
            status: int
            title: str = None
            sample: str = None
            error: str = None

        try:
            resp = requests.get(url, timeout=5)
            meta = {
                "url": url,
                "status": resp.status_code,
                "title": self._extract_title(resp.text),
                "sample": resp.text[:256],
            }
        except Exception as ex:
            meta = {"url": url, "error": str(ex), "status": 0}
        try:
            validated = WebMetaSchema(**meta).dict()
            return validated
        except ValidationError as ex:
            self.logger.warning(f"Invalid web metadata: {ex}")
            return meta

    def emit_metadata(self, meta: Dict[str, Any]) -> metadata_pb2.Metadata:
        """Convert extracted metadata to protobuf for ingestion."""
        pb = metadata_pb2.Metadata()
        for k, v in meta.items():
            setattr(pb, k, v)
        return pb

    async def find_similar_results(self, query_embedding, top_k=5, backend="faiss"):
        """
        Vector similarity search for crawl results in _web using FAISS/Qdrant.
        """
        import numpy as np
        import vector_db
        # Example: fetch all crawl_result nodes and their embeddings
        nodes = await self.webdb.get_nodes(node_type="crawl_result")
        embeddings = []
        node_ids = []
        for n in nodes:
            emb = n.node_data.get("embedding")
            if emb:
                embeddings.append(np.array(emb, dtype=np.float32))
                node_ids.append(n.id)
        if not embeddings:
            return []
        embeddings = np.stack(embeddings)
        vdb = vector_db.VectorDB(backend=backend, dim=embeddings.shape[1])
        vdb.add(embeddings)
        D, idxs = vdb.index.search(np.array([query_embedding], dtype=np.float32), top_k)
        return [node_ids[i] for i in idxs[0]]

    def run(self, codebase_root: str, urls: List[str]) -> List[metadata_pb2.Metadata]:
        """Crawl codebase and web, emit metadata for knowledge graph ingestion."""
        metas = []
        for meta in self.crawl_codebase(codebase_root):
            metas.append(self.emit_metadata(meta))
        for url in urls:
            meta = self.crawl_web(url)
            metas.append(self.emit_metadata(meta))
        return metas
