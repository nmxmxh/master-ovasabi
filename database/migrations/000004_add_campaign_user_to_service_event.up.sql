-- 000004_add_campaign_user_to_service_event.up.sql
-- Adds campaign_id and user_id columns to service_event for multi-tenant and user-specific event support

ALTER TABLE service_event ADD COLUMN IF NOT EXISTS campaign_id BIGINT;
ALTER TABLE service_event ADD COLUMN IF NOT EXISTS user_id UUID; 