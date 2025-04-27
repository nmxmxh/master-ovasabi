-- I18n (translation) service table
CREATE TABLE IF NOT EXISTS service_i18n (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    campaign_id INTEGER REFERENCES service_campaign(id) ON DELETE SET NULL,
    key VARCHAR(255) NOT NULL,
    language VARCHAR(16) NOT NULL,
    value TEXT NOT NULL,
    context VARCHAR(128),
    translations JSONB, -- for plural forms or additional metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_service_i18n_master_id ON service_i18n(master_id);
CREATE INDEX IF NOT EXISTS idx_service_i18n_campaign_id ON service_i18n(campaign_id);
CREATE INDEX IF NOT EXISTS idx_service_i18n_key_lang ON service_i18n(key, language);
CREATE INDEX IF NOT EXISTS idx_service_i18n_created_at ON service_i18n(created_at); 