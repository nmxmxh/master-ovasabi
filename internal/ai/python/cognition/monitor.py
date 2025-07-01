"""
monitor.py: KPI and campaign monitoring for cognitive architecture.
"""

from typing import Dict, Any


class Monitor:
    """
    Production-grade monitor for campaign KPIs, token usage, and system health.
    Can be extended for alerting, reporting, and dashboard integration.
    """
    def __init__(self, config: Dict[str, Any] = None):
        self.config = config or {}
        self.kpi_history = {}

    def log_kpi(self, campaign_id: str, kpi_name: str, value: float):
        """Log a KPI value for a campaign."""
        if campaign_id not in self.kpi_history:
            self.kpi_history[campaign_id] = {}
        self.kpi_history[campaign_id][kpi_name] = value

    def get_kpi(self, campaign_id: str, kpi_name: str) -> float:
        """Get the latest KPI value for a campaign."""
        return self.kpi_history.get(campaign_id, {}).get(kpi_name, 0.0)
