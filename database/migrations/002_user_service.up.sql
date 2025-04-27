-- User service table
CREATE TABLE IF NOT EXISTS service_user (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    referral_code VARCHAR(64),
    referred_by VARCHAR(64),
    device_hash VARCHAR(128),
    location VARCHAR(128),
    profile JSONB, -- stores UserProfile and custom fields
    status VARCHAR(32) DEFAULT 'active',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_service_user_master_id ON service_user(master_id);
CREATE INDEX IF NOT EXISTS idx_service_user_email ON service_user(email);
CREATE INDEX IF NOT EXISTS idx_service_user_referral_code ON service_user(referral_code);
CREATE INDEX IF NOT EXISTS idx_service_user_created_at ON service_user(created_at); 