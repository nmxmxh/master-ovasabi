-- +migrate Up
-- Service and Event Registry Tables
-- 000038_create_service_event_registry.up.sql

CREATE TABLE IF NOT EXISTS service_registry (
    service_name VARCHAR(128) PRIMARY KEY,
    methods JSONB NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS event_registry (
    event_name VARCHAR(128) PRIMARY KEY,
    parameters JSONB NOT NULL,
    required_fields JSONB NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


