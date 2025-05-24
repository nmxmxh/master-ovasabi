ALTER TABLE service_search_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_search_campaign_id ON service_search_main (campaign_id); 