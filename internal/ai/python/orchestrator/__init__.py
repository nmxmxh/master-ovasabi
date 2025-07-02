from utils import get_logger
from typing import Any, Dict, List
from db import db as db_module
from db import ai_web
from . import fallback
from cognition.manager import CognitiveManager
from cognition.optimizer import Optimizer
from cognition.monitor import Monitor


class PatternOrchestrator:
    """
    Production-grade orchestrator for monitoring, storing, and orchestrating patterns.
    Integrates with AsyncEnrichmentDB for pattern persistence and retrieval.
    """
    def __init__(self, db=None, logger=None, cognition_config=None):
        self.logger = logger or get_logger("PatternOrchestrator")
        self.db = db or db_module.AsyncEnrichmentDB()
        self.patterns: List[Dict[str, Any]] = []  # In-memory cache of patterns
        # Wire in cognitive modules
        self.cognitive_manager = CognitiveManager(config=cognition_config)
        self.optimizer = Optimizer(config=cognition_config)
        self.monitor = Monitor(config=cognition_config)

    async def monitor_patterns(self, pattern_stream: List[Dict[str, Any]]):
        """
        Monitor incoming patterns (from batcher, metrics, or external sources),
        orchestrate actions, and persist new/interesting patterns to DB.
        """
        for pattern in pattern_stream:
            if self.is_new_pattern(pattern):
                self.patterns.append(pattern)
                await self.save_pattern_to_db(pattern)
                self.logger.info(f"New pattern detected and saved: {pattern}")
                # Log KPI for cognitive monitoring
                campaign_id = pattern.get('campaign_id', 'default')
                kpi_name = pattern.get('category', 'pattern')
                value = pattern.get('score', 1.0)
                self.monitor.log_kpi(campaign_id, kpi_name, value)
            else:
                self.logger.debug(f"Pattern already known: {pattern}")
            # Orchestrate actions based on pattern (e.g., trigger fallback, alert, etc.)
            await self.orchestrate(pattern)
        # After monitoring, run closed-loop optimization
        self.optimizer.optimize(self.cognitive_manager.campaigns, self.monitor.kpi_history)

    def is_new_pattern(self, pattern: Dict[str, Any]) -> bool:
        # Simple deduplication by hash or content
        return pattern not in self.patterns

    async def save_pattern_to_db(self, pattern: Dict[str, Any]):
        # Store pattern as AI-relevant metadata
        ai_fields = {k: v for k, v in pattern.items() if isinstance(k, str)}
        meta = {
            'entity_type': 'pattern',
            'category': pattern.get('category', 'pattern'),
            'environment': pattern.get('environment', 'default'),
            'role': pattern.get('role', 'system'),
            'metadata': ai_fields
        }
        await self.db.batch_insert_metadata([meta])

    async def orchestrate(self, pattern: Dict[str, Any]):
        # Example: trigger fallback or alert if anomaly or critical pattern
        if pattern.get('anomaly', False):
            self.logger.warning(f"Anomaly pattern detected: {pattern}")
            handler = fallback.FallbackHandler()
            await handler.run_with_fallback(self.save_pattern_to_db, pattern)
            # Example: spend tokens for anomaly handling
            campaign_id = pattern.get('campaign_id', 'default')
            self.cognitive_manager.spend_tokens(campaign_id, amount=1.0)
        # Add more orchestration logic as needed

    async def get_patterns_from_db(self, **filters) -> List[Dict[str, Any]]:
        # Retrieve patterns from DB matching filters
        metas = await self.db.get_ai_relevant_metadata(categories=['pattern'], **filters)
        return [m.metadata for m in metas]


class WebOrchestrator:
    """
    Minimal orchestrator for storing and linking inferences/patterns in the _web table.
    Supports multi-domain, similarity, and chain expansion.
    """
    def __init__(self, webdb=None, logger=None):
        self.logger = logger
        self.webdb = webdb or ai_web.WebDB()

    async def store_inference(
        self, node_type: str, node_data: dict, parent_ids=None, edge_type=None, edge_data=None, model_hash=None, metadata_id=None
    ):
        """
        Store a new inference/pattern node in the _web table.
        """
        node_id = await self.webdb.insert_node(
            node_type=node_type,
            node_data=node_data,
            parent_ids=parent_ids,
            edge_type=edge_type,
            edge_data=edge_data,
            model_hash=model_hash,
            metadata_id=metadata_id
        )
        if self.logger:
            self.logger.info(f"Stored node in _web: {node_id}")
        return node_id

    async def find_similar(
        self, node_data: dict, node_type: str = None, threshold: float = 0.8
    ) -> list:
        """
        Minimal similarity check: fetch nodes of the same type and compare dimensions/fields.
        Extend with vector search as needed.
        """
        nodes = await self.webdb.get_nodes(node_type=node_type)
        # Example: compare by shared keys/values
        similar = []
        for n in nodes:
            shared = set(node_data.keys()) & set(n.node_data.keys())
            if shared:
                score = sum(1 for k in shared if node_data[k] == n.node_data[k]) / max(len(node_data), 1)
                if score >= threshold:
                    similar.append(n)
        return similar

    async def expand_chain(self, start_id):
        """
        Recursively fetch and return the chain of knowledge/inference from a starting node.
        """
        return await self.webdb.get_chain(start_id)

# Usage:
# orchestrator = PatternOrchestrator()
# await orchestrator.monitor_patterns(pattern_stream)
# patterns = await orchestrator.get_patterns_from_db()
#
# web_orchestrator = WebOrchestrator()
# await web_orchestrator.store_inference('inference', {...})
# similar = await web_orchestrator.find_similar({...})
# chain = await web_orchestrator.expand_chain(start_id)
