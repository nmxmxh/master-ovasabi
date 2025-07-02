-- PostgreSQL 18 Optimization Initialization Script
-- This script runs during container initialization to set up PostgreSQL 18 optimizations

-- Create the vector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create pg_stat_statements extension for query analysis
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Create additional useful extensions
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Set PostgreSQL 18 specific runtime parameters
-- Note: Some parameters require postgresql.conf changes

-- Enable enhanced statistics collection
ALTER SYSTEM SET track_io_timing = on;
ALTER SYSTEM SET track_wal_io_timing = on;
ALTER SYSTEM SET track_cost_delay_timing = on;

-- Optimize for our multi-tenant campaign architecture
ALTER SYSTEM SET effective_io_concurrency = 32;
ALTER SYSTEM SET maintenance_io_concurrency = 32;

-- Enhanced autovacuum for high-write workloads
ALTER SYSTEM SET autovacuum_worker_slots = 8;
ALTER SYSTEM SET autovacuum_vacuum_max_threshold = 1000000;

-- Optimize for virtual columns and computed expressions
ALTER SYSTEM SET work_mem = '8MB';
ALTER SYSTEM SET maintenance_work_mem = '128MB';

-- Enable comprehensive query tracking
ALTER SYSTEM SET pg_stat_statements.max = 10000;
ALTER SYSTEM SET pg_stat_statements.track = 'all';

-- Reload configuration
SELECT pg_reload_conf();

-- Log initialization completion
DO $$
BEGIN
    RAISE NOTICE 'PostgreSQL 18 optimization completed successfully';
    RAISE NOTICE 'Virtual columns, skip scan indexes, and async I/O are ready';
    RAISE NOTICE 'OVASABI database layer optimized for PostgreSQL 18';
END $$;
