-- 01-initi.up.sql
-- OVASABI Platform: Consolidated Initial Schema (2025-05-16)
-- This file consolidates all canonical service table creation statements, including indexes, triggers, and FTS columns. It is the single source of truth for initial schema setup. All previous migration files are superseded by this file.

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Master table for all entities (dual-ID: id (int) and uuid (UUID))
CREATE TABLE IF NOT EXISTS master (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    type TEXT NOT NULL,
    name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_master_type ON master(type);
CREATE INDEX IF NOT EXISTS idx_master_name_trgm ON master USING gin (name gin_trgm_ops);

-- Centralized event logging table
CREATE TABLE IF NOT EXISTS service_event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_event_master_uuid ON service_event(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_event_event_type ON service_event(event_type);

-- Reserved usernames (for user service validation)
CREATE TABLE IF NOT EXISTS service_user_reserved_username (
    username TEXT PRIMARY KEY,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- User service tables (dual-ID)
CREATE TABLE IF NOT EXISTS service_user_master (
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
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_user_master_uuid ON service_user_master(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_user_master_email ON service_user_master(email);
CREATE INDEX IF NOT EXISTS idx_service_user_master_username ON service_user_master(username);
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
CREATE TABLE IF NOT EXISTS service_admin_user (
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
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_admin_user_uuid ON service_admin_user(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_admin_user_user_id ON service_admin_user(user_id);
CREATE INDEX IF NOT EXISTS idx_service_admin_user_metadata_gin ON service_admin_user USING gin (metadata);

CREATE TABLE IF NOT EXISTS service_admin_role (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    name TEXT NOT NULL UNIQUE,
    permissions TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB
);
CREATE INDEX IF NOT EXISTS idx_service_admin_role_metadata_gin ON service_admin_role USING gin (metadata);

CREATE TABLE IF NOT EXISTS service_admin_user_role (
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_admin_user(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES service_admin_role(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS service_admin_audit_log (
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

CREATE TABLE IF NOT EXISTS service_admin_setting (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    values JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Content service tables (dual-ID)
CREATE TABLE IF NOT EXISTS service_content_main (
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
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_content_main_uuid ON service_content_main(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_content_main_author_id ON service_content_main(author_id);
CREATE INDEX IF NOT EXISTS idx_service_content_main_search_vector ON service_content_main USING gin (search_vector);

CREATE TABLE IF NOT EXISTS service_content_comment (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    content_id UUID NOT NULL REFERENCES service_content_main(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    body TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_content_comment_content_id ON service_content_comment(content_id);

CREATE TABLE IF NOT EXISTS service_content_reaction (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    content_id UUID NOT NULL REFERENCES service_content_main(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    reaction_type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_content_reaction_content_id ON service_content_reaction(content_id);

-- ContentModeration service table
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
CREATE TABLE IF NOT EXISTS service_commerce_order (
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
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_commerce_order_uuid ON service_commerce_order(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_metadata_gin ON service_commerce_order USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_service_commerce_order_master_uuid ON service_commerce_order(master_uuid);

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

CREATE TABLE IF NOT EXISTS service_commerce_transaction (
    id SERIAL PRIMARY KEY,
    payment_id TEXT NOT NULL REFERENCES service_commerce_payment(payment_id) ON DELETE CASCADE,
    amount NUMERIC(20,8),
    currency TEXT,
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_commerce_transaction_metadata_gin ON service_commerce_transaction USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_service_commerce_transaction_master_uuid ON service_commerce_transaction ((metadata->>'master_uuid'));

CREATE TABLE IF NOT EXISTS service_commerce_balance (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    balance NUMERIC(20,8) NOT NULL,
    currency TEXT NOT NULL,
    metadata JSONB,
    updated_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_commerce_balance_metadata_gin ON service_commerce_balance USING gin (metadata);

CREATE TABLE IF NOT EXISTS service_commerce_event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB,
    master_uuid UUID REFERENCES master(uuid)
);
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_master_uuid ON service_commerce_event(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_entity_type ON service_commerce_event(entity_type);
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_event_type ON service_commerce_event(event_type);
CREATE INDEX IF NOT EXISTS idx_service_commerce_event_metadata_gin ON service_commerce_event USING gin (metadata);

-- Notification service table
CREATE TABLE IF NOT EXISTS service_notification_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    channel TEXT,
    template_id TEXT,
    payload JSONB,
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_notification_main_uuid ON service_notification_main(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_notification_main_user_id ON service_notification_main(user_id);
CREATE INDEX IF NOT EXISTS idx_service_notification_main_metadata_gin ON service_notification_main USING gin (metadata);

-- Referral service table
CREATE TABLE IF NOT EXISTS service_referral_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    referrer_master_uuid UUID REFERENCES master(uuid),
    referred_master_uuid UUID REFERENCES master(uuid),
    referral_code TEXT,
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_referral_main_referrer_master_uuid ON service_referral_main(referrer_master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_referral_main_referred_master_uuid ON service_referral_main(referred_master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_referral_main_user_id ON service_referral_main(user_id);

-- Security service tables
CREATE TABLE IF NOT EXISTS service_security_master (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4(),
    type TEXT NOT NULL,
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_security_master_uuid ON service_security_master(uuid);
CREATE INDEX IF NOT EXISTS idx_service_security_master_type ON service_security_master(type);
CREATE INDEX IF NOT EXISTS idx_service_security_master_status ON service_security_master(status);
CREATE INDEX IF NOT EXISTS idx_service_security_master_created ON service_security_master(created_at);

CREATE TABLE IF NOT EXISTS service_security_identity (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    identity_type TEXT NOT NULL,
    value TEXT NOT NULL,
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_security_identity_master ON service_security_identity(master_id);
CREATE INDEX IF NOT EXISTS idx_service_security_identity_type ON service_security_identity(identity_type);

CREATE TABLE IF NOT EXISTS service_security_pattern (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    pattern_type TEXT NOT NULL,
    risk_assessment JSONB,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_security_pattern_master ON service_security_pattern(master_id);
CREATE INDEX IF NOT EXISTS idx_service_security_pattern_risk ON service_security_pattern((risk_assessment->>'risk_score'));

CREATE TABLE IF NOT EXISTS service_security_incident (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    incident_type TEXT NOT NULL,
    severity INT,
    description TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_security_incident_master ON service_security_incident(master_id);
CREATE INDEX IF NOT EXISTS idx_service_security_incident_type ON service_security_incident(incident_type);
CREATE INDEX IF NOT EXISTS idx_service_security_incident_severity ON service_security_incident(severity);

CREATE TABLE IF NOT EXISTS service_security_event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    principal TEXT,
    details JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB
);
CREATE INDEX IF NOT EXISTS idx_service_security_event_master ON service_security_event(master_id);
CREATE INDEX IF NOT EXISTS idx_service_security_event_type ON service_security_event(event_type);
CREATE INDEX IF NOT EXISTS idx_service_security_event_principal ON service_security_event(principal);
CREATE INDEX IF NOT EXISTS idx_service_security_event_occurred ON service_security_event(occurred_at);

-- Campaign service table
CREATE TABLE IF NOT EXISTS service_campaign_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES service_user_master(id),
    name TEXT NOT NULL,
    description TEXT,
    status INT,
    start_date DATE,
    end_date DATE,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    search_vector tsvector
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_campaign_main_uuid ON service_campaign_main(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_campaign_main_owner_id ON service_campaign_main(owner_id);
CREATE INDEX IF NOT EXISTS idx_service_campaign_main_search_vector ON service_campaign_main USING gin (search_vector);

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
CREATE TABLE IF NOT EXISTS service_localization_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL,
    locale TEXT NOT NULL,
    content JSONB,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_localization_main_entity_id ON service_localization_main(entity_id);

-- Search service table
CREATE TABLE IF NOT EXISTS service_search_index (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    entity_id UUID NOT NULL,
    entity_type TEXT NOT NULL,
    search_vector tsvector,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_search_index_entity_id ON service_search_index(entity_id);
CREATE INDEX IF NOT EXISTS idx_service_search_index_search_vector ON service_search_index USING gin (search_vector);

-- Analytics service tables
CREATE TABLE IF NOT EXISTS service_analytics_event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    entity_id UUID,
    user_id UUID,
    event_type TEXT,
    payload JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB
);
CREATE INDEX IF NOT EXISTS idx_service_analytics_event_user_id ON service_analytics_event(user_id);
CREATE INDEX IF NOT EXISTS idx_service_analytics_event_entity_id ON service_analytics_event(entity_id);
CREATE INDEX IF NOT EXISTS idx_service_analytics_event_event_type ON service_analytics_event(event_type);

CREATE TABLE IF NOT EXISTS service_analytics_report (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    report_type TEXT NOT NULL,
    data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB
);

-- Talent service tables
CREATE TABLE IF NOT EXISTS service_talent_profile (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    name TEXT,
    bio TEXT,
    skills TEXT[],
    experience JSONB,
    education JSONB,
    rating NUMERIC(3,2),
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    search_vector tsvector
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_talent_profile_uuid ON service_talent_profile(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_talent_profile_user_id ON service_talent_profile(user_id);
CREATE INDEX IF NOT EXISTS idx_service_talent_profile_search_vector ON service_talent_profile USING gin (search_vector);
CREATE INDEX IF NOT EXISTS idx_service_talent_profile_metadata ON service_talent_profile USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_service_talent_profile_metadata_gin ON service_talent_profile USING gin (metadata);

-- FTS trigger for talent
CREATE OR REPLACE FUNCTION update_talent_search_vector() RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', coalesce(NEW.name, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(NEW.bio, '')), 'B');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_talent_search_vector ON service_talent_profile;
CREATE TRIGGER trg_talent_search_vector
  BEFORE INSERT OR UPDATE ON service_talent_profile
  FOR EACH ROW EXECUTE FUNCTION update_talent_search_vector();

CREATE TABLE IF NOT EXISTS service_talent_experience (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES service_talent_profile(id) ON DELETE CASCADE,
    company TEXT,
    title TEXT,
    start_date DATE,
    end_date DATE,
    description TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS service_talent_education (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES service_talent_profile(id) ON DELETE CASCADE,
    institution TEXT,
    degree TEXT,
    start_date DATE,
    end_date DATE,
    description TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS service_talent_booking (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES service_talent_profile(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Nexus pattern table
CREATE TABLE IF NOT EXISTS service_nexus_pattern (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    pattern_type TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Messaging service tables
CREATE TABLE IF NOT EXISTS service_messaging_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    thread_id UUID,
    conversation_id UUID,
    chat_group_id UUID,
    sender_id UUID NOT NULL REFERENCES service_user_master(id),
    recipient_id UUID,
    message TEXT,
    status INT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_messaging_main_thread_id ON service_messaging_main(thread_id);
CREATE INDEX IF NOT EXISTS idx_service_messaging_main_conversation_id ON service_messaging_main(conversation_id);
CREATE INDEX IF NOT EXISTS idx_service_messaging_main_chat_group_id ON service_messaging_main(chat_group_id);
CREATE INDEX IF NOT EXISTS idx_service_messaging_main_sender_id ON service_messaging_main(sender_id);
CREATE INDEX IF NOT EXISTS idx_service_messaging_main_status ON service_messaging_main(status);
CREATE INDEX IF NOT EXISTS idx_service_messaging_main_created_at ON service_messaging_main(created_at);
CREATE INDEX IF NOT EXISTS idx_service_messaging_main_metadata_gin ON service_messaging_main USING gin (metadata);

CREATE TABLE IF NOT EXISTS service_messaging_thread (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    participant_ids UUID[],
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_messaging_thread_participant_ids ON service_messaging_thread USING gin (participant_ids);
CREATE INDEX IF NOT EXISTS idx_service_messaging_thread_metadata_gin ON service_messaging_thread USING gin (metadata);

CREATE TABLE IF NOT EXISTS service_messaging_conversation (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_group_id UUID,
    participant_ids UUID[],
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_messaging_conversation_participant_ids ON service_messaging_conversation USING gin (participant_ids);
CREATE INDEX IF NOT EXISTS idx_service_messaging_conversation_chat_group_id ON service_messaging_conversation(chat_group_id);
CREATE INDEX IF NOT EXISTS idx_service_messaging_conversation_metadata_gin ON service_messaging_conversation USING gin (metadata);

CREATE TABLE IF NOT EXISTS service_messaging_chat_group (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    member_ids UUID[],
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_messaging_chat_group_member_ids ON service_messaging_chat_group USING gin (member_ids);
CREATE INDEX IF NOT EXISTS idx_service_messaging_chat_group_metadata_gin ON service_messaging_chat_group USING gin (metadata);

CREATE TABLE IF NOT EXISTS service_messaging_event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id UUID NOT NULL REFERENCES service_messaging_main(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    event_type TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB
);
CREATE INDEX IF NOT EXISTS idx_service_messaging_event_message_id ON service_messaging_event(message_id);
CREATE INDEX IF NOT EXISTS idx_service_messaging_event_user_id ON service_messaging_event(user_id);
CREATE INDEX IF NOT EXISTS idx_service_messaging_event_event_type ON service_messaging_event(event_type);
CREATE INDEX IF NOT EXISTS idx_service_messaging_event_occurred_at ON service_messaging_event(occurred_at);

CREATE TABLE IF NOT EXISTS service_messaging_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    preferences JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Nexus event table
CREATE TABLE IF NOT EXISTS service_nexus_event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB
);

-- Product service table (added from 000023_create_service_product_main.up.sql)
CREATE TABLE IF NOT EXISTS service_product_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    type INT,
    status INT,
    tags TEXT,
    metadata JSONB,
    campaign_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- Add search_vector tsvector if you use FTS
CREATE INDEX IF NOT EXISTS idx_service_product_main_master_id ON service_product_main(master_id);
CREATE INDEX IF NOT EXISTS idx_service_product_main_campaign_id ON service_product_main(campaign_id);
-- Uncomment if you use FTS:
-- CREATE INDEX IF NOT EXISTS idx_service_product_main_search_vector ON service_product_main USING GIN (search_vector); 