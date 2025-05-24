-- 000021_fix_dual_id_consistency.up.sql
-- Fixes any legacy or incorrect use of master_id UUID or master(id UUID PK) in all tables

-- Example for a table with incorrect master_id type:
DO $$
DECLARE
    r RECORD;
BEGIN
    -- For each table with master_id UUID, convert to BIGINT
    FOR r IN SELECT table_name FROM information_schema.columns WHERE column_name = 'master_id' AND data_type = 'uuid' LOOP
        EXECUTE format('ALTER TABLE %I ALTER COLUMN master_id TYPE BIGINT USING master_id::text::bigint;', r.table_name);
    END LOOP;
    -- For each table with master_id as FK to master(uuid), drop and recreate as FK to master(id)
    FOR r IN SELECT tc.table_name FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name WHERE kcu.column_name = 'master_id' AND tc.constraint_type = 'FOREIGN KEY' AND kcu.ordinal_position = 1 LOOP
        EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I_master_id_fkey;', r.table_name, r.table_name);
        EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I_master_id_fkey FOREIGN KEY (master_id) REFERENCES master(id) ON DELETE CASCADE;', r.table_name, r.table_name);
    END LOOP;
END $$;

-- Backfill master_uuid from master where needed
-- Example for service_user_master:
UPDATE service_user_master SET master_uuid = m.uuid FROM master m WHERE service_user_master.master_id = m.id;
-- Repeat for other tables as needed

-- Add unique indexes if missing
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_user_master_uuid ON service_user_master(master_uuid);
-- ... repeat for other tables ... 