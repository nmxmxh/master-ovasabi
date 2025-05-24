-- 000018_align_service_messaging_main_fields.up.sql
-- Adds all missing fields to service_messaging_main for full proto/service alignment

ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS id UUID PRIMARY KEY DEFAULT uuid_generate_v4();
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS thread_id UUID;
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS conversation_id UUID;
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS chat_group_id UUID;
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS sender_id UUID;
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS recipient_id UUID;
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS message TEXT;
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS status INT;
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS campaign_id BIGINT; 