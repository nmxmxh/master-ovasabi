-- 000029_add_name_to_scheduler_job.up.sql
-- Add 'name' column to service_scheduler_job for job identification and display

ALTER TABLE service_scheduler_job
ADD COLUMN name TEXT;

COMMENT ON COLUMN service_scheduler_job.name IS 'Human-readable name for the scheduled job';
