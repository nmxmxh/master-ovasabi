CREATE TABLE IF NOT EXISTS service_campaign_broadcast_event (
    id SERIAL PRIMARY KEY,
    campaign_id VARCHAR NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    event_type VARCHAR NOT NULL,
    user_id VARCHAR,
    metadata JSONB,
    json_payload TEXT,
    binary_payload BYTEA,
    media_links JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_broadcast_event_campaign_id ON service_campaign_broadcast_event (campaign_id);
CREATE INDEX IF NOT EXISTS idx_campaign_broadcast_event_timestamp ON service_campaign_broadcast_event (timestamp);