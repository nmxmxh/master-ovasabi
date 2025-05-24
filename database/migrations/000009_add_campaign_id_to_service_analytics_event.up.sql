ALTER TABLE service_analytics_event ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_analytics_campaign_id ON service_analytics_event (campaign_id); 