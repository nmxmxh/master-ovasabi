-- 000021_create_service_event_table.up.sql
-- Migration to create the service_event table for event orchestration and metadata-driven event storage

CREATE TABLE IF NOT EXISTS service_event (
    id UUID PRIMARY KEY,
    master_id BIGINT NOT NULL,
    entity_type VARCHAR(64) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    payload JSONB NOT NULL,
    metadata JSONB,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    pattern_id VARCHAR(128),
    step VARCHAR(128),
    retries INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    CONSTRAINT fk_master_id FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_service_event_master_id ON service_event(master_id);
CREATE INDEX IF NOT EXISTS idx_service_event_entity_type ON service_event(entity_type);
CREATE INDEX IF NOT EXISTS idx_service_event_status ON service_event(status);
CREATE INDEX IF NOT EXISTS idx_service_event_pattern_id ON service_event(pattern_id);
CREATE INDEX IF NOT EXISTS idx_service_event_created_at ON service_event(created_at); 