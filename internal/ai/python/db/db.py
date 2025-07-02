"""
db.py: Unified Postgres connection and ORM helpers for AI enrichment
- Connects to _ai and _metadata_master tables
- Provides insert, update, query, and enrichment link logic
- Uses SQLAlchemy for robust, production-grade access
"""


import os
import uuid
from datetime import datetime
from sqlalchemy import (
    create_engine, MetaData, Table, Column, Integer, Text, TIMESTAMP, func, select as sa_select
)
from sqlalchemy.dialects.postgresql import UUID, BYTEA, JSONB
from sqlalchemy.ext.asyncio import create_async_engine


metadata_obj = MetaData()

ai_table = Table(
    '_ai', metadata_obj,
    Column('id', Integer, primary_key=True),
    Column('type', Text, nullable=False),
    Column('data', BYTEA, nullable=False),
    Column('meta', JSONB),
    Column('hash', Text, nullable=False, unique=True, index=True),
    Column('version', Text),
    Column('parent_hash', Text, index=True),
    Column('created_at', TIMESTAMP, server_default=func.now()),
    Column('updated_at', TIMESTAMP, server_default=func.now(), onupdate=func.now()),
)

metadata_master_table = Table(
    '_metadata_master', metadata_obj,
    Column('id', UUID(as_uuid=True), primary_key=True, default=uuid.uuid4),
    Column('entity_id', UUID(as_uuid=True)),
    Column('entity_type', Text, nullable=False),
    Column('category', Text, nullable=False),
    Column('environment', Text, nullable=False),
    Column('role', Text, nullable=False),
    Column('policy', JSONB, default={}),
    Column('metadata', JSONB, nullable=False),
    Column('lineage', JSONB, default={}),
    Column('created_at', TIMESTAMP, server_default=func.now()),
    Column('updated_at', TIMESTAMP, server_default=func.now(), onupdate=func.now()),
    Column('expires_at', TIMESTAMP),
    Column('deleted_at', TIMESTAMP),
)


def get_pg_url():
    url = os.getenv("DATABASE_URL")
    if url:
        # Ensure correct prefix for SQLAlchemy
        if url.startswith("postgres://"):
            url = url.replace("postgres://", "postgresql://", 1)
        return url
    user = os.getenv("POSTGRES_USER", "postgres")
    password = os.getenv("POSTGRES_PASSWORD", "postgres")
    db = os.getenv("POSTGRES_NAME", "master_ovasabi")
    host = os.getenv("DB_HOST", "db")
    port = os.getenv("DB_PORT", "5432")
    return f"postgresql://{user}:{password}@{host}:{port}/{db}"


def get_async_pg_url():
    url = os.getenv("DATABASE_URL")
    if url:
        # Ensure correct prefix for asyncpg
        if url.startswith("postgres://"):
            url = url.replace("postgres://", "postgresql://", 1)
        url = url.replace("postgresql://", "postgresql+asyncpg://", 1)
        return url
    user = os.getenv("POSTGRES_USER", "postgres")
    password = os.getenv("POSTGRES_PASSWORD", "postgres")
    db = os.getenv("POSTGRES_NAME", "master_ovasabi")
    host = os.getenv("DB_HOST", "db")
    port = os.getenv("DB_PORT", "5432")
    return f"postgresql+asyncpg://{user}:{password}@{host}:{port}/{db}"


PG_URL = os.getenv("AI_PG_URL", get_pg_url())
engine = create_engine(PG_URL)


class EnrichmentDB:
    def __init__(self):
        self.engine = engine

    def insert_model(self, type_: str, data: bytes, meta: dict, hash_: str, version: str = None, parent_hash: str = None):
        with self.engine.begin() as conn:
            ins = ai_table.insert().values(
                type=type_, data=data, meta=meta, hash=hash_, version=version, parent_hash=parent_hash
            )
            result = conn.execute(ins)
            return result.inserted_primary_key[0]

    def insert_metadata(self, entity_type: str, category: str, environment: str, role: str, metadata: dict, policy: dict = {}, lineage: dict = {}, entity_id: uuid.UUID = None):
        with self.engine.begin() as conn:
            ins = metadata_master_table.insert().values(
                entity_id=entity_id, entity_type=entity_type, category=category, environment=environment, role=role,
                metadata=metadata, policy=policy, lineage=lineage
            )
            result = conn.execute(ins)
            return result.inserted_primary_key[0]

    def link_model_to_metadata(self, model_hash: str, metadata_id: uuid.UUID):
        with self.engine.begin() as conn:
            sel = sa_select(metadata_master_table).where(metadata_master_table.c.id == metadata_id)
            meta = conn.execute(sel).fetchone()
            if meta:
                lineage = dict(meta.lineage or {})
                lineage.setdefault("model_hashes", []).append(model_hash)
                upd = metadata_master_table.update().where(metadata_master_table.c.id == metadata_id).values(lineage=lineage)
                conn.execute(upd)
                return True
            return False

    def get_latest_model(self, type_: str):
        with self.engine.begin() as conn:
            sel = sa_select(ai_table).where(ai_table.c.type == type_).order_by(ai_table.c.created_at.desc())
            result = conn.execute(sel).fetchone()
            return result

    def get_metadata(self, **filters):
        with self.engine.begin() as conn:
            sel = sa_select(metadata_master_table).filter_by(**filters)
            result = conn.execute(sel).fetchone()
            return result


ASYNC_PG_URL = os.getenv("AI_PG_ASYNC_URL", get_async_pg_url())
async_engine = create_async_engine(ASYNC_PG_URL, echo=False)


class AsyncEnrichmentDB:
    def __init__(self):
        self.engine = async_engine

    async def insert_model(self, type_: str, data: bytes, meta: dict, hash_: str, version: str = None, parent_hash: str = None):
        async with self.engine.begin() as conn:
            ins = ai_table.insert().values(
                type=type_, data=data, meta=meta, hash=hash_, version=version, parent_hash=parent_hash
            )
            result = await conn.execute(ins)
            return result.inserted_primary_key[0] if result.inserted_primary_key else None

    async def insert_metadata(self, entity_type: str, category: str, environment: str, role: str, metadata: dict, policy: dict = {}, lineage: dict = {}, entity_id: uuid.UUID = None):
        async with self.engine.begin() as conn:
            ins = metadata_master_table.insert().values(
                entity_id=entity_id, entity_type=entity_type, category=category, environment=environment, role=role,
                metadata=metadata, policy=policy, lineage=lineage
            )
            result = await conn.execute(ins)
            return result.inserted_primary_key[0] if result.inserted_primary_key else None

    async def link_model_to_metadata(self, model_hash: str, metadata_id: uuid.UUID):
        async with self.engine.begin() as conn:
            sel = sa_select(metadata_master_table).where(metadata_master_table.c.id == metadata_id)
            result = await conn.execute(sel)
            meta = result.fetchone()
            if meta:
                lineage = dict(meta.lineage or {})
                lineage.setdefault("model_hashes", []).append(model_hash)
                upd = metadata_master_table.update().where(metadata_master_table.c.id == metadata_id).values(lineage=lineage)
                await conn.execute(upd)
                return True
            return False

    async def get_latest_model(self, type_: str):
        async with self.engine.begin() as conn:
            sel = sa_select(ai_table).where(ai_table.c.type == type_).order_by(ai_table.c.created_at.desc())
            result = await conn.execute(sel)
            return result.fetchone()

    async def get_metadata(self, **filters):
        async with self.engine.begin() as conn:
            sel = sa_select(metadata_master_table).filter_by(**filters)
            result = await conn.execute(sel)
            return result.fetchone()

    async def get_ai_relevant_metadata(self, min_confidence: float = 0.0, categories: list = None, require_embedding: bool = False) -> list:
        async with self.engine.begin() as conn:
            sel = sa_select(metadata_master_table)
            result = await conn.execute(sel)
            all_meta = result.fetchall()
            results = []
            for m in all_meta:
                md = m.metadata or {}
                ai_conf = md.get('ai_confidence', 0.0)
                embedding_id = md.get('embedding_id')
                cats = set(md.get('categories', []))
                if ai_conf < min_confidence:
                    continue
                if categories and not cats.intersection(categories):
                    continue
                if require_embedding and not embedding_id:
                    continue
                results.append(m)
            return results

    async def update_ai_metadata(self, metadata_id: uuid.UUID, ai_fields: dict):
        async with self.engine.begin() as conn:
            sel = sa_select(metadata_master_table).where(metadata_master_table.c.id == metadata_id)
            result = await conn.execute(sel)
            meta = result.fetchone()
            if meta:
                md = dict(meta.metadata or {})
                md.update(ai_fields)
                upd = metadata_master_table.update().where(metadata_master_table.c.id == metadata_id).values(metadata=md)
                await conn.execute(upd)
                return True
            return False

    async def discard_stale_knowledge(self, reference_field: str = "ai_confidence", prefer_higher: bool = True):
        async with self.engine.begin() as conn:
            sel = sa_select(metadata_master_table)
            result = await conn.execute(sel)
            all_meta = result.fetchall()
            grouped = {}
            for m in all_meta:
                md = m.metadata or {}
                val = md.get(reference_field, None)
                key = (m.entity_type, m.entity_id)
                if key not in grouped or (
                    prefer_higher and val is not None and (grouped[key][1] is None or val > grouped[key][1])
                ):
                    grouped[key] = (m, val)
            to_delete = [m for m in all_meta if (m.entity_type, m.entity_id) not in grouped or m.id != grouped[(m.entity_type, m.entity_id)][0].id]
            for m in to_delete:
                upd = metadata_master_table.update().where(metadata_master_table.c.id == m.id).values(deleted_at=datetime.utcnow())
                await conn.execute(upd)
            # Repeat for models if needed (e.g., by ai_confidence in meta)
            sel = sa_select(ai_table)
            result = await conn.execute(sel)
            all_models = result.fetchall()
            grouped = {}
            for m in all_models:
                meta = m.meta or {}
                val = meta.get(reference_field, None)
                key = (m.type, m.parent_hash)
                if key not in grouped or (
                    prefer_higher and val is not None and (grouped[key][1] is None or val > grouped[key][1])
                ):
                    grouped[key] = (m, val)
            to_delete = [m for m in all_models if (m.type, m.parent_hash) not in grouped or m.id != grouped[(m.type, m.parent_hash)][0].id]
            for m in to_delete:
                upd = ai_table.update().where(ai_table.c.id == m.id).values(data=b'', meta={}, version=None)
                await conn.execute(upd)

    async def batch_insert_models(self, models: list):
        async with self.engine.begin() as conn:
            await conn.execute(ai_table.insert(), models)

    async def batch_insert_metadata(self, metadatas: list):
        async with self.engine.begin() as conn:
            await conn.execute(metadata_master_table.insert(), metadatas)

    async def auto_update_metadata(self, match_filters: dict, update_fields: dict):
        async with self.engine.begin() as conn:
            upd = metadata_master_table.update().filter_by(**match_filters).values(**update_fields)
            await conn.execute(upd)
