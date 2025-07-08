-- Ensure only unique patterns (by definition hash) are added to service_nexus_pattern
-- Add a hash column if not present, then enforce uniqueness
ALTER TABLE service_nexus_pattern ADD COLUMN IF NOT EXISTS definition_hash TEXT;
-- Backfill definition_hash for existing rows
UPDATE service_nexus_pattern SET definition_hash = md5(CAST(definition AS TEXT)) WHERE definition IS NOT NULL AND definition_hash IS NULL;
ALTER TABLE service_nexus_pattern ADD CONSTRAINT service_nexus_pattern_definition_hash_campaign_id_key UNIQUE (definition_hash, campaign_id);
