-- 000023_enforce_dual_id_and_indexes.up.sql
-- Enforce dual-ID (master_id, master_uuid) and index standards for all services

-- Media Service
ALTER TABLE service_media_main ADD COLUMN IF NOT EXISTS master_uuid UUID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_media_main_uuid ON service_media_main(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_media_main_master_id ON service_media_main(master_id);
CREATE INDEX IF NOT EXISTS idx_service_media_main_metadata_gin ON service_media_main USING gin (metadata);

-- Scheduler Service
ALTER TABLE service_scheduler_job ADD COLUMN IF NOT EXISTS master_uuid UUID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_scheduler_job_uuid ON service_scheduler_job(master_uuid);
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_master_id ON service_scheduler_job(master_id);
CREATE INDEX IF NOT EXISTS idx_service_scheduler_job_metadata_gin ON service_scheduler_job USING gin (metadata);

-- User Service
ALTER TABLE service_user_master ADD CONSTRAINT unique_username UNIQUE (username); 