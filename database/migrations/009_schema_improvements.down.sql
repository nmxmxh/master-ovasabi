-- Revert row-level security
DROP POLICY IF EXISTS user_isolation_policy ON service_user;
ALTER TABLE service_user DISABLE ROW LEVEL SECURITY;

-- Revert service_event partitioning
ALTER TABLE service_event RENAME TO service_event_partitioned;
ALTER TABLE service_event_old RENAME TO service_event;
DROP TABLE service_event_partitioned CASCADE;

-- Remove table comments
COMMENT ON TABLE master IS NULL;
COMMENT ON TABLE service_event IS NULL;
COMMENT ON TABLE service_campaign IS NULL;
COMMENT ON TABLE service_user IS NULL;
COMMENT ON TABLE schema_audit_log IS NULL;

-- Remove column comments
COMMENT ON COLUMN master.id IS NULL;
COMMENT ON COLUMN master.uuid IS NULL;
COMMENT ON COLUMN master.type IS NULL;
COMMENT ON COLUMN master.version IS NULL;

-- Drop audit logging table
DROP TABLE IF EXISTS schema_audit_log;

-- Remove version columns
ALTER TABLE master DROP COLUMN IF EXISTS version;
ALTER TABLE service_campaign DROP COLUMN IF EXISTS version;
ALTER TABLE service_user DROP COLUMN IF EXISTS version;

-- Remove status constraints
ALTER TABLE service_user DROP CONSTRAINT IF EXISTS valid_status;

-- Remove GIN indexes
DROP INDEX IF EXISTS idx_service_user_profile_gin;
DROP INDEX IF EXISTS idx_service_user_metadata_gin;
DROP INDEX IF EXISTS idx_service_campaign_metadata_gin;
DROP INDEX IF EXISTS idx_service_event_payload_gin;

-- Remove triggers
DROP TRIGGER IF EXISTS update_master_updated_at ON master;
DROP TRIGGER IF EXISTS update_service_campaign_updated_at ON service_campaign;
DROP TRIGGER IF EXISTS update_service_user_updated_at ON service_user;

-- Remove trigger function
DROP FUNCTION IF EXISTS update_updated_at_column(); 