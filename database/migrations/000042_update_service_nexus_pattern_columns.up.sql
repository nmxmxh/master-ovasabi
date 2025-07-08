-- Add missing columns for dual id pattern and full pattern registration
ALTER TABLE service_nexus_pattern ADD COLUMN IF NOT EXISTS version TEXT;
ALTER TABLE service_nexus_pattern ADD COLUMN IF NOT EXISTS origin TEXT;
ALTER TABLE service_nexus_pattern ADD COLUMN IF NOT EXISTS definition JSONB;
ALTER TABLE service_nexus_pattern ADD COLUMN IF NOT EXISTS created_by TEXT;
ALTER TABLE service_nexus_pattern ADD COLUMN IF NOT EXISTS campaign_id BIGINT;
