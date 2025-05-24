-- 000010_align_service_product_main_fields.up.sql
-- Adds all missing fields to service_product_main for full proto/service alignment

ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS id UUID PRIMARY KEY DEFAULT uuid_generate_v4();
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS name TEXT;
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS type INT;
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS status INT;
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS tags TEXT[];
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS main_image_url TEXT;
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS gallery_image_urls TEXT[];
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS owner_id UUID;
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS campaign_id BIGINT; 