-- 000013_align_service_notification_main_fields.up.sql
-- Adds all missing fields to service_notification_main for full proto/service alignment

ALTER TABLE service_notification_main ADD COLUMN IF NOT EXISTS id UUID PRIMARY KEY DEFAULT uuid_generate_v4();
ALTER TABLE service_notification_main ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_notification_main ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_notification_main ADD COLUMN IF NOT EXISTS user_id UUID;
ALTER TABLE service_notification_main ADD COLUMN IF NOT EXISTS channel TEXT;
ALTER TABLE service_notification_main ADD COLUMN IF NOT EXISTS template_id TEXT;
ALTER TABLE service_notification_main ADD COLUMN IF NOT EXISTS payload JSONB;
ALTER TABLE service_notification_main ADD COLUMN IF NOT EXISTS status INT;
ALTER TABLE service_notification_main ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE service_notification_main ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_notification_main ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now(); 