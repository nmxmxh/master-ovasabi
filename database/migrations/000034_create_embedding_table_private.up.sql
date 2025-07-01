CREATE EXTENSION IF NOT EXISTS vector;
-- 000034_create_embedding_table_private.up.sql
-- Migration: Create private _embedding table for pgvector-backed semantic search (internal use only)

CREATE TABLE IF NOT EXISTS _embedding (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    campaign_id BIGINT NOT NULL,
    embedding vector(384) NOT NULL, -- pgvector extension required
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx__embedding_campaign_id ON _embedding (campaign_id);
CREATE INDEX idx__embedding_master_id ON _embedding (master_id);
CREATE INDEX idx__embedding_created_at ON _embedding (created_at);
-- For ANN search, use ivfflat or hnsw index if needed:
-- CREATE INDEX idx__embedding_vector_ann ON _embedding USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Trigger to auto-update updated_at on row changes
CREATE OR REPLACE FUNCTION update__embedding_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg__embedding_updated_at ON _embedding;
CREATE TRIGGER trg__embedding_updated_at
BEFORE UPDATE ON _embedding
FOR EACH ROW EXECUTE FUNCTION update__embedding_updated_at_column();
