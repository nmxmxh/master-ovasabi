-- Add missing 'type' column to service_nexus_pattern for Nexus pattern registration compatibility
ALTER TABLE service_nexus_pattern ADD COLUMN IF NOT EXISTS type TEXT;
-- Optionally, backfill type from pattern_type if needed
UPDATE service_nexus_pattern SET type = pattern_type WHERE type IS NULL AND pattern_type IS NOT NULL;
-- You may want to add an index if queries on type are common
CREATE INDEX IF NOT EXISTS idx_service_nexus_pattern_type ON service_nexus_pattern(type);
