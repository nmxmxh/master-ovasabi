-- 000027_create_metadata_master_private.up.sql
-- Create the private _metadata_master table for orchestration, audit, and policy-driven metadata
-- This table is internal-only, supports ephemeral/TTL records, and is not exposed to public APIs

CREATE TABLE IF NOT EXISTS _metadata_master (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID,
    entity_type TEXT NOT NULL,
    category TEXT NOT NULL,
    environment TEXT NOT NULL,
    role TEXT NOT NULL,
    policy JSONB DEFAULT '{}',
    metadata JSONB NOT NULL CHECK (octet_length(metadata) <= 65536), -- Canonical metadata, max 64KB
    lineage JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

-- GIN index for efficient metadata queries
CREATE INDEX IF NOT EXISTS idx__metadata_master_metadata_gin ON _metadata_master USING gin (metadata);

-- Indexes for policy-driven and orchestration queries
CREATE INDEX IF NOT EXISTS idx__metadata_master_entity_id ON _metadata_master(entity_id);
CREATE INDEX IF NOT EXISTS idx__metadata_master_entity_type ON _metadata_master(entity_type);
CREATE INDEX IF NOT EXISTS idx__metadata_master_category ON _metadata_master(category);
CREATE INDEX IF NOT EXISTS idx__metadata_master_environment ON _metadata_master(environment);
CREATE INDEX IF NOT EXISTS idx__metadata_master_role ON _metadata_master(role);
CREATE INDEX IF NOT EXISTS idx__metadata_master_expires_at ON _metadata_master(expires_at);

-- Trigger to auto-update updated_at on row changes
CREATE OR REPLACE FUNCTION update__metadata_master_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg__metadata_master_updated_at ON _metadata_master;
CREATE TRIGGER trg__metadata_master_updated_at
BEFORE UPDATE ON _metadata_master
FOR EACH ROW EXECUTE FUNCTION update__metadata_master_updated_at_column(); 