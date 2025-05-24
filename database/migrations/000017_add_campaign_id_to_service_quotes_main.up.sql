ALTER TABLE service_quotes_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_quotes_campaign_id ON service_quotes_main (campaign_id); 