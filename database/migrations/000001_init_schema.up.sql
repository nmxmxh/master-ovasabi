-- +goose Up
-- Consolidated initial schema for all core services (2025-05-12)

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Master table for all entities
CREATE TABLE master (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type TEXT NOT NULL,
    name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_master_type ON master(type);
CREATE INDEX idx_master_name_trgm ON master USING gin (name gin_trgm_ops);

-- Centralized event logging table
CREATE TABLE service_event (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_event_master_id ON service_event(master_id);
CREATE INDEX idx_service_event_event_type ON service_event(event_type);

-- User service tables
CREATE TABLE service_user_master (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    profile JSONB,
    roles TEXT[],
    status SMALLINT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_user_master_email ON service_user_master(email);
CREATE INDEX idx_service_user_master_username ON service_user_master(username);

-- Admin service tables
CREATE TABLE service_admin_user (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID REFERENCES service_user_master(id),
    email TEXT NOT NULL UNIQUE,
    name TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_admin_user_user_id ON service_admin_user(user_id);

CREATE TABLE service_admin_role (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    name TEXT NOT NULL UNIQUE,
    permissions TEXT[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE service_admin_user_role (
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_admin_user(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES service_admin_role(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE service_admin_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID REFERENCES service_admin_user(id),
    action TEXT NOT NULL,
    resource TEXT,
    details TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE service_admin_setting (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    values JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Content service tables
CREATE TABLE service_content_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
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
CREATE INDEX idx_service_content_main_author_id ON service_content_main(author_id);
CREATE INDEX idx_service_content_main_search_vector ON service_content_main USING gin (search_vector);

CREATE TABLE service_content_comment (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    content_id UUID NOT NULL REFERENCES service_content_main(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    body TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_content_comment_content_id ON service_content_comment(content_id);

CREATE TABLE service_content_reaction (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    content_id UUID NOT NULL REFERENCES service_content_main(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    reaction_type TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_content_reaction_content_id ON service_content_reaction(content_id);

-- Commerce service table
CREATE TABLE service_commerce_order (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    amount NUMERIC(12,2) NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_commerce_order_user_id ON service_commerce_order(user_id);

-- Notification service table
CREATE TABLE service_notification_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    type TEXT NOT NULL,
    payload JSONB,
    read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_notification_main_user_id ON service_notification_main(user_id);

-- Referral service table
CREATE TABLE service_referral_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    code TEXT NOT NULL UNIQUE,
    referred_by UUID REFERENCES service_user_master(id),
    status TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_referral_main_user_id ON service_referral_main(user_id);

-- Security service table
CREATE TABLE service_security_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    service TEXT NOT NULL,
    user_id UUID REFERENCES service_user_master(id),
    action TEXT NOT NULL,
    resource TEXT,
    details TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Campaign service table
CREATE TABLE service_campaign_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    owner_id UUID REFERENCES service_user_master(id),
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Localization service table
CREATE TABLE service_localization_main (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
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
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
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
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
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
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    parameters JSONB,
    data BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ContentModeration service table
CREATE TABLE service_contentmoderation_result (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    content_id UUID NOT NULL,
    user_id UUID,
    status SMALLINT NOT NULL,
    reason TEXT,
    scores JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_service_contentmoderation_result_content_id ON service_contentmoderation_result(content_id);
CREATE INDEX idx_service_contentmoderation_result_status ON service_contentmoderation_result(status);

-- Talent service tables
CREATE TABLE service_talent_profile (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES service_user_master(id),
    display_name TEXT NOT NULL,
    bio TEXT,
    skills TEXT[],
    tags TEXT[],
    location TEXT,
    avatar_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE service_talent_experience (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES service_talent_profile(id) ON DELETE CASCADE,
    company TEXT,
    title TEXT,
    description TEXT,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ
);

CREATE TABLE service_talent_education (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES service_talent_profile(id) ON DELETE CASCADE,
    institution TEXT,
    degree TEXT,
    field_of_study TEXT,
    start_date TIMESTAMPTZ,
    end_date TIMESTAMPTZ
);

CREATE TABLE service_talent_booking (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    talent_id UUID NOT NULL REFERENCES service_talent_profile(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    status TEXT NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Nexus service table (pattern registry)
CREATE TABLE service_nexus_pattern (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id UUID NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    pattern JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Triggers for content comment/reaction counters (example, can be extended)
-- (Add trigger functions and triggers as needed for your implementation) 