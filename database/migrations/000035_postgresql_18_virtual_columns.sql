-- PostgreSQL 18 Enhancement Migration: Virtual Generated Columns
-- This migration adds virtual generated columns to leverage PostgreSQL 18's improved performance

-- ================================
-- Content Service Enhancements
-- ================================

-- Add virtual generated column for full-text search
-- This replaces application-level tsvector computation
ALTER TABLE service_content_main 
ADD COLUMN search_vector_virtual tsvector 
GENERATED ALWAYS AS (
    setweight(to_tsvector('english', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(body, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(tags, '')), 'C')
) VIRTUAL;

-- Create index optimized for skip scans with campaign filtering
CREATE INDEX CONCURRENTLY idx_content_search_virtual_campaign 
ON service_content_main USING GIN (campaign_id, search_vector_virtual);

-- Add virtual column for content score computation
ALTER TABLE service_content_main 
ADD COLUMN content_score_virtual integer 
GENERATED ALWAYS AS (
    COALESCE(
        (metadata->>'view_count')::integer * 1 +
        (metadata->>'like_count')::integer * 3 +
        (metadata->>'share_count')::integer * 5 +
        (metadata->>'comment_count')::integer * 2,
        0
    )
) VIRTUAL;

-- Index for content ranking queries
CREATE INDEX CONCURRENTLY idx_content_score_campaign 
ON service_content_main (campaign_id, content_score_virtual DESC, created_at DESC);

-- ================================
-- User Service Enhancements  
-- ================================

-- Add virtual generated column for display name
ALTER TABLE service_user_main 
ADD COLUMN display_name_virtual text 
GENERATED ALWAYS AS (
    CASE 
        WHEN first_name IS NOT NULL AND last_name IS NOT NULL 
        THEN first_name || ' ' || last_name
        WHEN username IS NOT NULL 
        THEN username
        ELSE 'User#' || id::text
    END
) VIRTUAL;

-- Add virtual column for user activity score
ALTER TABLE service_user_main 
ADD COLUMN activity_score_virtual integer 
GENERATED ALWAYS AS (
    COALESCE(
        CASE 
            WHEN last_login_at > NOW() - INTERVAL '1 day' THEN 100
            WHEN last_login_at > NOW() - INTERVAL '7 days' THEN 75
            WHEN last_login_at > NOW() - INTERVAL '30 days' THEN 50
            WHEN last_login_at > NOW() - INTERVAL '90 days' THEN 25
            ELSE 10
        END +
        (metadata->>'post_count')::integer * 2 +
        (metadata->>'interaction_count')::integer,
        0
    )
) VIRTUAL;

-- Index for user activity queries
CREATE INDEX CONCURRENTLY idx_user_activity_campaign 
ON service_user_main (campaign_id, activity_score_virtual DESC, last_login_at DESC);

-- ================================
-- Product Service Enhancements
-- ================================

-- Add virtual column for price with tax
ALTER TABLE service_product_main 
ADD COLUMN price_with_tax_virtual decimal(10,2) 
GENERATED ALWAYS AS (
    price * (1 + COALESCE((metadata->>'tax_rate')::decimal, 0.0))
) VIRTUAL;

-- Add virtual column for product availability score
ALTER TABLE service_product_main 
ADD COLUMN availability_score_virtual integer 
GENERATED ALWAYS AS (
    CASE 
        WHEN status = 'active' AND (metadata->>'stock_count')::integer > 0 THEN 100
        WHEN status = 'active' AND (metadata->>'stock_count')::integer = 0 THEN 50
        WHEN status = 'pending' THEN 25
        ELSE 0
    END
) VIRTUAL;

-- Index for product search and filtering
CREATE INDEX CONCURRENTLY idx_product_availability_campaign 
ON service_product_main (campaign_id, availability_score_virtual DESC, price_with_tax_virtual);

-- ================================
-- Event Service Enhancements
-- ================================

-- Add virtual column for event categorization
ALTER TABLE service_event 
ADD COLUMN event_category_virtual text 
GENERATED ALWAYS AS (
    CASE 
        WHEN event_type LIKE 'user_%' THEN 'user'
        WHEN event_type LIKE 'content_%' THEN 'content'
        WHEN event_type LIKE 'product_%' THEN 'product'
        WHEN event_type LIKE 'campaign_%' THEN 'campaign'
        WHEN event_type LIKE 'system_%' THEN 'system'
        ELSE 'other'
    END
) VIRTUAL;

-- Add virtual column for event importance score
ALTER TABLE service_event 
ADD COLUMN importance_score_virtual integer 
GENERATED ALWAYS AS (
    CASE 
        WHEN event_type IN ('user_register', 'user_login', 'purchase_complete') THEN 100
        WHEN event_type IN ('content_view', 'product_view', 'user_update') THEN 50
        WHEN event_type LIKE '%_error' OR event_type LIKE '%_failed' THEN 75
        ELSE 25
    END
) VIRTUAL;

-- Optimized index for event analytics with skip scan capability
CREATE INDEX CONCURRENTLY idx_event_analytics_campaign 
ON service_event (campaign_id, event_category_virtual, importance_score_virtual DESC, occurred_at DESC);

-- ================================
-- Campaign-Aware Skip Scan Indexes
-- ================================

-- Enhanced indexes designed for PostgreSQL 18 skip scan optimization
-- These indexes put campaign_id first to enable efficient campaign isolation

-- Content indexes
CREATE INDEX CONCURRENTLY idx_content_campaign_status_date 
ON service_content_main (campaign_id, status, created_at DESC, id);

CREATE INDEX CONCURRENTLY idx_content_campaign_type_score 
ON service_content_main (campaign_id, content_type, content_score_virtual DESC, updated_at DESC);

-- User indexes  
CREATE INDEX CONCURRENTLY idx_user_campaign_status_activity 
ON service_user_main (campaign_id, status, activity_score_virtual DESC, created_at DESC);

CREATE INDEX CONCURRENTLY idx_user_campaign_role_login 
ON service_user_main (campaign_id, role, last_login_at DESC, id);

-- Product indexes
CREATE INDEX CONCURRENTLY idx_product_campaign_category_availability 
ON service_product_main (campaign_id, category, availability_score_virtual DESC, created_at DESC);

CREATE INDEX CONCURRENTLY idx_product_campaign_price_range 
ON service_product_main (campaign_id, price_with_tax_virtual, created_at DESC);

-- Event indexes for analytics
CREATE INDEX CONCURRENTLY idx_event_campaign_category_time 
ON service_event (campaign_id, event_category_virtual, occurred_at DESC);

CREATE INDEX CONCURRENTLY idx_event_campaign_importance_time 
ON service_event (campaign_id, importance_score_virtual DESC, occurred_at DESC);

-- ================================
-- PostgreSQL 18 Configuration Updates
-- ================================

-- Update configuration to optimize for our workload
-- Note: These should be added to postgresql.conf and require restart

/*
# Async I/O Configuration (Linux only)
io_method = 'io_uring'
effective_io_concurrency = 32
maintenance_io_concurrency = 32

# Enhanced vacuum settings
autovacuum_worker_slots = 8
autovacuum_vacuum_max_threshold = 1000000
vacuum_max_eager_freeze_failure_rate = 0.1
vacuum_truncate = on

# Monitoring and statistics
track_cost_delay_timing = on
track_wal_io_timing = on
log_lock_failure = on

# Statement tracking
pg_stat_statements.track = 'all'
pg_stat_statements.track_utility = on

# Memory settings optimized for virtual columns
work_mem = '16MB'
maintenance_work_mem = '256MB'
shared_buffers = '1GB'
*/

-- ================================
-- Query Examples Using New Features
-- ================================

-- Example 1: Content search using virtual generated column
-- This query will use the skip scan index for campaign filtering
/*
SELECT id, title, content_score_virtual 
FROM service_content_main 
WHERE campaign_id = 123 
  AND search_vector_virtual @@ to_tsquery('english', 'postgresql & performance')
ORDER BY content_score_virtual DESC, created_at DESC
LIMIT 20;
*/

-- Example 2: User activity analysis
-- Uses virtual columns for computed activity scores
/*
SELECT display_name_virtual, activity_score_virtual, last_login_at
FROM service_user_main 
WHERE campaign_id = 123 
  AND activity_score_virtual > 50
ORDER BY activity_score_virtual DESC
LIMIT 100;
*/

-- Example 3: Product pricing with tax calculation
-- Virtual column eliminates need for application-level computation
/*
SELECT name, price, price_with_tax_virtual, availability_score_virtual
FROM service_product_main 
WHERE campaign_id = 123 
  AND availability_score_virtual > 0
  AND price_with_tax_virtual BETWEEN 10.00 AND 100.00
ORDER BY availability_score_virtual DESC, price_with_tax_virtual
LIMIT 50;
*/

-- Example 4: Event analytics by category
-- Uses virtual categorization for efficient grouping
/*
SELECT 
    event_category_virtual,
    COUNT(*) as event_count,
    AVG(importance_score_virtual) as avg_importance,
    DATE_TRUNC('hour', occurred_at) as hour_bucket
FROM service_event 
WHERE campaign_id = 123 
  AND occurred_at >= NOW() - INTERVAL '24 hours'
GROUP BY event_category_virtual, hour_bucket
ORDER BY hour_bucket DESC, avg_importance DESC;
*/

-- ================================
-- Performance Validation Queries
-- ================================

-- Use these queries to validate the performance improvements

-- Check index usage for skip scans
/*
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM service_content_main 
WHERE campaign_id = 123 AND status = 'published'
ORDER BY created_at DESC LIMIT 10;
*/

-- Validate virtual column performance
/*
EXPLAIN (ANALYZE, BUFFERS) 
SELECT title, content_score_virtual 
FROM service_content_main 
WHERE campaign_id = 123 
ORDER BY content_score_virtual DESC LIMIT 10;
*/

-- Check GIN index effectiveness
/*
EXPLAIN (ANALYZE, BUFFERS) 
SELECT title FROM service_content_main 
WHERE campaign_id = 123 
  AND search_vector_virtual @@ to_tsquery('english', 'test');
*/

-- Monitor index usage statistics
/*
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_read,
    idx_tup_fetch
FROM pg_stat_user_indexes 
WHERE indexname LIKE '%campaign%'
ORDER BY idx_scan DESC;
*/

-- ================================
-- Rollback Plan (if needed)
-- ================================

-- Drop virtual columns (this will not cause data loss)
/*
ALTER TABLE service_content_main DROP COLUMN IF EXISTS search_vector_virtual;
ALTER TABLE service_content_main DROP COLUMN IF EXISTS content_score_virtual;
ALTER TABLE service_user_main DROP COLUMN IF EXISTS display_name_virtual;
ALTER TABLE service_user_main DROP COLUMN IF EXISTS activity_score_virtual;
ALTER TABLE service_product_main DROP COLUMN IF EXISTS price_with_tax_virtual;
ALTER TABLE service_product_main DROP COLUMN IF EXISTS availability_score_virtual;
ALTER TABLE service_event DROP COLUMN IF EXISTS event_category_virtual;
ALTER TABLE service_event DROP COLUMN IF EXISTS importance_score_virtual;
*/

-- Drop indexes
/*
DROP INDEX CONCURRENTLY IF EXISTS idx_content_search_virtual_campaign;
DROP INDEX CONCURRENTLY IF EXISTS idx_content_score_campaign;
DROP INDEX CONCURRENTLY IF EXISTS idx_user_activity_campaign;
DROP INDEX CONCURRENTLY IF EXISTS idx_product_availability_campaign;
DROP INDEX CONCURRENTLY IF EXISTS idx_event_analytics_campaign;
DROP INDEX CONCURRENTLY IF EXISTS idx_content_campaign_status_date;
DROP INDEX CONCURRENTLY IF EXISTS idx_content_campaign_type_score;
DROP INDEX CONCURRENTLY IF EXISTS idx_user_campaign_status_activity;
DROP INDEX CONCURRENTLY IF EXISTS idx_user_campaign_role_login;
DROP INDEX CONCURRENTLY IF EXISTS idx_product_campaign_category_availability;
DROP INDEX CONCURRENTLY IF EXISTS idx_product_campaign_price_range;
DROP INDEX CONCURRENTLY IF EXISTS idx_event_campaign_category_time;
DROP INDEX CONCURRENTLY IF EXISTS idx_event_campaign_importance_time;
*/

COMMIT;
