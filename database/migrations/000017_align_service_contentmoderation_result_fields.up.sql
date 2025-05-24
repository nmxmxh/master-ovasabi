-- 000017_align_service_contentmoderation_result_fields.up.sql
-- Adds all missing fields to service_contentmoderation_result for full proto/service alignment

ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS content_id VARCHAR(64) PRIMARY KEY;
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS user_id VARCHAR(64);
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS status VARCHAR(32);
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS reason TEXT;
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS campaign_id BIGINT;
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS content_type TEXT;
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS content TEXT; 