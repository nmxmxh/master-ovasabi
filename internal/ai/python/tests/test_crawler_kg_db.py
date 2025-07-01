"""
Test suite for Devourer (crawler), KnowledgeGraphEnricher, and DB/AI web modules.
Covers event-driven crawling, validation, enrichment, DB integration, and vector search.
"""

import pytest
from crawler import devourer
from knowledge import graph
import ai_web
import vector_db


@pytest.mark.asyncio
async def test_devourer_crawl_and_store():
    d = devourer.Devourer()
    await d.start()
    # Simulate event/crawl result
    event = {'id': 'evt1', 'data': {'url': 'http://test', 'content': 'abc'}}
    await d.handle_event(event)
    # Check DB insert
    result = await ai_web.get_by_id('evt1')
    assert result is not None
    await d.shutdown()


@pytest.mark.asyncio
async def test_knowledge_graph_enricher():
    kge = graph.KnowledgeGraphEnricher()
    json_data = {'nodes': [{'id': 1}], 'edges': []}
    proto = kge.json_to_proto(json_data)
    assert proto is not None
    # Test DB persistence
    await kge.persist_metadata({'id': 1, 'meta': 'x'})


@pytest.mark.asyncio
async def test_vector_db_similarity():
    vdb = vector_db.VectorDB()
    await vdb.add_vector('test', [0.1, 0.2, 0.3])
    result = await vdb.similarity_search([0.1, 0.2, 0.3])
    assert result
