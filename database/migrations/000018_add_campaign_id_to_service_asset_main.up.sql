ALTER TABLE service_asset_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_asset_campaign_id ON service_asset_main (campaign_id); 