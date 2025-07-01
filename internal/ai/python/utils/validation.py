
# Validation and utility helpers for AI enrichment, orchestration, and event-driven flows
from typing import Any, Optional, Callable
from pydantic import BaseModel, ValidationError


class EventModel(BaseModel):
    event_id: str
    payload: Optional[dict]
    source: Optional[str]
    timestamp: Optional[float]


def validate_event(event: Any) -> bool:
    """
    Validate incoming event structure (using Pydantic schema or custom logic).
    Accepts dict, object with proto, or Pydantic model.
    """
    if isinstance(event, dict):
        try:
            EventModel(**event)
            return True
        except ValidationError:
            return False
    if hasattr(event, 'proto') and hasattr(event.proto, 'event_id'):
        return True
    if isinstance(event, EventModel):
        return True
    return False


def validate_schema(schema: BaseModel, data: dict) -> bool:
    """
    Validate a dict against a Pydantic schema.
    """
    try:
        schema.parse_obj(data)
        return True
    except ValidationError:
        return False


def safe_execute(fn: Callable, *args, fallback=None, **kwargs):
    """
    Execute a function, return fallback on exception.
    """
    try:
        return fn(*args, **kwargs)
    except Exception:
        return fallback


def sanitize_input(data: dict, allowed_fields: Optional[list] = None) -> dict:
    """
    Remove keys not in allowed_fields from data.
    """
    if allowed_fields is None:
        return data
    return {k: v for k, v in data.items() if k in allowed_fields}
