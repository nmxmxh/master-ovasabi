ALTER TABLE service_scheduler_job ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_scheduler_job_campaign_id ON service_scheduler_job (campaign_id);

ALTER TABLE service_scheduler_job_run ADD COLUMN campaign_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX idx_scheduler_job_run_campaign_id ON service_scheduler_job_run (campaign_id); 