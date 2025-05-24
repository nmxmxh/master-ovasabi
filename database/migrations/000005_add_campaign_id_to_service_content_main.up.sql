ALTER TABLE service_content_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_content_campaign_id ON service_content_main (campaign_id);
CREATE INDEX idx_content_campaign_fts ON service_content_main USING GIN (campaign_id, search_vector); 