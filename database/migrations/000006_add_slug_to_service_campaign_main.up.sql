-- 000006_add_slug_to_service_campaign_main.up.sql
-- Adds the 'slug' field to service_campaign_main for proto/service alignment

ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS slug TEXT UNIQUE; 