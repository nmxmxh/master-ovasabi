-- 000012_align_service_content_main_fields.up.sql
-- Adds all missing fields to service_content_main for full proto/service alignment

ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS id UUID PRIMARY KEY DEFAULT uuid_generate_v4();
ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS author_id UUID;
ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS title TEXT;
ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS body TEXT;
ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS comment_count INT DEFAULT 0;
ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS reaction_counts JSONB;
ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_content_main ADD COLUMN IF NOT EXISTS search_vector tsvector; 