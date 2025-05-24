ALTER TABLE service_product_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_product_campaign_id ON service_product_main (campaign_id); 