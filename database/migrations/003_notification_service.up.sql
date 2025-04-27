-- Notification service table
CREATE TABLE IF NOT EXISTS service_notification (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    campaign_id INTEGER REFERENCES service_campaign(id) ON DELETE SET NULL,
    channel VARCHAR(32) NOT NULL, -- e.g., email, sms, push
    title VARCHAR(255),
    body TEXT,
    payload JSONB,
    is_read BOOLEAN DEFAULT FALSE,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_service_notification_master_id ON service_notification(master_id);
CREATE INDEX IF NOT EXISTS idx_service_notification_campaign_id ON service_notification(campaign_id);
CREATE INDEX IF NOT EXISTS idx_service_notification_channel ON service_notification(channel);
CREATE INDEX IF NOT EXISTS idx_service_notification_created_at ON service_notification(created_at); 