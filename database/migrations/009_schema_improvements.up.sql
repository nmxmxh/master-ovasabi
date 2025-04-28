-- Add updated_at triggers for all tables
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add triggers for each table
CREATE TRIGGER update_master_updated_at
    BEFORE UPDATE ON master
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_service_campaign_updated_at
    BEFORE UPDATE ON service_campaign
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_service_user_updated_at
    BEFORE UPDATE ON service_user
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add GIN indexes for JSONB fields
CREATE INDEX IF NOT EXISTS idx_service_user_profile_gin ON service_user USING GIN (profile);
CREATE INDEX IF NOT EXISTS idx_service_user_metadata_gin ON service_user USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_service_campaign_metadata_gin ON service_campaign USING GIN (metadata);
CREATE INDEX IF NOT EXISTS idx_service_event_payload_gin ON service_event USING GIN (payload);

-- Add status constraints
ALTER TABLE service_user 
ADD CONSTRAINT valid_status 
CHECK (status IN ('active', 'inactive', 'suspended', 'deleted'));

-- Add version column for optimistic locking
ALTER TABLE master ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE service_campaign ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE service_user ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

-- Create audit logging table
CREATE TABLE IF NOT EXISTS schema_audit_log (
    id SERIAL PRIMARY KEY,
    table_name VARCHAR(128) NOT NULL,
    operation VARCHAR(16) NOT NULL,
    changed_by VARCHAR(255) NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    old_values JSONB,
    new_values JSONB
);
CREATE INDEX IF NOT EXISTS idx_schema_audit_log_table ON schema_audit_log(table_name);
CREATE INDEX IF NOT EXISTS idx_schema_audit_log_operation ON schema_audit_log(operation);
CREATE INDEX IF NOT EXISTS idx_schema_audit_log_changed_at ON schema_audit_log(changed_at);

-- Add table comments
COMMENT ON TABLE master IS 'Core entity table that serves as the central reference for all service-specific tables';
COMMENT ON TABLE service_event IS 'Centralized event logging table for tracking all service events';
COMMENT ON TABLE service_campaign IS 'Campaign management table storing campaign details and metadata';
COMMENT ON TABLE service_user IS 'User service table storing user profiles and authentication data';
COMMENT ON TABLE schema_audit_log IS 'Audit logging table for tracking schema and data changes';

-- Add column comments for master table
COMMENT ON COLUMN master.id IS 'Primary key';
COMMENT ON COLUMN master.uuid IS 'Unique external identifier';
COMMENT ON COLUMN master.type IS 'Entity type (e.g., user, campaign)';
COMMENT ON COLUMN master.version IS 'Version number for optimistic locking';

-- Partition service_event table by month
CREATE TABLE service_event_partitioned (
    LIKE service_event INCLUDING ALL
) PARTITION BY RANGE (occurred_at);

-- Create initial partition
CREATE TABLE service_event_y2024m01 PARTITION OF service_event_partitioned
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Create next month's partition
CREATE TABLE service_event_y2024m02 PARTITION OF service_event_partitioned
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Migrate data to partitioned table
INSERT INTO service_event_partitioned 
SELECT * FROM service_event;

-- Rename tables
ALTER TABLE service_event RENAME TO service_event_old;
ALTER TABLE service_event_partitioned RENAME TO service_event;

-- Add row-level security
ALTER TABLE service_user ENABLE ROW LEVEL SECURITY;

-- Create policy for user access
CREATE POLICY user_isolation_policy ON service_user
    FOR ALL
    USING (
        current_user = 'service_user_' || id::text
        OR current_user = 'admin'
    ); 