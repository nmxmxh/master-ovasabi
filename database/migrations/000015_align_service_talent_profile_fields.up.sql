-- 000015_align_service_talent_profile_fields.up.sql
-- Adds all missing fields to service_talent_profile for full proto/service alignment

ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS id UUID PRIMARY KEY DEFAULT uuid_generate_v4();
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS user_id UUID;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS name TEXT;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS bio TEXT;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS skills TEXT[];
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS experience JSONB;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS education JSONB;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS rating NUMERIC(3,2);
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS status INT;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS search_vector tsvector;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS display_name TEXT;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS tags TEXT[];
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS location TEXT;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS campaign_id BIGINT; 