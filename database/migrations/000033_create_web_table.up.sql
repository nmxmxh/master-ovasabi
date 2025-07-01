-- 000033_create_web_table.up.sql
-- Migration: Create private _web table for AI knowledge web/chain
CREATE TABLE IF NOT EXISTS _web (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_type TEXT NOT NULL,                -- e.g., 'inference', 'entity', 'fact', 'pattern'
    node_data JSONB NOT NULL,               -- Arbitrary node data (inference, entity, etc.)
    parent_ids UUID[],                      -- For chains/lineage
    edge_type TEXT,                         -- Relationship type
    edge_data JSONB,                        -- Arbitrary edge metadata
    model_hash TEXT,                        -- Link to _ai table
    metadata_id UUID,                       -- Link to _metadata_master
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx__web_model_hash ON _web(model_hash);
CREATE INDEX IF NOT EXISTS idx__web_metadata_id ON _web(metadata_id);
CREATE INDEX IF NOT EXISTS idx__web_node_type ON _web(node_type);
