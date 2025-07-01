# Schema helpers for knowledge graph validation and introspection

from common.v1 import metadata_pb2


import networkx as nx
from typing import Any, Dict, List, Tuple, Optional
from utils import get_logger


class KnowledgeGraphIntrospector:
    def federated_aggregate(self, graphs: List[metadata_pb2.KnowledgeGraph], strategy: str = "union") -> nx.DiGraph:
        """
        Federated learning/aggregation: merge knowledge graphs from multiple domains/nodes.
        Supports strategies: 'union', 'intersection', 'weighted'.
        Returns a unified NetworkX graph for global reasoning.
        """
        nxgs = [self._to_networkx(g) for g in graphs]
        if strategy == "union":
            agg = nx.compose_all(nxgs)
        elif strategy == "intersection":
            agg = nx.intersection_all(nxgs)
        elif strategy == "weighted":
            agg = nx.DiGraph()
            for nxg in nxgs:
                for n, d in nxg.nodes(data=True):
                    if agg.has_node(n):
                        # Merge/average attributes
                        for k, v in d.items():
                            if isinstance(v, (int, float)):
                                agg.nodes[n][k] = (agg.nodes[n].get(k, 0) + v) / 2
                            else:
                                agg.nodes[n][k] = v
                    else:
                        agg.add_node(n, **d)
                for u, v, ed in nxg.edges(data=True):
                    agg.add_edge(u, v, **ed)
        else:
            raise ValueError(f"Unknown federated aggregation strategy: {strategy}")
        return agg

    def federated_introspect(self, graphs: List[metadata_pb2.KnowledgeGraph], strategy: str = "union", real_time_update_cb: Optional[Any] = None) -> Dict[str, Any]:
        """Federated introspection: aggregate multiple graphs, run analytics, and provide
        orchestrator/UX feedback.
        """
        agg = self.federated_aggregate(graphs, strategy=strategy)
        result = {
            "spatial": self.spatialize(agg),
            "degree_centrality": nx.degree_centrality(agg),
            "betweenness_centrality": nx.betweenness_centrality(agg),
            "referential_metadata": {n: d.get('metadata', {}) for n, d in agg.nodes(data=True)},
        }
        if real_time_update_cb:
            real_time_update_cb(result)
        return result
    """
    Advanced introspection and pattern-matching for local and global knowledge graphs.
    - Validates, visualizes, and connects knowledge at a multi-dimensional, biological level.
    - Supports local codebase analysis and external world knowledge integration.
    - Uses graph theory, pattern mining, and spatial reasoning for deep understanding.
    """
    def __init__(self, logger=None):
        self.logger = logger or get_logger("KnowledgeGraphIntrospector")

    def validate(self, graph: metadata_pb2.KnowledgeGraph) -> bool:
        # Check for required fields and structural soundness
        if not graph.services:
            self.logger.warning("Knowledge graph missing services.")
            return False
        # Example: check for cycles, disconnected components, etc.
        nxg = self._to_networkx(graph)
        if not nx.is_weakly_connected(nxg):
            self.logger.warning("Knowledge graph is not fully connected.")
        # ... more advanced validation ...
        return True

    def _to_networkx(self, graph: metadata_pb2.KnowledgeGraph) -> nx.DiGraph:
        """
        Convert protobuf graph to NetworkX DiGraph for analysis, including referential metadata.
        Nodes and edges are annotated with metadata for advanced analytics and orchestrator use.
        """
        nxg = nx.DiGraph()
        for svc in graph.services:
            node_attrs = {f: getattr(svc, f) for f in svc.DESCRIPTOR.fields_by_name}
            # Attach referential metadata if present
            if hasattr(svc, 'metadata') and svc.metadata:
                node_attrs['metadata'] = self._metadata_to_dict(svc.metadata)
            nxg.add_node(svc.name, type="service", **node_attrs)
            for dep in getattr(svc, "dependencies", []):
                nxg.add_edge(svc.name, dep)
        # Add patterns, entities, referential attributes, etc. as needed
        # Example: add pattern nodes/edges
        if hasattr(graph, 'patterns'):
            for pat in getattr(graph, 'patterns', []):
                nxg.add_node(
                    pat.name,
                    type="pattern",
                    **{f: getattr(pat, f) for f in pat.DESCRIPTOR.fields_by_name}
                )
                for svc in getattr(pat, 'applies_to', []):
                    nxg.add_edge(pat.name, svc)
        return nxg

    def _metadata_to_dict(self, meta) -> Dict[str, Any]:
        # Convert protobuf Metadata to dict for analytics/orchestration
        result = {}
        if hasattr(meta, 'service_specific'):
            for k, v in meta.service_specific.items():
                result[k] = v
        # Add canonical keys (see Go metadata.go)
        attrs = [
            'task_details', 'type', 'target', 'depth',
            'result_summary', 'status', 'extracted_links', 'error_message',
            'security_analysis', 'malware_detected', 'pii_redacted', 'high_entropy',
            'video_metadata', 'audio_path', 'duration', 'format', 'video_codec'
        ]
        for attr in attrs:
            if hasattr(meta, attr):
                result[attr] = getattr(meta, attr)
        return result

    def pattern_match(self, graph: metadata_pb2.KnowledgeGraph, pattern: Dict[str, Any]) -> List[Tuple[str, str]]:
        """
        Advanced pattern mining: find subgraphs, motifs, anti-patterns, and semantic relations.
        Returns a list of (node, match_type) tuples.
        """
        nxg = self._to_networkx(graph)
        matches = []
        # Property match
        for n, d in nxg.nodes(data=True):
            if all(d.get(k) == v for k, v in pattern.items()):
                matches.append((n, "property_match"))
        # Motif/subgraph isomorphism (e.g., triangle, star, clique)
        # Example: find all triangles (3-node cycles)
        triangles = [c for c in nx.simple_cycles(nxg) if len(c) == 3]
        for tri in triangles:
            matches.append((tuple(tri), "triangle_motif"))
        # Anomaly detection: singleton nodes, high-degree hubs, bridges
        for n, d in nxg.degree():
            if d == 0:
                matches.append((n, "singleton"))
            elif d > 5:
                matches.append((n, "hub"))
        bridges = list(nx.bridges(nxg.to_undirected()))
        for b in bridges:
            matches.append((b, "bridge"))
        # ... add more advanced pattern mining as needed ...
        return matches

    def visualize(self, nxg: nx.DiGraph, dim: int = 3, highlight: Optional[List[str]] = None, out_file: Optional[str] = None):
        """
        Visualization hook: plot the graph in 2D/3D, highlight patterns, save to file if needed.
        """
        import matplotlib.pyplot as plt
        from mpl_toolkits.mplot3d import Axes3D  # noqa: F401
        pos = self.spatialize(nxg, dim=dim)
        if dim == 3:
            fig = plt.figure(figsize=(10, 8))
            ax = fig.add_subplot(111, projection='3d')
            xs, ys, zs = zip(
                *[pos[n] for n in nxg.nodes()]
            )
            ax.scatter(xs, ys, zs, c='b', marker='o')
            for n, (x, y, z) in pos.items():
                ax.text(x, y, z, n, fontsize=8)
            for u, v in nxg.edges():
                x = [pos[u][0], pos[v][0]]
                y = [pos[u][1], pos[v][1]]
                z = [pos[u][2], pos[v][2]]
                ax.plot(x, y, z, c='gray')
        else:
            nx.draw(nxg, pos, with_labels=True, node_color='b', edge_color='gray')
        if highlight:
            # Highlight nodes/edges
            nx.draw_networkx_nodes(nxg, pos, nodelist=highlight, node_color='r')
        if out_file:
            plt.savefig(out_file)
        else:
            plt.show()

    def anomaly_report(self, graph: metadata_pb2.KnowledgeGraph) -> Dict[str, Any]:
        """
        Run anomaly detection: find disconnected, cyclic, or suspicious patterns.
        Returns a report dict.
        """
        nxg = self._to_networkx(graph)
        report = {"singletons": [], "cycles": [], "hubs": [], "bridges": []}
        for n, d in nxg.degree():
            if d == 0:
                report["singletons"].append(n)
            elif d > 5:
                report["hubs"].append(n)
        report["cycles"] = [c for c in nx.simple_cycles(nxg) if len(c) > 2]
        report["bridges"] = list(nx.bridges(nxg.to_undirected()))
        # ... add more anomaly types as needed ...
        return report

    def connect_external_knowledge(self, graph: metadata_pb2.KnowledgeGraph, external_kg: Any) -> nx.DiGraph:
        """
        Integrate external world knowledge (e.g., Wikidata, web crawls, code graphs) into the local knowledge graph.
        Returns a merged NetworkX graph.
        """
        nxg = self._to_networkx(graph)
        # Assume external_kg is a NetworkX graph or convertible
        if hasattr(external_kg, "nodes"):
            nxg = nx.compose(nxg, external_kg)
        # ... add logic for mapping, alignment, and semantic linking ...
        return nxg

    def spatialize(self, nxg: nx.DiGraph, dim: int = 3) -> Dict[str, Tuple[float, ...]]:
        """
        Project the knowledge graph into N-dimensional space for visualization and biological-style reasoning.
        Returns a dict of node -> coordinates.
        """
        if dim == 3:
            pos = nx.spring_layout(nxg, dim=3, seed=42)
        else:
            pos = nx.spring_layout(nxg, dim=dim, seed=42)
        return pos

    def introspect(self, graph: metadata_pb2.KnowledgeGraph, external_kg: Optional[Any] = None, real_time_update_cb: Optional[Any] = None) -> Dict[str, Any]:
        """
        Full introspection: validate, pattern match, connect external, and spatialize.
        Supports real-time graph updates and orchestrator feedback via callback.
        Returns a summary dict for agent/AI consumption.
        """
        result = {"valid": self.validate(graph)}
        nxg = self._to_networkx(graph)
        if external_kg:
            nxg = self.connect_external_knowledge(graph, external_kg)
        result["spatial"] = self.spatialize(nxg)
        # Example: find all singleton services
        result["singletons"] = [n for n, d in nxg.degree() if d == 0]
        # Advanced analytics: degree distribution, centrality, referential metadata
        result["degree_centrality"] = nx.degree_centrality(nxg)
        result["betweenness_centrality"] = nx.betweenness_centrality(nxg)
        result["referential_metadata"] = {n: d.get('metadata', {}) for n, d in nxg.nodes(data=True)}
        # Real-time update hook for orchestrator/UX
        if real_time_update_cb:
            real_time_update_cb(result)
        # ... add more biological/semantic analysis ...
        return result
