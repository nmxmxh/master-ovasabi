ALTER TABLE service_broadcast_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_broadcast_campaign_id ON service_broadcast_main (campaign_id); 