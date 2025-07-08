-- Add pattern_id column to service_nexus_pattern for dual id pattern
ALTER TABLE service_nexus_pattern ADD COLUMN IF NOT EXISTS pattern_id TEXT UNIQUE;
-- Optionally, backfill pattern_id with id::text for existing rows
UPDATE service_nexus_pattern SET pattern_id = id::text WHERE pattern_id IS NULL;
