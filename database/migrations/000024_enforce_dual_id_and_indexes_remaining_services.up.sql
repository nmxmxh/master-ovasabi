-- 000024_enforce_dual_id_and_indexes_remaining_services.up.sql
-- Enforce dual-ID (master_id, master_uuid) and index standards for remaining services

-- Commerce Service
ALTER TABLE service_commerce_quote ADD COLUMN IF NOT EXISTS master_uuid UUID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_commerce_quote_uuid ON service_commerce_quote(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_commerce_quote_master_id ON service_commerce_quote(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_quote_metadata_gin ON service_commerce_quote USING gin (metadata);

ALTER TABLE service_commerce_order ADD COLUMN IF NOT EXISTS master_uuid UUID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_commerce_order_uuid ON service_commerce_order(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_master_id ON service_commerce_order(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_metadata_gin ON service_commerce_order USING gin (metadata);

ALTER TABLE service_commerce_payment ADD COLUMN IF NOT EXISTS master_uuid UUID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_commerce_payment_uuid ON service_commerce_payment(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_commerce_payment_master_id ON service_commerce_payment(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_payment_metadata_gin ON service_commerce_payment USING gin (metadata);

ALTER TABLE service_commerce_transaction ADD COLUMN IF NOT EXISTS master_uuid UUID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_commerce_transaction_uuid ON service_commerce_transaction(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_commerce_transaction_master_id ON service_commerce_transaction(master_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_transaction_metadata_gin ON service_commerce_transaction USING gin (metadata);

-- Analytics Service
ALTER TABLE service_analytics_event ADD COLUMN IF NOT EXISTS master_uuid UUID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_analytics_event_uuid ON service_analytics_event(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_analytics_event_master_id ON service_analytics_event(master_id);
CREATE INDEX IF NOT EXISTS idx_service_analytics_event_metadata_gin ON service_analytics_event USING gin (metadata);

-- Talent Service
ALTER TABLE service_talent_profile ADD COLUMN IF NOT EXISTS master_uuid UUID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_talent_profile_uuid ON service_talent_profile(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_talent_profile_master_id ON service_talent_profile(master_id);
CREATE INDEX IF NOT EXISTS idx_service_talent_profile_metadata_gin ON service_talent_profile USING gin (metadata);

-- Content Moderation Service
ALTER TABLE service_contentmoderation_result ADD COLUMN IF NOT EXISTS master_uuid UUID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_contentmoderation_result_uuid ON service_contentmoderation_result(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_contentmoderation_result_master_id ON service_contentmoderation_result(master_id);
CREATE INDEX IF NOT EXISTS idx_service_contentmoderation_result_metadata_gin ON service_contentmoderation_result USING gin (metadata);

-- Search Service
ALTER TABLE service_search_index ADD COLUMN IF NOT EXISTS master_uuid UUID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_search_index_uuid ON service_search_index(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_search_index_master_id ON service_search_index(master_id);
CREATE INDEX IF NOT EXISTS idx_service_search_index_metadata_gin ON service_search_index USING gin (metadata); 