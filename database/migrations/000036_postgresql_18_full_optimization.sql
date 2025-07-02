-- PostgreSQL 18 Full Optimization Migration
-- This migration transforms our entire database to leverage PostgreSQL 18's cutting-edge features
-- No holding back - full optimization for modern high-performance architecture

-- ================================
-- PostgreSQL 18 Server Configuration
-- ================================

-- Set PostgreSQL 18 specific optimizations
-- Note: These require postgresql.conf changes or ALTER SYSTEM commands

-- Enable the new async I/O subsystem (Linux only)
-- ALTER SYSTEM SET io_method = 'io_uring';

-- Optimize I/O concurrency for our multi-service architecture
-- ALTER SYSTEM SET effective_io_concurrency = 32;
-- ALTER SYSTEM SET maintenance_io_concurrency = 32;

-- Enhanced autovacuum for high-write event tables
-- ALTER SYSTEM SET autovacuum_worker_slots = 8;
-- ALTER SYSTEM SET autovacuum_vacuum_max_threshold = 1000000;
-- ALTER SYSTEM SET vacuum_max_eager_freeze_failure_rate = 0.1;

-- Enable comprehensive monitoring
-- ALTER SYSTEM SET track_cost_delay_timing = on;
-- ALTER SYSTEM SET track_wal_io_timing = on;
-- ALTER SYSTEM SET log_lock_failure = on;

-- ================================
-- Drop Old Indexes Before Optimization
-- ================================

-- Drop existing indexes that will be replaced with skip-scan optimized versions
DROP INDEX CONCURRENTLY IF EXISTS idx_service_content_main_title_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_service_content_main_body_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_service_user_master_username_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_service_product_main_name_trgm;

-- ================================
-- Master Table: PostgreSQL 18 Enhancements
-- ================================

-- Add virtual computed columns to master table
ALTER TABLE master 
ADD COLUMN entity_age_days_virtual integer 
GENERATED ALWAYS AS (
    EXTRACT(DAYS FROM (NOW() - created_at))::integer
) VIRTUAL;

-- Add virtual column for entity activity score
ALTER TABLE master 
ADD COLUMN activity_score_virtual integer 
GENERATED ALWAYS AS (
    CASE type
        WHEN 'user' THEN 100
        WHEN 'content' THEN 80
        WHEN 'product' THEN 90
        WHEN 'campaign' THEN 95
        WHEN 'order' THEN 85
        ELSE 50
    END - LEAST(EXTRACT(DAYS FROM (NOW() - updated_at))::integer, 50)
) VIRTUAL;

-- Skip-scan optimized indexes for master table
CREATE INDEX CONCURRENTLY idx_master_type_activity_age 
ON master (type, activity_score_virtual DESC, entity_age_days_virtual, id);

CREATE INDEX CONCURRENTLY idx_master_type_created_updated 
ON master (type, created_at DESC, updated_at DESC);

-- ================================
-- Service Event: Full PostgreSQL 18 Optimization
-- ================================

-- Add campaign_id to service_event for better partitioning
ALTER TABLE service_event 
ADD COLUMN campaign_id BIGINT DEFAULT 0;

-- Add virtual columns for enhanced event processing
ALTER TABLE service_event 
ADD COLUMN event_category_virtual text 
GENERATED ALWAYS AS (
    CASE 
        WHEN event_type LIKE 'user_%' THEN 'user'
        WHEN event_type LIKE 'content_%' THEN 'content'
        WHEN event_type LIKE 'product_%' THEN 'product'
        WHEN event_type LIKE 'campaign_%' THEN 'campaign'
        WHEN event_type LIKE 'commerce_%' THEN 'commerce'
        WHEN event_type LIKE 'media_%' THEN 'media'
        WHEN event_type LIKE 'system_%' THEN 'system'
        WHEN event_type LIKE '%_error' OR event_type LIKE '%_failed' THEN 'error'
        ELSE 'other'
    END
) VIRTUAL;

ALTER TABLE service_event 
ADD COLUMN importance_score_virtual integer 
GENERATED ALWAYS AS (
    CASE 
        WHEN event_type IN ('user_register', 'user_login', 'commerce_purchase_complete', 'campaign_launch') THEN 100
        WHEN event_type IN ('content_create', 'product_create', 'user_update_profile') THEN 80
        WHEN event_type LIKE '%_view' OR event_type LIKE '%_click' THEN 50
        WHEN event_type LIKE '%_error' OR event_type LIKE '%_failed' THEN 90
        WHEN event_type LIKE 'system_%' THEN 30
        ELSE 40
    END
) VIRTUAL;

ALTER TABLE service_event 
ADD COLUMN event_hour_virtual text 
GENERATED ALWAYS AS (
    TO_CHAR(occurred_at, 'YYYY-MM-DD HH24')
) VIRTUAL;

-- Skip-scan optimized indexes for events (campaign-first for isolation)
CREATE INDEX CONCURRENTLY idx_event_campaign_category_importance 
ON service_event (campaign_id, event_category_virtual, importance_score_virtual DESC, occurred_at DESC);

CREATE INDEX CONCURRENTLY idx_event_campaign_type_time 
ON service_event (campaign_id, event_type, occurred_at DESC);

CREATE INDEX CONCURRENTLY idx_event_campaign_hour_category 
ON service_event (campaign_id, event_hour_virtual, event_category_virtual);

-- Optimized for analytics queries
CREATE INDEX CONCURRENTLY idx_event_analytics_master 
ON service_event (master_id, event_category_virtual, importance_score_virtual DESC, occurred_at DESC);

-- ================================
-- Content Service: PostgreSQL 18 Virtual Columns
-- ================================

-- Enhance content with comprehensive virtual columns
ALTER TABLE service_content_main 
ADD COLUMN search_vector_full_virtual tsvector 
GENERATED ALWAYS AS (
    setweight(to_tsvector('english', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(body, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(metadata->>'tags', '')), 'C') ||
    setweight(to_tsvector('english', COALESCE(metadata->>'description', '')), 'D')
) VIRTUAL;

ALTER TABLE service_content_main 
ADD COLUMN engagement_score_virtual integer 
GENERATED ALWAYS AS (
    COALESCE(
        (metadata->>'view_count')::integer * 1 +
        (metadata->>'like_count')::integer * 5 +
        (metadata->>'share_count')::integer * 10 +
        (metadata->>'comment_count')::integer * 8 +
        (metadata->>'bookmark_count')::integer * 3,
        0
    )
) VIRTUAL;

ALTER TABLE service_content_main 
ADD COLUMN content_freshness_virtual integer 
GENERATED ALWAYS AS (
    GREATEST(0, 100 - EXTRACT(DAYS FROM (NOW() - updated_at))::integer * 2)
) VIRTUAL;

ALTER TABLE service_content_main 
ADD COLUMN seo_score_virtual integer 
GENERATED ALWAYS AS (
    CASE 
        WHEN LENGTH(title) BETWEEN 30 AND 60 THEN 30
        WHEN LENGTH(title) BETWEEN 20 AND 80 THEN 20
        ELSE 10
    END +
    CASE 
        WHEN LENGTH(body) > 300 THEN 30
        WHEN LENGTH(body) > 100 THEN 20
        ELSE 10
    END +
    CASE 
        WHEN metadata->>'meta_description' IS NOT NULL AND LENGTH(metadata->>'meta_description') BETWEEN 120 AND 160 THEN 40
        WHEN metadata->>'meta_description' IS NOT NULL THEN 20
        ELSE 0
    END
) VIRTUAL;

-- Add campaign_id if not exists
ALTER TABLE service_content_main 
ADD COLUMN IF NOT EXISTS campaign_id BIGINT DEFAULT 0;

-- Skip-scan optimized indexes for content
CREATE INDEX CONCURRENTLY idx_content_campaign_engagement_fresh 
ON service_content_main (campaign_id, engagement_score_virtual DESC, content_freshness_virtual DESC, created_at DESC);

CREATE INDEX CONCURRENTLY idx_content_campaign_status_seo 
ON service_content_main (campaign_id, status, seo_score_virtual DESC, updated_at DESC);

CREATE INDEX CONCURRENTLY idx_content_search_campaign_gin 
ON service_content_main USING GIN (campaign_id, search_vector_full_virtual);

-- ================================
-- User Service: Advanced Virtual Columns
-- ================================

-- Add campaign_id to user table
ALTER TABLE service_user_master 
ADD COLUMN IF NOT EXISTS campaign_id BIGINT DEFAULT 0;

-- Enhanced virtual columns for users
ALTER TABLE service_user_master 
ADD COLUMN display_name_virtual text 
GENERATED ALWAYS AS (
    CASE 
        WHEN profile->>'display_name' IS NOT NULL AND LENGTH(profile->>'display_name') > 0 
        THEN profile->>'display_name'
        WHEN profile->>'first_name' IS NOT NULL AND profile->>'last_name' IS NOT NULL 
        THEN profile->>'first_name' || ' ' || profile->>'last_name'
        WHEN username IS NOT NULL 
        THEN username
        ELSE 'User#' || id::text
    END
) VIRTUAL;

ALTER TABLE service_user_master 
ADD COLUMN user_tier_virtual text 
GENERATED ALWAYS AS (
    CASE 
        WHEN COALESCE((profile->>'total_spent')::numeric, 0) > 1000 THEN 'premium'
        WHEN COALESCE((profile->>'total_spent')::numeric, 0) > 100 THEN 'standard'
        WHEN last_login_at > NOW() - INTERVAL '30 days' THEN 'active'
        WHEN created_at > NOW() - INTERVAL '7 days' THEN 'new'
        ELSE 'basic'
    END
) VIRTUAL;

ALTER TABLE service_user_master 
ADD COLUMN activity_score_virtual integer 
GENERATED ALWAYS AS (
    CASE 
        WHEN last_login_at > NOW() - INTERVAL '1 day' THEN 100
        WHEN last_login_at > NOW() - INTERVAL '7 days' THEN 80
        WHEN last_login_at > NOW() - INTERVAL '30 days' THEN 60
        WHEN last_login_at > NOW() - INTERVAL '90 days' THEN 40
        WHEN last_login_at IS NOT NULL THEN 20
        ELSE 10
    END +
    COALESCE((profile->>'login_count')::integer / 10, 0) +
    COALESCE((profile->>'content_created')::integer * 2, 0)
) VIRTUAL;

-- Skip-scan optimized user indexes
CREATE INDEX CONCURRENTLY idx_user_campaign_tier_activity 
ON service_user_master (campaign_id, user_tier_virtual, activity_score_virtual DESC, last_login_at DESC);

CREATE INDEX CONCURRENTLY idx_user_campaign_status_created 
ON service_user_master (campaign_id, status, created_at DESC);

-- ================================
-- Product Service: E-commerce Optimization
-- ================================

-- Add campaign_id to products
ALTER TABLE service_product_main 
ADD COLUMN IF NOT EXISTS campaign_id BIGINT DEFAULT 0;

-- Advanced product virtual columns
ALTER TABLE service_product_main 
ADD COLUMN price_with_tax_virtual decimal(12,2) 
GENERATED ALWAYS AS (
    price * (1 + COALESCE((metadata->>'tax_rate')::decimal, 0.0))
) VIRTUAL;

ALTER TABLE service_product_main 
ADD COLUMN inventory_status_virtual text 
GENERATED ALWAYS AS (
    CASE 
        WHEN COALESCE((metadata->>'stock_count')::integer, 0) > 50 THEN 'in_stock'
        WHEN COALESCE((metadata->>'stock_count')::integer, 0) > 10 THEN 'low_stock'
        WHEN COALESCE((metadata->>'stock_count')::integer, 0) > 0 THEN 'very_low'
        ELSE 'out_of_stock'
    END
) VIRTUAL;

ALTER TABLE service_product_main 
ADD COLUMN product_score_virtual integer 
GENERATED ALWAYS AS (
    COALESCE((metadata->>'rating_avg')::numeric * 20, 50)::integer +
    LEAST(COALESCE((metadata->>'review_count')::integer, 0), 30) +
    CASE 
        WHEN created_at > NOW() - INTERVAL '30 days' THEN 20
        WHEN created_at > NOW() - INTERVAL '90 days' THEN 10
        ELSE 0
    END
) VIRTUAL;

-- Skip-scan optimized product indexes
CREATE INDEX CONCURRENTLY idx_product_campaign_status_score 
ON service_product_main (campaign_id, inventory_status_virtual, product_score_virtual DESC, updated_at DESC);

CREATE INDEX CONCURRENTLY idx_product_campaign_price_range 
ON service_product_main (campaign_id, price_with_tax_virtual, product_score_virtual DESC);

-- ================================
-- Campaign Service: Campaign Management
-- ================================

-- Add virtual columns to campaign table
ALTER TABLE service_campaign_main 
ADD COLUMN campaign_health_virtual integer 
GENERATED ALWAYS AS (
    CASE status
        WHEN 'active' THEN 100
        WHEN 'paused' THEN 50
        WHEN 'draft' THEN 25
        ELSE 0
    END +
    CASE 
        WHEN metadata->>'performance_score' IS NOT NULL 
        THEN LEAST((metadata->>'performance_score')::integer, 50)
        ELSE 25
    END
) VIRTUAL;

ALTER TABLE service_campaign_main 
ADD COLUMN days_running_virtual integer 
GENERATED ALWAYS AS (
    CASE 
        WHEN status = 'active' AND created_at IS NOT NULL 
        THEN EXTRACT(DAYS FROM (NOW() - created_at))::integer
        ELSE 0
    END
) VIRTUAL;

-- Campaign-focused indexes
CREATE INDEX CONCURRENTLY idx_campaign_health_days 
ON service_campaign_main (campaign_health_virtual DESC, days_running_virtual, created_at DESC);

-- ================================
-- Commerce Service: Order Processing
-- ================================

-- Add enhanced virtual columns for orders
ALTER TABLE service_commerce_order 
ADD COLUMN IF NOT EXISTS campaign_id BIGINT DEFAULT 0;

ALTER TABLE service_commerce_order 
ADD COLUMN order_value_tier_virtual text 
GENERATED ALWAYS AS (
    CASE 
        WHEN total_amount > 500 THEN 'high_value'
        WHEN total_amount > 100 THEN 'medium_value'
        WHEN total_amount > 20 THEN 'low_value'
        ELSE 'micro'
    END
) VIRTUAL;

ALTER TABLE service_commerce_order 
ADD COLUMN fulfillment_urgency_virtual integer 
GENERATED ALWAYS AS (
    CASE status
        WHEN 'pending' THEN 100
        WHEN 'processing' THEN 80
        WHEN 'shipped' THEN 30
        WHEN 'delivered' THEN 10
        ELSE 0
    END +
    CASE 
        WHEN created_at < NOW() - INTERVAL '24 hours' THEN 20
        WHEN created_at < NOW() - INTERVAL '12 hours' THEN 10
        ELSE 0
    END
) VIRTUAL;

-- Order processing indexes
CREATE INDEX CONCURRENTLY idx_order_campaign_urgency_value 
ON service_commerce_order (campaign_id, fulfillment_urgency_virtual DESC, order_value_tier_virtual, created_at);

-- ================================
-- Media Service: Asset Management
-- ================================

-- Add campaign scoping to media
ALTER TABLE service_media_main 
ADD COLUMN IF NOT EXISTS campaign_id BIGINT DEFAULT 0;

ALTER TABLE service_media_main 
ADD COLUMN media_efficiency_virtual integer 
GENERATED ALWAYS AS (
    CASE 
        WHEN metadata->>'compression_ratio' IS NOT NULL 
        THEN LEAST((metadata->>'compression_ratio')::numeric * 10, 50)::integer
        ELSE 25
    END +
    CASE file_type
        WHEN 'webp' THEN 30
        WHEN 'avif' THEN 40
        WHEN 'jpeg' THEN 20
        WHEN 'png' THEN 10
        ELSE 15
    END +
    CASE 
        WHEN file_size < 100000 THEN 25  -- < 100KB
        WHEN file_size < 500000 THEN 15  -- < 500KB
        WHEN file_size < 1000000 THEN 10 -- < 1MB
        ELSE 5
    END
) VIRTUAL;

-- Media optimization indexes
CREATE INDEX CONCURRENTLY idx_media_campaign_efficiency_type 
ON service_media_main (campaign_id, media_efficiency_virtual DESC, file_type, created_at DESC);

-- ================================
-- Scheduler Service: Job Optimization
-- ================================

ALTER TABLE service_scheduler_job 
ADD COLUMN IF NOT EXISTS campaign_id BIGINT DEFAULT 0;

ALTER TABLE service_scheduler_job 
ADD COLUMN job_priority_virtual integer 
GENERATED ALWAYS AS (
    CASE status
        WHEN 'PENDING' THEN 100
        WHEN 'RUNNING' THEN 80
        WHEN 'FAILED' THEN 90
        WHEN 'RETRY' THEN 95
        ELSE 10
    END +
    CASE 
        WHEN next_run_time < NOW() THEN 50  -- Overdue
        WHEN next_run_time < NOW() + INTERVAL '1 hour' THEN 30  -- Due soon
        ELSE 10
    END
) VIRTUAL;

-- Scheduler optimization indexes
CREATE INDEX CONCURRENTLY idx_scheduler_campaign_priority_time 
ON service_scheduler_job (campaign_id, job_priority_virtual DESC, next_run_time);

-- ================================
-- Cross-Service Analytics Optimization
-- ================================

-- Create materialized view for campaign analytics (PostgreSQL 18 optimized)
CREATE MATERIALIZED VIEW IF NOT EXISTS campaign_analytics_summary AS
SELECT 
    c.id as campaign_id,
    c.name as campaign_name,
    c.campaign_health_virtual,
    c.days_running_virtual,
    COUNT(DISTINCT u.id) as total_users,
    COUNT(DISTINCT co.id) as total_content,
    COUNT(DISTINCT p.id) as total_products,
    COUNT(DISTINCT o.id) as total_orders,
    COALESCE(SUM(o.total_amount), 0) as total_revenue,
    COUNT(DISTINCT e.id) as total_events,
    AVG(u.activity_score_virtual) as avg_user_activity,
    AVG(co.engagement_score_virtual) as avg_content_engagement,
    AVG(p.product_score_virtual) as avg_product_score,
    NOW() as last_updated
FROM service_campaign_main c
LEFT JOIN service_user_master u ON u.campaign_id = c.id
LEFT JOIN service_content_main co ON co.campaign_id = c.id  
LEFT JOIN service_product_main p ON p.campaign_id = c.id
LEFT JOIN service_commerce_order o ON o.campaign_id = c.id
LEFT JOIN service_event e ON e.campaign_id = c.id AND e.occurred_at > NOW() - INTERVAL '7 days'
GROUP BY c.id, c.name, c.campaign_health_virtual, c.days_running_virtual;

-- Index for the materialized view
CREATE UNIQUE INDEX CONCURRENTLY idx_campaign_analytics_summary_id 
ON campaign_analytics_summary (campaign_id);

CREATE INDEX CONCURRENTLY idx_campaign_analytics_health_revenue 
ON campaign_analytics_summary (campaign_health_virtual DESC, total_revenue DESC);

-- ================================
-- PostgreSQL 18 Function Optimizations
-- ================================

-- Enhanced search function using virtual columns
CREATE OR REPLACE FUNCTION search_content_pg18(
    p_campaign_id BIGINT,
    p_query TEXT,
    p_limit INTEGER DEFAULT 20,
    p_offset INTEGER DEFAULT 0
) RETURNS TABLE (
    id BIGINT,
    title TEXT,
    engagement_score INTEGER,
    freshness_score INTEGER,
    rank REAL
) LANGUAGE sql STABLE PARALLEL SAFE AS $$
    SELECT 
        c.id,
        c.title,
        c.engagement_score_virtual,
        c.content_freshness_virtual,
        ts_rank(c.search_vector_full_virtual, to_tsquery('english', p_query)) as rank
    FROM service_content_main c
    WHERE c.campaign_id = p_campaign_id
      AND c.search_vector_full_virtual @@ to_tsquery('english', p_query)
      AND c.status = 'published'
    ORDER BY rank DESC, c.engagement_score_virtual DESC, c.content_freshness_virtual DESC
    LIMIT p_limit OFFSET p_offset;
$$;

-- Campaign performance function
CREATE OR REPLACE FUNCTION get_campaign_performance_pg18(p_campaign_id BIGINT)
RETURNS TABLE (
    metric_name TEXT,
    metric_value NUMERIC,
    trend TEXT
) LANGUAGE sql STABLE AS $$
    SELECT * FROM (
        VALUES 
            ('total_users', (SELECT COUNT(*)::numeric FROM service_user_master WHERE campaign_id = p_campaign_id), 'stable'),
            ('avg_user_activity', (SELECT AVG(activity_score_virtual)::numeric FROM service_user_master WHERE campaign_id = p_campaign_id), 'up'),
            ('total_content', (SELECT COUNT(*)::numeric FROM service_content_main WHERE campaign_id = p_campaign_id), 'stable'),
            ('avg_engagement', (SELECT AVG(engagement_score_virtual)::numeric FROM service_content_main WHERE campaign_id = p_campaign_id), 'up'),
            ('total_orders', (SELECT COUNT(*)::numeric FROM service_commerce_order WHERE campaign_id = p_campaign_id), 'up'),
            ('total_revenue', (SELECT COALESCE(SUM(total_amount), 0)::numeric FROM service_commerce_order WHERE campaign_id = p_campaign_id), 'up')
    ) AS metrics(metric_name, metric_value, trend);
$$;

-- ================================
-- Cleanup and Refresh Statistics
-- ================================

-- Refresh materialized view
REFRESH MATERIALIZED VIEW CONCURRENTLY campaign_analytics_summary;

-- Update table statistics for PostgreSQL 18 optimizer
ANALYZE master;
ANALYZE service_event;
ANALYZE service_content_main;
ANALYZE service_user_master;
ANALYZE service_product_main;
ANALYZE service_campaign_main;
ANALYZE service_commerce_order;
ANALYZE service_media_main;
ANALYZE service_scheduler_job;

-- ================================
-- Performance Validation Queries
-- ================================

-- Test skip scan performance
/*
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM service_content_main 
WHERE campaign_id = 1 AND engagement_score_virtual > 50
ORDER BY engagement_score_virtual DESC LIMIT 10;
*/

-- Test virtual column FTS performance  
/*
EXPLAIN (ANALYZE, BUFFERS)
SELECT title, engagement_score_virtual 
FROM service_content_main 
WHERE campaign_id = 1 
  AND search_vector_full_virtual @@ to_tsquery('english', 'postgresql & performance')
ORDER BY engagement_score_virtual DESC;
*/

-- Test cross-service analytics
/*
EXPLAIN (ANALYZE, BUFFERS)
SELECT * FROM campaign_analytics_summary 
WHERE campaign_health_virtual > 75 
ORDER BY total_revenue DESC;
*/

COMMIT;
