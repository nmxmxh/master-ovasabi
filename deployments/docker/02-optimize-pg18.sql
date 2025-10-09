-- PostgreSQL 17/18 Optimization Initialization Script
-- This script runs during container initialization to set up required extensions.

-- Create the vector extension for vector similarity search
CREATE EXTENSION IF NOT EXISTS vector;

-- Create pg_stat_statements extension for query performance analysis
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Create additional useful extensions for text search and indexing
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Log initialization completion
DO $
BEGIN
    RAISE NOTICE 'PostgreSQL 17/18 extension initialization completed successfully';
    RAISE NOTICE 'Extensions ready: vector, pg_stat_statements, pg_trgm, btree_gin, btree_gist';
END $;
