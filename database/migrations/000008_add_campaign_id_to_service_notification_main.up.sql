ALTER TABLE service_notification_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_notification_campaign_id ON service_notification_main (campaign_id); 