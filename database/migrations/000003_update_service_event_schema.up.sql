-- 000003_update_service_event_schema.up.sql
-- Adds missing columns to service_event for orchestration and metadata

ALTER TABLE service_event ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE service_event ADD COLUMN IF NOT EXISTS status TEXT;
ALTER TABLE service_event ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_event ADD COLUMN IF NOT EXISTS processed_at TIMESTAMPTZ;
ALTER TABLE service_event ADD COLUMN IF NOT EXISTS pattern_id UUID;
ALTER TABLE service_event ADD COLUMN IF NOT EXISTS step INT;
ALTER TABLE service_event ADD COLUMN IF NOT EXISTS retries INT;
ALTER TABLE service_event ADD COLUMN IF NOT EXISTS error TEXT; 