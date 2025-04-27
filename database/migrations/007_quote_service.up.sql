-- Quote service table
CREATE TABLE IF NOT EXISTS service_quote (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    campaign_id INTEGER REFERENCES service_campaign(id) ON DELETE SET NULL,
    description TEXT,
    author VARCHAR(255),
    metadata JSONB,
    amount NUMERIC(18,2),
    currency VARCHAR(16),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_service_quote_master_id ON service_quote(master_id);
CREATE INDEX IF NOT EXISTS idx_service_quote_campaign_id ON service_quote(campaign_id);
CREATE INDEX IF NOT EXISTS idx_service_quote_author ON service_quote(author);
CREATE INDEX IF NOT EXISTS idx_service_quote_created_at ON service_quote(created_at); 