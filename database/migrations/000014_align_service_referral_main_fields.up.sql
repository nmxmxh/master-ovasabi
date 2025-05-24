-- 000014_align_service_referral_main_fields.up.sql
-- Adds all missing fields to service_referral_main for full proto/service alignment

ALTER TABLE service_referral_main ADD COLUMN IF NOT EXISTS id UUID PRIMARY KEY DEFAULT uuid_generate_v4();
ALTER TABLE service_referral_main ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_referral_main ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_referral_main ADD COLUMN IF NOT EXISTS user_id UUID;
ALTER TABLE service_referral_main ADD COLUMN IF NOT EXISTS referrer_master_uuid UUID;
ALTER TABLE service_referral_main ADD COLUMN IF NOT EXISTS referred_master_uuid UUID;
ALTER TABLE service_referral_main ADD COLUMN IF NOT EXISTS referral_code TEXT;
ALTER TABLE service_referral_main ADD COLUMN IF NOT EXISTS status INT;
ALTER TABLE service_referral_main ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE service_referral_main ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_referral_main ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now(); 