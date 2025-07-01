"""
goal.py: Goal and campaign representation for cognitive architecture.
"""

from typing import Dict, Optional


class Goal:
    """
    Represents a campaign or autonomous goal for the cognitive system.
    Includes metadata, progress, and evaluation criteria.
    """
    def __init__(self, goal_id: str, description: str, kpi_targets: Optional[Dict[str, float]] = None):
        self.goal_id = goal_id
        self.description = description
        self.kpi_targets = kpi_targets or {}
        self.progress = 0.0
        self.status = "pending"

    def update_progress(self, value: float):
        """Update progress toward the goal (0.0-1.0)."""
        self.progress = min(max(value, 0.0), 1.0)
        if self.progress >= 1.0:
            self.status = "complete"

    def evaluate(self, kpis: Dict[str, float]) -> bool:
        """Evaluate if goal is met based on KPIs."""
        for k, target in self.kpi_targets.items():
            if k not in kpis or kpis[k] < target:
                return False
        return True
