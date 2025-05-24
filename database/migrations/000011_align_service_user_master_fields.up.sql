-- 000011_align_service_user_master_fields.up.sql
-- Adds all missing fields to service_user_master for full proto/service alignment

ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS id UUID PRIMARY KEY DEFAULT uuid_generate_v4();
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS username TEXT;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS email TEXT;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS referral_code TEXT;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS referred_by TEXT;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS device_hash TEXT;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS location TEXT;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS password_hash TEXT;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS profile JSONB;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS roles TEXT[];
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS status INT;
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS tags TEXT[];
ALTER TABLE service_user_master ADD COLUMN IF NOT EXISTS external_ids JSONB; 