-- 000007_add_title_to_service_campaign_main.up.sql
-- Adds the 'title' field to service_campaign_main for proto/service alignment

ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS title TEXT; 