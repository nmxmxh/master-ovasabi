-- 000031_create_scheduler_tables.up.sql
-- This migration creates the tables for the Scheduler service.
-- It includes service_scheduler_job and service_scheduler_job_run,
-- adhering to OVASABI naming conventions and patterns, including master_id, master_uuid, and campaign_id.

-- Enable uuid-ossp extension for UUID generation if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Refactor: Instead of CREATE TABLE IF NOT EXISTS for service_scheduler_job (which might already exist
-- from 000020 and have different schema), add missing columns and alter types.
-- master_uuid is added by 000023, name by 000029.
-- Ensure id remains UUID, as per repo.go's uuid.Parse usage and 000020.

-- Add missing columns to service_scheduler_job with default values first, then make NOT NULL if required
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_scheduler_job' AND column_name = 'payload') THEN
        ALTER TABLE service_scheduler_job ADD COLUMN payload TEXT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_scheduler_job' AND column_name = 'trigger_type') THEN
        ALTER TABLE service_scheduler_job ADD COLUMN trigger_type SMALLINT;
        -- Set a default for existing rows before making it NOT NULL
        UPDATE service_scheduler_job SET trigger_type = 0 WHERE trigger_type IS NULL; -- 0 for UNSPECIFIED
        ALTER TABLE service_scheduler_job ALTER COLUMN trigger_type SET NOT NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_scheduler_job' AND column_name = 'owner') THEN
        ALTER TABLE service_scheduler_job ADD COLUMN owner TEXT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_scheduler_job' AND column_name = 'labels') THEN
        ALTER TABLE service_scheduler_job ADD COLUMN labels JSONB;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_scheduler_job' AND column_name = 'campaign_id') THEN
        ALTER TABLE service_scheduler_job ADD COLUMN campaign_id BIGINT;
        -- Set a default for existing rows before making it NOT NULL
        UPDATE service_scheduler_job SET campaign_id = 0 WHERE campaign_id IS NULL;
        ALTER TABLE service_scheduler_job ALTER COLUMN campaign_id SET NOT NULL;
    END IF;
END $$;

-- Handle type changes for existing columns in service_scheduler_job
-- schedule: JSONB to TEXT
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_scheduler_job' AND column_name = 'schedule' AND data_type = 'jsonb') THEN
        ALTER TABLE service_scheduler_job ALTER COLUMN schedule TYPE TEXT USING schedule::text;
    END IF;
END $$;

-- status: INT to SMALLINT
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_scheduler_job' AND column_name = 'status' AND data_type = 'integer') THEN
        ALTER TABLE service_scheduler_job ALTER COLUMN status TYPE SMALLINT USING status::smallint;
    END IF;
END $$;

-- job_type: TEXT to SMALLINT (Requires careful data migration if existing string values are not numeric)
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_scheduler_job' AND column_name = 'job_type' AND data_type = 'text') THEN
        ALTER TABLE service_scheduler_job ALTER COLUMN job_type TYPE SMALLINT USING job_type::smallint;
    END IF;
END $$;

-- next_run_time: TIMESTAMPTZ to BIGINT (Unix timestamp)
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_scheduler_job' AND column_name = 'next_run_at' AND data_type = 'timestamp with time zone') THEN
        ALTER TABLE service_scheduler_job RENAME COLUMN next_run_at TO next_run_time; -- Rename to match proto
        ALTER TABLE service_scheduler_job ALTER COLUMN next_run_time TYPE BIGINT USING EXTRACT(EPOCH FROM next_run_time)::bigint;
    END IF;
END $$;

-- Add indexes for newly added columns (if not already present from previous migrations)
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_campaign_id ON service_scheduler_job(campaign_id);
-- master_id, master_uuid, name, status indexes are expected to be handled by previous migrations (000020, 000023, 000029)


-- Table: service_scheduler_job_run
-- Note: Foreign key to service_scheduler_job.id will be added later due to circular dependency.
CREATE TABLE IF NOT EXISTS service_scheduler_job_run (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    master_uuid UUID NOT NULL REFERENCES master(uuid) ON DELETE CASCADE,
    -- job_id TEXT NOT NULL, -- FK to service_scheduler_job(id) will be added after both tables exist
    started_at BIGINT NOT NULL, -- Unix timestamp
    finished_at BIGINT, -- Unix timestamp
    status TEXT NOT NULL,
    result TEXT,
    error TEXT,
    metadata JSONB,
    campaign_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_run_master_id ON service_scheduler_job_run(master_id); -- Already exists from 000020
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_run_master_uuid ON service_scheduler_job_run(master_uuid); -- Already exists from 000023
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_run_campaign_id ON service_scheduler_job_run(campaign_id); -- New index

-- Add foreign key constraints after both tables are created to handle circular dependencies
-- last_run_id on service_scheduler_job
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_scheduler_job' AND column_name = 'last_run_id') THEN
        ALTER TABLE service_scheduler_job ADD COLUMN last_run_id UUID;
    END IF;
    -- Add FK constraint only if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_service_scheduler_job_last_run' AND conrelid = 'service_scheduler_job'::regclass) THEN
        ALTER TABLE service_scheduler_job ADD CONSTRAINT fk_service_scheduler_job_last_run FOREIGN KEY (last_run_id) REFERENCES service_scheduler_job_run(id) ON DELETE SET NULL;
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_last_run_id ON service_scheduler_job(last_run_id);

-- job_id on service_scheduler_job_run
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'service_scheduler_job_run' AND column_name = 'job_id') THEN -- Changed TEXT to UUID and removed unnecessary UPDATE
        ALTER TABLE service_scheduler_job_run ADD COLUMN job_id UUID NOT NULL;
    END IF;
    -- Add FK constraint only if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_service_scheduler_job_run_job' AND conrelid = 'service_scheduler_job_run'::regclass) THEN
        ALTER TABLE service_scheduler_job_run ADD CONSTRAINT fk_service_scheduler_job_run_job FOREIGN KEY (job_id) REFERENCES service_scheduler_job(id) ON DELETE CASCADE;
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_run_job_id ON service_scheduler_job_run(job_id);