-- 000016_align_service_commerce_order_fields.up.sql
-- Adds all missing fields to service_commerce_order for full proto/service alignment

ALTER TABLE service_commerce_order ADD COLUMN IF NOT EXISTS order_id TEXT PRIMARY KEY;
ALTER TABLE service_commerce_order ADD COLUMN IF NOT EXISTS master_id BIGINT;
ALTER TABLE service_commerce_order ADD COLUMN IF NOT EXISTS master_uuid UUID;
ALTER TABLE service_commerce_order ADD COLUMN IF NOT EXISTS user_id TEXT;
ALTER TABLE service_commerce_order ADD COLUMN IF NOT EXISTS total NUMERIC(20,8);
ALTER TABLE service_commerce_order ADD COLUMN IF NOT EXISTS currency TEXT;
ALTER TABLE service_commerce_order ADD COLUMN IF NOT EXISTS status INT;
ALTER TABLE service_commerce_order ADD COLUMN IF NOT EXISTS metadata JSONB;
ALTER TABLE service_commerce_order ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE service_commerce_order ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now(); 