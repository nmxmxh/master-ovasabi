-- 000005_align_all_tables_with_proto_and_service.up.sql
-- Adds all missing fields to tables for full proto/service alignment

-- service_contentmoderation_result
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS campaign_id BIGINT;
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS content_type TEXT;
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS content TEXT;

-- service_commerce_transaction
ALTER TABLE service_commerce_transaction ADD COLUMN IF NOT EXISTS user_id TEXT;
ALTER TABLE service_commerce_transaction ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_commerce_transaction ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_commerce_transaction ADD COLUMN IF NOT EXISTS campaign_id BIGINT;

-- service_commerce_balance
ALTER TABLE service_commerce_balance ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_commerce_balance ADD COLUMN IF NOT EXISTS master_uuid UUID;

-- service_talent_profile
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS display_name TEXT;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS tags TEXT;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS location TEXT;
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS campaign_id BIGINT;

-- service_talent_booking
ALTER TABLE service_talent_booking ADD COLUMN IF NOT EXISTS notes TEXT;
ALTER TABLE service_talent_booking ADD COLUMN IF NOT EXISTS campaign_id BIGINT;
ALTER TABLE service_talent_booking ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ;

-- service_messaging_main
ALTER TABLE service_messaging_main ADD COLUMN IF NOT EXISTS campaign_id BIGINT;

-- service_product_main
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS main_image_url TEXT;
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS gallery_image_urls TEXT;
ALTER TABLE service_product_main ADD COLUMN IF NOT EXISTS owner_id UUID; 