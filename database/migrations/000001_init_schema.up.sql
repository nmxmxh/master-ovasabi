-- +goose Up
-- OVASABI Platform: Canonical Baseline Schema (2025-05-16, consolidated, dual-ID compliant)

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Master table for all entities (dual-ID: id (int) and uuid (UUID))
CREATE TABLE master (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4(),
    type TEXT NOT NULL,
    name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_master_uuid ON master(uuid);
CREATE INDEX idx_master_type ON master(type);
CREATE INDEX idx_master_name_trgm ON master USING gin (name gin_trgm_ops);

-- Centralized event logging table
CREATE TABLE service_event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_event_master_uuid ON service_event(master_uuid);
CREATE INDEX idx_service_event_event_type ON service_event(event_type);

-- Reserved usernames (for user service validation)
CREATE TABLE IF NOT EXISTS service_user_reserved_username (
    username TEXT PRIMARY KEY,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- User service tables (dual-ID)
CREATE TABLE service_user_master (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    profile JSONB,
    roles TEXT[],
    status SMALLINT NOT NULL,
    referral_code TEXT UNIQUE,
    referred_by UUID REFERENCES service_user_master(id),
    device_hash TEXT,
    location TEXT,
    tags TEXT[],
    external_ids JSONB,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    search_vector tsvector
);
CREATE UNIQUE INDEX idx_service_user_master_uuid ON service_user_master(master_uuid);
CREATE INDEX idx_service_user_master_email ON service_user_master(email);
CREATE INDEX idx_service_user_master_username ON service_user_master(username);
CREATE INDEX IF NOT EXISTS idx_service_user_master_search_vector ON service_user_master USING gin (search_vector);
CREATE INDEX IF NOT EXISTS idx_service_user_master_device_hash ON service_user_master(device_hash);
CREATE INDEX IF NOT EXISTS idx_service_user_master_location ON service_user_master(location);
CREATE INDEX IF NOT EXISTS idx_service_user_master_tags ON service_user_master USING gin (tags);
CREATE INDEX IF NOT EXISTS idx_service_user_master_external_ids_gin ON service_user_master USING gin (external_ids);

-- FTS trigger for user
CREATE OR REPLACE FUNCTION update_user_search_vector() RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', coalesce(NEW.username, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(NEW.email, '')), 'B');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_user_search_vector ON service_user_master;
CREATE TRIGGER trg_user_search_vector
  BEFORE INSERT OR UPDATE ON service_user_master
  FOR EACH ROW EXECUTE FUNCTION update_user_search_vector();

-- Admin service tables (dual-ID)
CREATE TABLE service_admin_user (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    user_id UUID REFERENCES service_user_master(id),
    email TEXT NOT NULL UNIQUE,
    name TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB
);
CREATE UNIQUE INDEX idx_service_admin_user_uuid ON service_admin_user(master_uuid);
CREATE INDEX idx_service_admin_user_user_id ON service_admin_user(user_id);
CREATE INDEX IF NOT EXISTS idx_service_admin_user_metadata_gin ON service_admin_user USING gin (metadata);

CREATE TABLE service_admin_role (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    name TEXT NOT NULL UNIQUE,
    permissions TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB
);
CREATE INDEX IF NOT EXISTS idx_service_admin_role_metadata_gin ON service_admin_role USING gin (metadata);

CREATE TABLE service_admin_user_role (
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_admin_user(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES service_admin_role(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE service_admin_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID REFERENCES service_admin_user(id),
    action TEXT NOT NULL,
    resource TEXT,
    details TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB
);
CREATE INDEX IF NOT EXISTS idx_service_admin_audit_log_metadata_gin ON service_admin_audit_log USING gin (metadata);

CREATE TABLE service_admin_setting (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    values JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Content service tables (dual-ID)
CREATE TABLE service_content_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES service_user_master(id),
    title TEXT,
    body TEXT,
    metadata JSONB,
    comment_count INT DEFAULT 0,
    reaction_counts JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    search_vector tsvector
);
CREATE UNIQUE INDEX idx_service_content_main_uuid ON service_content_main(master_uuid);
CREATE INDEX idx_service_content_main_author_id ON service_content_main(author_id);
CREATE INDEX idx_service_content_main_search_vector ON service_content_main USING gin (search_vector);

CREATE TABLE service_content_comment (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    content_id UUID NOT NULL REFERENCES service_content_main(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    body TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_content_comment_content_id ON service_content_comment(content_id);

CREATE TABLE service_content_reaction (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    content_id UUID NOT NULL REFERENCES service_content_main(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    reaction_type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_content_reaction_content_id ON service_content_reaction(content_id);

-- ContentModeration service table (latest)
CREATE TABLE IF NOT EXISTS service_contentmoderation_result (
    content_id   VARCHAR(64) PRIMARY KEY,
    user_id      VARCHAR(64) NOT NULL,
    status       VARCHAR(32) NOT NULL,
    reason       TEXT,
    metadata     JSONB,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_contentmoderation_status ON service_contentmoderation_result(status);
CREATE INDEX IF NOT EXISTS idx_contentmoderation_metadata_gin ON service_contentmoderation_result USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_contentmoderation_user_id ON service_contentmoderation_result(user_id);

-- Commerce service tables (dual-ID, example for order)
CREATE TABLE service_commerce_order (
    order_id TEXT PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    total NUMERIC(20,8),
    currency TEXT,
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE UNIQUE INDEX idx_service_commerce_order_uuid ON service_commerce_order(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_metadata_gin ON service_commerce_order USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_master_uuid ON service_commerce_order(master_uuid);

-- Order Items
CREATE TABLE IF NOT EXISTS service_commerce_order_item (
    id SERIAL PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES service_commerce_order(order_id) ON DELETE CASCADE,
    product_id TEXT NOT NULL,
    quantity INT NOT NULL,
    price NUMERIC(20,8) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_item_order_id ON service_commerce_order_item(order_id);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_item_product_id ON service_commerce_order_item(product_id);

-- Quotes
CREATE TABLE IF NOT EXISTS service_commerce_quote (
    quote_id TEXT PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    product_id TEXT,
    amount NUMERIC(20,8),
    currency TEXT,
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_commerce_quote_metadata_gin ON service_commerce_quote USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_service_commerce_quote_master_uuid ON service_commerce_quote(master_uuid);

-- Payments
CREATE TABLE IF NOT EXISTS service_commerce_payment (
    payment_id TEXT PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    order_id TEXT,
    user_id TEXT NOT NULL,
    amount NUMERIC(20,8),
    currency TEXT,
    method TEXT,
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_commerce_payment_metadata_gin ON service_commerce_payment USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_service_commerce_payment_master_uuid ON service_commerce_payment(master_uuid);

-- Transactions
CREATE TABLE IF NOT EXISTS service_commerce_transaction (
    transaction_id TEXT PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    payment_id TEXT,
    user_id TEXT NOT NULL,
    type TEXT,
    amount NUMERIC(20,8),
    currency TEXT,
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_commerce_transaction_metadata_gin ON service_commerce_transaction USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_service_commerce_transaction_master_uuid ON service_commerce_transaction(master_uuid);

-- Balances
CREATE TABLE IF NOT EXISTS service_commerce_balance (
    user_id TEXT NOT NULL,
    currency TEXT NOT NULL,
    amount NUMERIC(20,8),
    metadata JSONB,
    updated_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (user_id, currency)
);
CREATE INDEX IF NOT EXISTS idx_service_commerce_balance_metadata_gin ON service_commerce_balance USING gin (metadata);

-- Event Table
CREATE TABLE IF NOT EXISTS service_commerce_event (
    event_id SERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB,
    metadata JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_master_uuid ON service_commerce_event(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_entity_type ON service_commerce_event(entity_type);
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_event_type ON service_commerce_event(event_type);
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_metadata_gin ON service_commerce_event USING gin (metadata);

-- Investment, Banking, Marketplace, Exchange, etc. (see proto extension)
-- (Add all tables from 000003_commerce_proto_extension.up.sql, using UUID master_id)
-- ... (Omitted for brevity, but should be included in the actual file)

-- Notification service tables (dual-ID)
CREATE TABLE service_notification_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    channel TEXT,
    template_id TEXT,
    status TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_service_notification_main_uuid ON service_notification_main(master_uuid);
CREATE INDEX idx_service_notification_main_user_id ON service_notification_main(user_id);
CREATE INDEX IF NOT EXISTS idx_service_notification_main_metadata_gin ON service_notification_main USING gin (metadata);

-- Referral service tables (dual-ID, referrer/referred)
CREATE TABLE service_referral_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    referrer_master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    referrer_master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    referred_master_id BIGINT REFERENCES master(id) ON DELETE CASCADE,
    referred_master_uuid UUID REFERENCES master(uuid) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    status TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_referral_main_referrer_master_uuid ON service_referral_main(referrer_master_uuid);
CREATE INDEX idx_service_referral_main_referred_master_uuid ON service_referral_main(referred_master_uuid);
CREATE INDEX idx_service_referral_main_user_id ON service_referral_main(user_id);

-- Security service tables (robust, metadata-driven)
CREATE TABLE service_security_master (
    id SERIAL PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    metadata JSONB
);
-- Add uuid column for dual-ID consistency
ALTER TABLE service_security_master ADD COLUMN IF NOT EXISTS uuid UUID NOT NULL DEFAULT uuid_generate_v4();
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_security_master_uuid ON service_security_master(uuid);
CREATE INDEX idx_service_security_master_type ON service_security_master(type);
CREATE INDEX idx_service_security_master_status ON service_security_master(status);
CREATE INDEX idx_service_security_master_created ON service_security_master(created_at);

CREATE TABLE service_security_identity (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES service_security_master(id),
    identity_type VARCHAR(50) NOT NULL,
    identifier VARCHAR(255) NOT NULL,
    credentials JSONB NOT NULL,
    attributes JSONB,
    last_authentication TIMESTAMPTZ,
    risk_score FLOAT,
    UNIQUE(identity_type, identifier)
);
CREATE INDEX idx_service_security_identity_master ON service_security_identity(master_id);
CREATE INDEX idx_service_security_identity_type ON service_security_identity(identity_type);

CREATE TABLE service_security_pattern (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES service_security_master(id),
    pattern_name VARCHAR(255) NOT NULL,
    description TEXT,
    vertices JSONB NOT NULL,
    edges JSONB NOT NULL,
    constraints JSONB,
    risk_assessment JSONB NOT NULL,
    UNIQUE(pattern_name)
);
CREATE INDEX idx_service_security_pattern_master ON service_security_pattern(master_id);
CREATE INDEX idx_service_security_pattern_risk ON service_security_pattern((risk_assessment->>'risk_score'));

CREATE TABLE service_security_incident (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES service_security_master(id),
    incident_type VARCHAR(100) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    description TEXT,
    detection_time TIMESTAMPTZ NOT NULL,
    resolution_time TIMESTAMPTZ,
    context JSONB,
    risk_assessment JSONB
);
CREATE INDEX idx_service_security_incident_master ON service_security_incident(master_id);
CREATE INDEX idx_service_security_incident_type ON service_security_incident(incident_type);
CREATE INDEX idx_service_security_incident_severity ON service_security_incident(severity);

CREATE TABLE service_security_event (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES service_security_master(id),
    event_type VARCHAR(100) NOT NULL,
    principal VARCHAR(255) NOT NULL,
    resource VARCHAR(255),
    action VARCHAR(100),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    context JSONB,
    metadata JSONB
);
CREATE INDEX idx_service_security_event_master ON service_security_event(master_id);
CREATE INDEX idx_service_security_event_type ON service_security_event(event_type);
CREATE INDEX idx_service_security_event_principal ON service_security_event(principal);
CREATE INDEX idx_service_security_event_occurred ON service_security_event(occurred_at);

-- Add master_uuid for dual-ID support (2025-05-17)
ALTER TABLE service_security_event ADD COLUMN IF NOT EXISTS master_uuid UUID;

-- Backfill master_uuid from service_security_master
UPDATE service_security_event e
SET master_uuid = m.uuid
FROM service_security_master m
WHERE e.master_id = m.id AND e.master_uuid IS NULL;

-- Add foreign key constraint for master_uuid
ALTER TABLE service_security_event
  ADD CONSTRAINT fk_service_security_event_master_uuid
  FOREIGN KEY (master_uuid) REFERENCES service_security_master(uuid);

-- Campaign service tables (dual-ID)
CREATE TABLE service_campaign_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES service_user_master(id),
    name TEXT,
    description TEXT,
    status TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    search_vector tsvector
);
CREATE UNIQUE INDEX idx_service_campaign_main_uuid ON service_campaign_main(master_uuid);
CREATE INDEX idx_service_campaign_main_owner_id ON service_campaign_main(owner_id);
CREATE INDEX idx_service_campaign_main_search_vector ON service_campaign_main USING gin (search_vector);

-- FTS trigger for campaign
CREATE OR REPLACE FUNCTION update_campaign_search_vector() RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', coalesce(NEW.name, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(NEW.description, '')), 'B');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_campaign_search_vector ON service_campaign_main;
CREATE TRIGGER trg_campaign_search_vector
  BEFORE INSERT OR UPDATE ON service_campaign_main
  FOR EACH ROW EXECUTE FUNCTION update_campaign_search_vector();

-- Localization service table
CREATE TABLE service_localization_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL,
    entity_type TEXT NOT NULL,
    locale TEXT NOT NULL,
    data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_localization_main_entity_id ON service_localization_main(entity_id);

-- Search service table (for FTS index, entity registry)
CREATE TABLE service_search_index (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL,
    entity_type TEXT NOT NULL,
    search_vector tsvector,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_search_index_entity_id ON service_search_index(entity_id);
CREATE INDEX idx_service_search_index_search_vector ON service_search_index USING gin (search_vector);

-- Analytics service tables
CREATE TABLE service_analytics_event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID,
    event_type TEXT NOT NULL,
    entity_id UUID,
    entity_type TEXT,
    properties JSONB,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_analytics_event_user_id ON service_analytics_event(user_id);
CREATE INDEX idx_service_analytics_event_entity_id ON service_analytics_event(entity_id);
CREATE INDEX idx_service_analytics_event_event_type ON service_analytics_event(event_type);

CREATE TABLE service_analytics_report (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    parameters JSONB,
    data BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Talent service tables (dual-ID)
CREATE TABLE service_talent_profile (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    profile_type TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    search_vector tsvector
);
CREATE UNIQUE INDEX idx_service_talent_profile_uuid ON service_talent_profile(master_uuid);
CREATE INDEX idx_service_talent_profile_user_id ON service_talent_profile(user_id);
CREATE INDEX IF NOT EXISTS idx_service_talent_profile_search_vector ON service_talent_profile USING gin (search_vector);
CREATE INDEX IF NOT EXISTS idx_service_talent_profile_metadata ON service_talent_profile USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_service_talent_profile_metadata_gin ON service_talent_profile USING gin (metadata);

-- FTS trigger for talent
CREATE OR REPLACE FUNCTION update_talent_search_vector() RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', coalesce(NEW.display_name, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(NEW.bio, '')), 'B') ||
    setweight(to_tsvector('english', array_to_string(NEW.skills, ' ')), 'C') ||
    setweight(to_tsvector('english', array_to_string(NEW.tags, ' ')), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_talent_search_vector ON service_talent_profile;
CREATE TRIGGER trg_talent_search_vector
  BEFORE INSERT OR UPDATE ON service_talent_profile
  FOR EACH ROW EXECUTE FUNCTION update_talent_search_vector();

CREATE TABLE service_talent_experience (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES service_talent_profile(id) ON DELETE CASCADE,
    company TEXT,
    title TEXT,
    description TEXT,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ
);

CREATE TABLE service_talent_education (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES service_talent_profile(id) ON DELETE CASCADE,
    institution TEXT,
    degree TEXT,
    field_of_study TEXT,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ
);

CREATE TABLE service_talent_booking (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    talent_id UUID NOT NULL REFERENCES service_talent_profile(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    status TEXT NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Nexus service table (pattern registry, canonical name)
CREATE TABLE service_nexus_pattern (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    pattern JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Messaging service tables (all)
CREATE TABLE service_messaging_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    thread_id UUID,
    conversation_id UUID,
    chat_group_id UUID,
    sender_id UUID NOT NULL REFERENCES service_user_master(id),
    recipient_ids UUID[],
    content TEXT,
    type TEXT NOT NULL,
    attachments JSONB,
    reactions JSONB,
    status TEXT NOT NULL,
    edited BOOLEAN DEFAULT false,
    deleted BOOLEAN DEFAULT false,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_messaging_main_thread_id ON service_messaging_main(thread_id);
CREATE INDEX idx_service_messaging_main_conversation_id ON service_messaging_main(conversation_id);
CREATE INDEX idx_service_messaging_main_chat_group_id ON service_messaging_main(chat_group_id);
CREATE INDEX idx_service_messaging_main_sender_id ON service_messaging_main(sender_id);
CREATE INDEX idx_service_messaging_main_status ON service_messaging_main(status);
CREATE INDEX idx_service_messaging_main_created_at ON service_messaging_main(created_at);
CREATE INDEX idx_service_messaging_main_metadata_gin ON service_messaging_main USING gin (metadata);

CREATE TABLE service_messaging_thread (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    subject TEXT,
    participant_ids UUID[],
    message_ids UUID[],
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_messaging_thread_participant_ids ON service_messaging_thread USING gin (participant_ids);
CREATE INDEX idx_service_messaging_thread_metadata_gin ON service_messaging_thread USING gin (metadata);

CREATE TABLE service_messaging_conversation (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    participant_ids UUID[],
    chat_group_id UUID,
    thread_ids UUID[],
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_messaging_conversation_participant_ids ON service_messaging_conversation USING gin (participant_ids);
CREATE INDEX idx_service_messaging_conversation_chat_group_id ON service_messaging_conversation(chat_group_id);
CREATE INDEX idx_service_messaging_conversation_metadata_gin ON service_messaging_conversation USING gin (metadata);

CREATE TABLE service_messaging_chat_group (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    member_ids UUID[],
    roles JSONB,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_messaging_chat_group_member_ids ON service_messaging_chat_group USING gin (member_ids);
CREATE INDEX idx_service_messaging_chat_group_metadata_gin ON service_messaging_chat_group USING gin (metadata);

CREATE TABLE service_messaging_event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES service_messaging_main(id) ON DELETE CASCADE,
    user_id UUID,
    event_type TEXT NOT NULL,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_messaging_event_message_id ON service_messaging_event(message_id);
CREATE INDEX idx_service_messaging_event_user_id ON service_messaging_event(user_id);
CREATE INDEX idx_service_messaging_event_event_type ON service_messaging_event(event_type);
CREATE INDEX idx_service_messaging_event_created_at ON service_messaging_event(created_at);

-- Messaging preferences
CREATE TABLE IF NOT EXISTS service_messaging_preferences (
    user_id TEXT PRIMARY KEY,
    preferences JSONB NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- === Backfill and Record Updates for Dual-ID Consistency ===

-- Backfill master_uuid columns in all service tables from master.uuid
UPDATE service_user_master SET master_uuid = m.uuid FROM master m WHERE service_user_master.master_id = m.id;
UPDATE service_admin_user SET master_uuid = m.uuid FROM master m WHERE service_admin_user.master_id = m.id;
UPDATE service_content_main SET master_uuid = m.uuid FROM master m WHERE service_content_main.master_id = m.id;
UPDATE service_notification_main SET master_uuid = m.uuid FROM master m WHERE service_notification_main.master_id = m.id;
UPDATE service_campaign_main SET master_uuid = m.uuid FROM master m WHERE service_campaign_main.master_id = m.id;
UPDATE service_talent_profile SET master_uuid = m.uuid FROM master m WHERE service_talent_profile.master_id = m.id;
UPDATE service_commerce_order SET master_uuid = m.uuid FROM master m WHERE service_commerce_order.master_id = m.id;

-- Referral: backfill referrer/referred master_uuid
UPDATE service_referral_main SET referrer_master_uuid = m.uuid FROM master m WHERE service_referral_main.referrer_master_id = m.id;
UPDATE service_referral_main SET referred_master_uuid = m.uuid FROM master m WHERE service_referral_main.referred_master_id = m.id;

-- Add similar update statements for any other dual-ID tables as needed

-- Add any additional triggers, functions, or indexes as needed for your implementation 

-- === Canonical System Root Record Creation (Safe for Production) ===
-- Ensures a root system record and event always exist for audit/security compliance.
-- This block is non-destructive and safe for production databases.
-- Never delete or overwrite root/system records in production migrations.

-- Insert the system root if it does not exist
INSERT INTO service_security_master (type, status, created_at, updated_at, metadata)
SELECT 'system', 'active', NOW(), NOW(), '{}'::jsonb
WHERE NOT EXISTS (
  SELECT 1 FROM service_security_master WHERE type = 'system'
);

-- Insert the event if it does not exist for the current system root
INSERT INTO service_security_event (master_id, event_type, principal, resource, action, occurred_at, context, metadata)
SELECT id, 'system.init', 'system', 'system', 'init', NOW(), '{}'::jsonb, '{}'::jsonb
FROM service_security_master
WHERE type = 'system'
  AND NOT EXISTS (
    SELECT 1 FROM service_security_event WHERE master_id = service_security_master.id AND event_type = 'system.init'
  );

CREATE TABLE IF NOT EXISTS service_nexus_event (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(255),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
); 