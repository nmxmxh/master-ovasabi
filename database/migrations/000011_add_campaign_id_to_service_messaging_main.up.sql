ALTER TABLE service_messaging_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_messaging_campaign_id ON service_messaging_main (campaign_id); 