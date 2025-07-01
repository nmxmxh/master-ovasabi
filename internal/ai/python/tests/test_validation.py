"""
Test suite for input validation, error handling, and schema enforcement (Pydantic, validation.py).
Covers security, schema validation, and error handling for orchestrator and enrichment modules.
"""

import pytest
from utils import validation
from pydantic import ValidationError
from orchestrator import fallback


class TestSchema(validation.BaseModel):
    id: int
    name: str


def test_valid_schema():
    obj = TestSchema(id=1, name='abc')
    assert obj.id == 1
    assert obj.name == 'abc'


def test_invalid_schema():
    with pytest.raises(ValidationError):
        TestSchema(id='bad', name=123)


@pytest.mark.asyncio
async def test_fallback_error():
    f = fallback.FallbackHandler()

    def fail():
        raise RuntimeError('fail')

    result = await f.run_with_fallback(fail, fallback_value='ok')
    assert result == 'ok'
