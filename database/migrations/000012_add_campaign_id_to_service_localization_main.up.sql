ALTER TABLE service_localization_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_localization_campaign_id ON service_localization_main (campaign_id); 