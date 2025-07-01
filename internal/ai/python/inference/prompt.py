# prompt.py: Prompt builder/validator for structured enrichment (JSON, compact, robust)

import json
from utils import get_logger
from pydantic import BaseModel, ValidationError, Field
from datetime import datetime
from typing import List, Optional, Dict, Any


class EnrichmentSchema(BaseModel):
    summary: str = Field(..., max_length=120)
    confidence: float = Field(..., ge=0.0, le=1.0)
    categories: List[str] = Field(..., min_items=1, max_items=3)
    timestamp: Optional[str] = None


class PromptBuilder:
    """
    Robust, production-grade prompt builder and validator for LLM enrichment.
    - Enforces JSON output, schema, and compactness
    - Handles postprocessing, normalization, and error recovery
    - Supports prompt versioning and analytics hooks
    """
    PROMPT_VERSION = "v1.0"

    def __init__(self, logger=None):
        self.logger = logger or get_logger("PromptBuilder")

    def build(self, title: str, text: str, system: Optional[str] = None) -> str:
        sys_block = system or (
            "You are an expert AI assistant for structured information extraction and summarization. "
            "Given any document, text, or data, extract a concise summary (max 30 words), a confidence score (0.0-1.0), "
            "and 1-3 relevant categories. Always respond with a single valid JSON object: "
            "{summary: <string>, confidence: <float>, categories: <list of strings>, timestamp: <ISO8601 string>} "
            "Be robust to noisy, incomplete, or ambiguous input. If unsure, provide your best estimate."
        )
        prompt = (
            f"[INST] <<SYS>>\n{sys_block}\n<</SYS>>\n"
            f"InputTitle: {title}\n"
            f"InputText: {text}\n"
            f"PromptVersion: {self.PROMPT_VERSION}\n"
            f"Timestamp: {datetime.utcnow().isoformat()}\n"
            f"[/INST]"
        )
        return prompt

    def validate(self, output: str) -> Dict[str, Any]:
        # Try to parse and validate output against schema
        try:
            data = json.loads(output)
            # Normalize types and fields
            if "confidence" in data:
                try:
                    data["confidence"] = float(data["confidence"])
                except Exception:
                    data["confidence"] = 0.5
            if "categories" in data and isinstance(data["categories"], str):
                data["categories"] = [data["categories"]]
            # Add timestamp if missing
            if "timestamp" not in data:
                data["timestamp"] = datetime.utcnow().isoformat()
            # Validate with pydantic
            validated = EnrichmentSchema(**data)
            return validated.dict()
        except (json.JSONDecodeError, ValidationError, Exception) as e:
            self.logger.warning(f"Failed to validate LLM output: {output} | Error: {e}")
            # Fallback safe default (higher quality, more generic)
            return {
                "summary": "No valid summary extracted. Input may be too ambiguous or unstructured.",
                "confidence": 0.5,
                "categories": ["Uncategorized"],
                "timestamp": datetime.utcnow().isoformat(),
            }
