ALTER TABLE service_scheduler_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_scheduler_campaign_id ON service_scheduler_main (campaign_id); 