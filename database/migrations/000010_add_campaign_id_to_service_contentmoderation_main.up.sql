ALTER TABLE service_contentmoderation_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_contentmoderation_campaign_id ON service_contentmoderation_main (campaign_id); 