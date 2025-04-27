-- Referral service table
CREATE TABLE IF NOT EXISTS service_referral (
    id SERIAL PRIMARY KEY,
    referrer_master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    campaign_id INTEGER REFERENCES service_campaign(id) ON DELETE SET NULL,
    device_hash VARCHAR(128),
    referral_code VARCHAR(64) NOT NULL UNIQUE,
    successful BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_service_referral_referrer_master_id ON service_referral(referrer_master_id);
CREATE INDEX IF NOT EXISTS idx_service_referral_campaign_id ON service_referral(campaign_id);
CREATE INDEX IF NOT EXISTS idx_service_referral_referral_code ON service_referral(referral_code);
CREATE INDEX IF NOT EXISTS idx_service_referral_created_at ON service_referral(created_at); 