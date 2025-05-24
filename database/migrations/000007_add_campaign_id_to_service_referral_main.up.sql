ALTER TABLE service_referral_main ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_referral_campaign_id ON service_referral_main (campaign_id); 