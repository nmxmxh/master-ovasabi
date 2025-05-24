ALTER TABLE service_nexus_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_nexus_campaign_id ON service_nexus_main (campaign_id); 