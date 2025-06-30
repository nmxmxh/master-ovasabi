-- 000032_create_crawler_service_tables.up.sql
-- service_crawler_tasks: Stores crawl jobs submitted to the system.
CREATE TABLE IF NOT EXISTS service_crawler_tasks (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID UNIQUE NOT NULL,
    master_id BIGINT,
    master_uuid UUID,
    task_type SMALLINT NOT NULL DEFAULT 0,
    target TEXT NOT NULL,
    depth INTEGER NOT NULL DEFAULT 0,
    filters TEXT[],
    status SMALLINT NOT NULL DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_master_id FOREIGN KEY (master_id) REFERENCES master_records(id) ON DELETE SET NULL
);

COMMENT ON TABLE service_crawler_tasks IS 'Stores crawl jobs submitted to the system, including their type, target, and status.';
COMMENT ON COLUMN service_crawler_tasks.id IS 'Unique identifier for the crawl task.';
COMMENT ON COLUMN service_crawler_tasks.uuid IS 'Service-specific unique identifier for the crawl task.';
COMMENT ON COLUMN service_crawler_tasks.master_id IS 'Foreign key to the master record for this task.';
COMMENT ON COLUMN service_crawler_tasks.master_uuid IS 'UUID of the master record for this task.';
COMMENT ON COLUMN service_crawler_tasks.task_type IS 'The type of worker to use (e.g., HTML, Torrent, API).';
COMMENT ON COLUMN service_crawler_tasks.target IS 'The resource to crawl (URL, file path, magnet link, etc.).';
COMMENT ON COLUMN service_crawler_tasks.depth IS 'Recursion depth for the crawl.';
COMMENT ON COLUMN service_crawler_tasks.filters IS 'Array of filters to apply during the crawl (e.g., no-executable).';
COMMENT ON COLUMN service_crawler_tasks.status IS 'Current status of the task (e.g., Pending, Processing, Completed).';
COMMENT ON COLUMN service_crawler_tasks.metadata IS 'Canonical metadata for orchestration and extensibility.';

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_crawler_tasks_status ON service_crawler_tasks(status);
CREATE INDEX IF NOT EXISTS idx_crawler_tasks_type ON service_crawler_tasks(task_type);
CREATE INDEX IF NOT EXISTS idx_crawler_tasks_created_at ON service_crawler_tasks(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_crawler_tasks_metadata ON service_crawler_tasks USING GIN(metadata);
CREATE INDEX IF NOT EXISTS idx_crawler_tasks_master_id ON service_crawler_tasks(master_id);
CREATE INDEX IF NOT EXISTS idx_crawler_tasks_master_uuid ON service_crawler_tasks(master_uuid);


-- service_crawler_results: Stores the output from completed crawl tasks.
CREATE TABLE IF NOT EXISTS service_crawler_results (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID UNIQUE NOT NULL,
    master_id BIGINT,
    master_uuid UUID,
    task_uuid UUID NOT NULL,
    status SMALLINT NOT NULL DEFAULT 0,
    extracted_content BYTEA,
    extracted_links TEXT[],
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_task_uuid FOREIGN KEY (task_uuid) REFERENCES service_crawler_tasks(uuid) ON DELETE CASCADE,
    CONSTRAINT fk_master_id FOREIGN KEY (master_id) REFERENCES master_records(id) ON DELETE SET NULL
);

COMMENT ON TABLE service_crawler_results IS 'Stores the output from completed crawl tasks, including extracted content and links.';
COMMENT ON COLUMN service_crawler_results.id IS 'Unique identifier for the crawl result.';
COMMENT ON COLUMN service_crawler_results.uuid IS 'Service-specific unique identifier for the crawl result.';
COMMENT ON COLUMN service_crawler_results.master_id IS 'Foreign key to the master record for this result.';
COMMENT ON COLUMN service_crawler_results.master_uuid IS 'UUID of the master record for this result.';
COMMENT ON COLUMN service_crawler_results.task_uuid IS 'Foreign key to the corresponding crawl task.';
COMMENT ON COLUMN service_crawler_results.status IS 'Final status of the task.';
COMMENT ON COLUMN service_crawler_results.extracted_content IS 'Raw or cleaned content extracted by the crawler.';
COMMENT ON COLUMN service_crawler_results.extracted_links IS 'Array of links discovered during the crawl.';
COMMENT ON COLUMN service_crawler_results.error_message IS 'Details on failure, if any.';
COMMENT ON COLUMN service_crawler_results.metadata IS 'Enriched metadata from the crawl process.';

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_crawler_results_metadata ON service_crawler_results USING GIN(metadata);
CREATE INDEX IF NOT EXISTS idx_crawler_results_master_id ON service_crawler_results(master_id);
CREATE INDEX IF NOT EXISTS idx_crawler_results_master_uuid ON service_crawler_results(master_uuid);
