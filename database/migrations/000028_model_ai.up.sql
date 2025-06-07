-- Migration: Create private _ai table for AI models and updates
CREATE TABLE IF NOT EXISTS _ai (
    id SERIAL PRIMARY KEY,
    type TEXT NOT NULL,                -- 'model' or 'update'
    data BYTEA NOT NULL,               -- Model weights, parameters, or update data
    meta JSONB,                        -- Arbitrary metadata (versioning, audit, etc.)
    hash TEXT NOT NULL,                -- Unique, tamper-evident identifier
    version TEXT,                      -- Model version string
    parent_hash TEXT,                  -- (Optional) for lineage/ancestry tracking
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx__ai_hash ON _ai(hash);
CREATE INDEX IF NOT EXISTS idx__ai_type ON _ai(type);
CREATE INDEX IF NOT EXISTS idx__ai_parent_hash ON _ai(parent_hash); 