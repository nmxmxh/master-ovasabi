-- Master table: core entity for all services
CREATE TABLE IF NOT EXISTS master (
    id SERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    type VARCHAR(64) NOT NULL, -- e.g., user, campaign, etc.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_master_uuid ON master(uuid);
CREATE INDEX IF NOT EXISTS idx_master_type ON master(type);

-- Centralized event logging table
CREATE TABLE IF NOT EXISTS service_event (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    event_type VARCHAR(128) NOT NULL,
    payload JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_service_event_master_id ON service_event(master_id);
CREATE INDEX IF NOT EXISTS idx_service_event_event_type ON service_event(event_type);
CREATE INDEX IF NOT EXISTS idx_service_event_occurred_at ON service_event(occurred_at);

-- Campaign table (renamed to service_campaign)
CREATE TABLE IF NOT EXISTS service_campaign (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    slug VARCHAR(128) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    ranking_formula TEXT,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_service_campaign_master_id ON service_campaign(master_id);
CREATE INDEX IF NOT EXISTS idx_service_campaign_slug ON service_campaign(slug);
CREATE INDEX IF NOT EXISTS idx_service_campaign_created_at ON service_campaign(created_at); 