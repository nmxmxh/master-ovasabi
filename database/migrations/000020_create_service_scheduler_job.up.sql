-- 000020_create_service_scheduler_job.up.sql
-- Creates the service_scheduler_job table for robust job scheduling and orchestration

CREATE TABLE IF NOT EXISTS service_scheduler_job (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    job_type TEXT NOT NULL,
    entity_id UUID,
    entity_type TEXT,
    schedule JSONB, -- e.g., cron, interval, one-off
    status INT,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    run_count INT DEFAULT 0,
    error TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_entity_id ON service_scheduler_job(entity_id);
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_entity_type ON service_scheduler_job(entity_type);
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_job_type ON service_scheduler_job(job_type);
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_status ON service_scheduler_job(status);
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_next_run_at ON service_scheduler_job(next_run_at);
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_metadata_gin ON service_scheduler_job USING gin (metadata); 