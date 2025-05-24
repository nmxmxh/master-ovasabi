-- 000009_align_service_campaign_main_fields.up.sql
-- Adds all missing fields to service_campaign_main for full proto/service alignment

ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS id SERIAL PRIMARY KEY;
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS slug TEXT UNIQUE;
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS title TEXT;
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS ranking_formula TEXT;
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS start_date TIMESTAMPTZ;
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS end_date TIMESTAMPTZ;
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS status TEXT;
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS master_id BIGINT; 