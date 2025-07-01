# FederatedOrchestrator: Orchestrates federated knowledge graph learning, privacy, and incentives

from utils import get_logger
from typing import List, Callable, Dict, Any
from common.v1 import metadata_pb2
from knowledge.schema import KnowledgeGraphIntrospector


class FederatedOrchestrator:
    """
    Orchestrates federated knowledge graph aggregation, privacy-preserving analytics, and quality attribution.
    - Aggregates graphs from multiple domains/agents (edge, cloud, partners)
    - Applies privacy-preserving aggregation (differential privacy, k-anonymity, etc.)
    - Tracks quality/contribution for rewards/incentives
    - Emits live updates to subscribers (UX, agents, etc.)
    """
    def __init__(self, logger=None):
        self.logger = logger or get_logger("FederatedOrchestrator")
        self.logger.info("Initializing FederatedOrchestrator...")
        self.introspector = KnowledgeGraphIntrospector(logger=self.logger)
        self.subscribers: List[Callable[[Dict[str, Any]], None]] = []
        self.quality_scores: Dict[str, float] = {}  # node/domain -> score
        self.logger.info("FederatedOrchestrator initialized.")

    def subscribe(self, cb: Callable[[Dict[str, Any]], None]):
        self.logger.info(f"Subscriber added: {cb}")
        self.subscribers.append(cb)

    def aggregate(self, graphs: List[metadata_pb2.KnowledgeGraph], strategy: str = "union", privacy: str = "dp") -> Dict[str, Any]:
        self.logger.info(f"Aggregating {len(graphs)} graphs with strategy={strategy}, privacy={privacy}")
        """
        Aggregate graphs with privacy-preserving analytics and quality attribution.
        privacy: 'dp' (differential privacy), 'kanon' (k-anonymity), or 'none'
        """
        agg = self.introspector.federated_aggregate(graphs, strategy=strategy)
        self.logger.info("Aggregation complete. Running privacy-preserving analytics...")
        # Privacy-preserving aggregation
        if privacy == "dp":
            self.logger.info("Applying differential privacy...")
            agg = self._apply_differential_privacy(agg)
        elif privacy == "kanon":
            self.logger.info("Applying k-anonymity...")
            agg = self._apply_k_anonymity(agg)
        # Quality attribution
        self.logger.info("Updating quality scores...")
        self._update_quality_scores(agg)
        # Notify subscribers
        result = {
            "spatial": self.introspector.spatialize(agg),
            "quality_scores": self.quality_scores,
        }
        self.logger.info(f"Notifying {len(self.subscribers)} subscribers.")
        for cb in self.subscribers:
            self.logger.info(f"Notifying subscriber: {cb}")
            cb(result)
        self.logger.info("Aggregation and notification complete.")
        return result

    def _apply_differential_privacy(self, nxg):
        self.logger.info("Applying Laplace noise for differential privacy.")
        # Example: add Laplace noise to node degrees (real DP would be more advanced)
        import numpy as np
        for n in nxg.nodes():
            deg = nxg.degree(n)
            nxg.nodes[n]["dp_degree"] = deg + np.random.laplace(0, 1)
        return nxg

    def _apply_k_anonymity(self, nxg, k: int = 3):
        self.logger.info(f"Applying k-anonymity with k={k}.")
        # Example: group nodes by degree, mask rare degrees
        degs = [nxg.degree(n) for n in nxg.nodes()]
        from collections import Counter
        deg_counts = Counter(degs)
        for n in nxg.nodes():
            if deg_counts[nxg.degree(n)] < k:
                nxg.nodes[n]["k_anon"] = True
        return nxg

    def _update_quality_scores(self, nxg):
        self.logger.info("Calculating betweenness centrality for quality scores.")
        # Example: reward nodes with high betweenness centrality
        import networkx as nx
        bc = nx.betweenness_centrality(nxg)
        for n, score in bc.items():
            self.quality_scores[n] = score

    def reward_top_contributors(self, threshold: float = 0.1) -> List[str]:
        self.logger.info(f"Rewarding contributors with threshold >= {threshold}")
        # Return node/domain names eligible for rewards
        return [n for n, score in self.quality_scores.items() if score >= threshold]
