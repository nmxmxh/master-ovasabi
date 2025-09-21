-- 000007_add_ranking_formula_to_service_campaign_main.up.sql
-- Adds the 'ranking_formula' field to service_campaign_main for proto/service alignment

ALTER TABLE service_campaign_main ADD COLUMN IF NOT EXISTS ranking_formula TEXT;
