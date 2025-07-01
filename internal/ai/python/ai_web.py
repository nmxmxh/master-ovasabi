"""
ai_web.py: Interface for the _web knowledge web/chain table
- Store and query AI inferences, entities, facts, and relationships
- Link to models and metadata for full referential intelligence
"""

import os
import uuid
from typing import List, Optional
from sqlalchemy.dialects.postgresql import UUID as PGUUID, JSONB
from sqlalchemy import MetaData, Table, Column, Text, TIMESTAMP, ARRAY, select as sa_select
from sqlalchemy.ext.asyncio import create_async_engine
from datetime import datetime


metadata_obj = MetaData()

web_table = Table(
    '_web', metadata_obj,
    Column('id', PGUUID(as_uuid=True), primary_key=True, default=uuid.uuid4),
    Column('node_type', Text, nullable=False),
    Column('node_data', JSONB, nullable=False),
    Column('parent_ids', ARRAY(PGUUID(as_uuid=True))),
    Column('edge_type', Text),
    Column('edge_data', JSONB),
    Column('model_hash', Text),
    Column('metadata_id', PGUUID(as_uuid=True)),
    Column('created_at', TIMESTAMP, default=datetime.utcnow),
)


ASYNC_PG_URL = os.getenv(
    "AI_PG_ASYNC_URL",
    "postgresql+asyncpg://user:password@localhost:5432/ovasabi"
)
async_engine = create_async_engine(ASYNC_PG_URL, echo=False)


class WebDB:
    def __init__(self):
        self.engine = async_engine

    async def insert_node(
        self,
        node_type: str,
        node_data: dict,
        parent_ids: Optional[List[uuid.UUID]] = None,
        edge_type: Optional[str] = None,
        edge_data: Optional[dict] = None,
        model_hash: Optional[str] = None,
        metadata_id: Optional[uuid.UUID] = None
    ) -> uuid.UUID:
        async with self.engine.begin() as conn:
            ins = web_table.insert().values(
                node_type=node_type,
                node_data=node_data,
                parent_ids=parent_ids or [],
                edge_type=edge_type,
                edge_data=edge_data,
                model_hash=model_hash,
                metadata_id=metadata_id,
                created_at=datetime.utcnow(),
            )
            result = await conn.execute(ins)
            inserted_id = result.inserted_primary_key[0] if result.inserted_primary_key else None
            return inserted_id

    async def get_nodes(
        self,
        node_type: Optional[str] = None,
        model_hash: Optional[str] = None,
        metadata_id: Optional[uuid.UUID] = None
    ) -> list:
        async with self.engine.begin() as conn:
            sel = sa_select(web_table)
            if node_type:
                sel = sel.where(web_table.c.node_type == node_type)
            if model_hash:
                sel = sel.where(web_table.c.model_hash == model_hash)
            if metadata_id:
                sel = sel.where(web_table.c.metadata_id == metadata_id)
            result = await conn.execute(sel)
            return result.fetchall()

    async def get_chain(self, start_id: uuid.UUID) -> list:
        # Recursively fetch nodes by parent_ids to build a chain
        async with self.engine.begin() as conn:
            chain = []
            to_visit = [start_id]
            visited = set()
            while to_visit:
                nid = to_visit.pop()
                if nid in visited:
                    continue
                sel = sa_select(web_table).where(web_table.c.id == nid)
                result = await conn.execute(sel)
                node = result.fetchone()
                if node:
                    chain.append(node)
                    visited.add(nid)
                    parent_ids = node.parent_ids or []
                    to_visit.extend(parent_ids)
            return chain
