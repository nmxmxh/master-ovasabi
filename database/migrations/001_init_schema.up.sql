-- MASTER TABLE: Core entity for all services
CREATE TABLE master (
    id SERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),
    name VARCHAR(255),
    type VARCHAR(64) NOT NULL, -- e.g., user, campaign, etc.
    description TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE UNIQUE INDEX idx_master_uuid ON master(uuid);
CREATE INDEX idx_master_type ON master(type);

-- USER SERVICE TABLE
CREATE TABLE service_user (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_service_user_master_id ON service_user(master_id);

-- CAMPAIGN SERVICE TABLE
CREATE TABLE service_campaign (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    slug VARCHAR(128) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    ranking_formula TEXT,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_service_campaign_master_id ON service_campaign(master_id);
CREATE INDEX idx_service_campaign_slug ON service_campaign(slug);
CREATE INDEX idx_service_campaign_created_at ON service_campaign(created_at);

-- NOTIFICATION SERVICE TABLE
CREATE TABLE service_notification (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL,
    type VARCHAR(32) NOT NULL,
    title VARCHAR(255),
    content TEXT,
    status VARCHAR(32) NOT NULL,
    metadata JSONB,
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_service_notification_master_id ON service_notification(master_id);
CREATE INDEX idx_service_notification_user_id ON service_notification(user_id);
CREATE INDEX idx_service_notification_status ON service_notification(status);

-- BROADCAST SERVICE TABLE
CREATE TABLE service_broadcast (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    type VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_service_broadcast_master_id ON service_broadcast(master_id);
CREATE INDEX idx_service_broadcast_status ON service_broadcast(status);

-- QUOTE SERVICE TABLE
CREATE TABLE service_quote (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    symbol VARCHAR(32) NOT NULL,
    price NUMERIC(18,8) NOT NULL,
    volume NUMERIC(18,8),
    metadata JSONB,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_service_quote_master_id ON service_quote(master_id);
CREATE INDEX idx_service_quote_symbol ON service_quote(symbol);
CREATE INDEX idx_service_quote_timestamp ON service_quote(timestamp);

-- I18N SERVICE TABLE
CREATE TABLE service_i18n (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    key VARCHAR(128) NOT NULL,
    locale VARCHAR(16) NOT NULL,
    value TEXT NOT NULL,
    description TEXT,
    tags TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_service_i18n_master_id ON service_i18n(master_id);
CREATE INDEX idx_service_i18n_key_locale ON service_i18n(key, locale);

-- REFERRAL SERVICE TABLE
CREATE TABLE service_referral (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    referrer_id INTEGER,
    referee_id INTEGER,
    referral_code VARCHAR(64) UNIQUE NOT NULL,
    status VARCHAR(32),
    reward_claimed BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_service_referral_master_id ON service_referral(master_id);
CREATE INDEX idx_service_referral_code ON service_referral(referral_code);

-- EVENT TABLE (PARTITIONED BY MONTH)
CREATE TABLE service_event (
    id SERIAL,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    event_type VARCHAR(128) NOT NULL,
    payload JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, occurred_at)
) PARTITION BY RANGE (occurred_at);

-- Example partitions
CREATE TABLE service_event_y2024m01 PARTITION OF service_event
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
CREATE TABLE service_event_y2024m02 PARTITION OF service_event
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- INDEXES FOR JSONB FIELDS
CREATE INDEX idx_service_user_metadata_gin ON service_user USING GIN (metadata);
CREATE INDEX idx_service_campaign_metadata_gin ON service_campaign USING GIN (metadata);
CREATE INDEX idx_service_notification_metadata_gin ON service_notification USING GIN (metadata);
CREATE INDEX idx_service_broadcast_metadata_gin ON service_broadcast USING GIN (metadata);
CREATE INDEX idx_service_quote_metadata_gin ON service_quote USING GIN (metadata);
CREATE INDEX idx_service_referral_metadata_gin ON service_referral USING GIN (metadata);

-- TRIGGERS FOR UPDATED_AT
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_master_updated_at
    BEFORE UPDATE ON master
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_service_user_updated_at
    BEFORE UPDATE ON service_user
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_service_campaign_updated_at
    BEFORE UPDATE ON service_campaign
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_service_notification_updated_at
    BEFORE UPDATE ON service_notification
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_service_broadcast_updated_at
    BEFORE UPDATE ON service_broadcast
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_service_quote_updated_at
    BEFORE UPDATE ON service_quote
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_service_i18n_updated_at
    BEFORE UPDATE ON service_i18n
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_service_referral_updated_at
    BEFORE UPDATE ON service_referral
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column(); 