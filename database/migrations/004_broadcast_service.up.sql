-- Broadcast service table
CREATE TABLE IF NOT EXISTS service_broadcast (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    campaign_id INTEGER REFERENCES service_campaign(id) ON DELETE SET NULL,
    channel VARCHAR(32),
    subject VARCHAR(255),
    message TEXT,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scheduled_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_service_broadcast_master_id ON service_broadcast(master_id);
CREATE INDEX IF NOT EXISTS idx_service_broadcast_campaign_id ON service_broadcast(campaign_id);
CREATE INDEX IF NOT EXISTS idx_service_broadcast_channel ON service_broadcast(channel);
CREATE INDEX IF NOT EXISTS idx_service_broadcast_created_at ON service_broadcast(created_at); 