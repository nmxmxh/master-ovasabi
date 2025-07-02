# PostgreSQL 18 Enhancement Plan for OVASABI

This document outlines specific improvements to leverage PostgreSQL 18 features in our codebase for better performance, concurrency, and maintainability.

## Repository Layer Enhancements

### 1. Async I/O and Connection Pool Optimization

#### Current State Analysis
Our current `BaseRepository` uses standard `database/sql` with basic transaction management. We need to enhance this to leverage PostgreSQL 18's async I/O capabilities.

#### Recommended Improvements

1. **Enhanced Connection Pool Configuration**
   ```go
   // pkg/database/config.go - New configuration structure
   type AsyncDBConfig struct {
       MaxOpenConns       int
       MaxIdleConns       int
       ConnMaxLifetime    time.Duration
       AsyncIOEnabled     bool
       IOConcurrency      int
       MaintenanceIOConc  int
   }
   ```

2. **Prepared Statement Pool**
   ```go
   // internal/repository/prepared.go - New file
   type PreparedStatementPool struct {
       mu         sync.RWMutex
       statements map[string]*sql.Stmt
       db         *sql.DB
       log        *zap.Logger
   }
   
   func (p *PreparedStatementPool) GetOrPrepare(ctx context.Context, query string) (*sql.Stmt, error) {
       // Implementation with caching and automatic cleanup
   }
   ```

3. **Batch Operation Support**
   ```go
   // internal/repository/batch.go - New file
   type BatchInserter interface {
       AddRow(values ...interface{}) error
       Execute(ctx context.Context) error
       GetInsertedIDs() []int64
   }
   ```

### 2. Virtual Generated Columns Implementation

#### Service-Specific Enhancements

1. **Content Service Search Optimization**
   ```sql
   -- Migration: Add virtual generated column for content search
   ALTER TABLE service_content_main 
   ADD COLUMN search_vector_virtual tsvector 
   GENERATED ALWAYS AS (
       setweight(to_tsvector('english', COALESCE(title, '')), 'A') ||
       setweight(to_tsvector('english', COALESCE(body, '')), 'B') ||
       setweight(to_tsvector('english', COALESCE(tags, '')), 'C')
   ) VIRTUAL;
   
   CREATE INDEX idx_content_search_virtual 
   ON service_content_main USING GIN (campaign_id, search_vector_virtual);
   ```

2. **User Service Computed Fields**
   ```sql
   -- Add virtual columns for commonly computed user fields
   ALTER TABLE service_user_main 
   ADD COLUMN display_name_computed text 
   GENERATED ALWAYS AS (
       CASE 
           WHEN first_name IS NOT NULL AND last_name IS NOT NULL 
           THEN first_name || ' ' || last_name
           WHEN username IS NOT NULL 
           THEN username
           ELSE 'User#' || id::text
       END
   ) VIRTUAL;
   ```

3. **Product Service Aggregations**
   ```sql
   -- Virtual columns for product metrics
   ALTER TABLE service_product_main 
   ADD COLUMN price_with_tax_virtual decimal(10,2) 
   GENERATED ALWAYS AS (
       price * (1 + COALESCE((metadata->>'tax_rate')::decimal, 0.0))
   ) VIRTUAL;
   ```

### 3. Skip Scan Index Optimization

#### Index Strategy Updates

1. **Campaign-Scoped Composite Indexes**
   ```sql
   -- Optimized indexes for skip scan capability
   CREATE INDEX idx_content_campaign_skip 
   ON service_content_main (campaign_id, status, created_at, id);
   
   CREATE INDEX idx_events_campaign_skip 
   ON service_event (campaign_id, event_type, occurred_at, entity_id);
   
   CREATE INDEX idx_users_campaign_skip 
   ON service_user_main (campaign_id, status, last_login_at, id);
   ```

2. **Repository Method Optimization**
   ```go
   // Enhanced query methods that leverage skip scans
   func (r *Repository) FindContentByStatus(ctx context.Context, campaignID int64, statuses []string) ([]*Content, error) {
       // Query designed to use skip scan when campaign_id filter is present
       query := `
           SELECT id, title, body, status, created_at 
           FROM service_content_main 
           WHERE campaign_id = $1 AND status = ANY($2)
           ORDER BY created_at DESC
           LIMIT 100
       `
       // Implementation
   }
   ```

### 4. Improved Error Handling and Monitoring

#### Enhanced Repository Base

```go
// internal/repository/enhanced_base.go - New enhanced base repository
type EnhancedBaseRepository struct {
    *BaseRepository
    preparedPool    *PreparedStatementPool
    batchInserter   BatchInserter
    metrics         *RepositoryMetrics
}

type RepositoryMetrics struct {
    QueryDuration    prometheus.HistogramVec
    ConnectionsInUse prometheus.GaugeVec
    PreparedStmtHits prometheus.CounterVec
    BatchOperations  prometheus.CounterVec
}

func (r *EnhancedBaseRepository) QueryWithMetrics(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    start := time.Now()
    defer func() {
        r.metrics.QueryDuration.WithLabelValues("query").Observe(time.Since(start).Seconds())
    }()
    
    stmt, err := r.preparedPool.GetOrPrepare(ctx, query)
    if err != nil {
        return nil, err
    }
    
    return stmt.QueryContext(ctx, args...)
}
```

## Service-Level Improvements

### 1. Event Processing Optimization

#### Batch Event Processing
```go
// internal/service/nexus/batch_processor.go - New file
type BatchEventProcessor struct {
    repository   *nexus.Repository
    batchSize    int
    flushTimeout time.Duration
    buffer       []*nexusv1.EventRequest
    mu           sync.Mutex
}

func (p *BatchEventProcessor) ProcessEvents(ctx context.Context, events []*nexusv1.EventRequest) error {
    // Batch process events to reduce lock contention
    // Use PostgreSQL 18's improved hash operations for deduplication
}
```

### 2. Campaign-Scoped Query Patterns

#### Universal Campaign Filter
```go
// pkg/query/campaign.go - New package for campaign-scoped queries
type CampaignQueryBuilder struct {
    campaignID int64
    baseQuery  strings.Builder
    params     []interface{}
}

func NewCampaignQuery(campaignID int64) *CampaignQueryBuilder {
    return &CampaignQueryBuilder{campaignID: campaignID}
}

func (q *CampaignQueryBuilder) WithTable(table string) *CampaignQueryBuilder {
    q.baseQuery.WriteString(fmt.Sprintf("FROM %s WHERE campaign_id = $1", table))
    q.params = append(q.params, q.campaignID)
    return q
}
```

### 3. Enhanced Monitoring Integration

#### PostgreSQL 18 Statistics Integration
```go
// pkg/monitoring/pg18_stats.go - New monitoring package
type PG18StatsCollector struct {
    db  *sql.DB
    log *zap.Logger
}

func (c *PG18StatsCollector) CollectIOStats(ctx context.Context) (*IOStats, error) {
    // Collect from pg_stat_io with new byte-level metrics
    query := `
        SELECT 
            backend_type,
            object,
            context,
            read_bytes,
            write_bytes,
            extend_bytes
        FROM pg_stat_io 
        WHERE read_bytes > 0 OR write_bytes > 0
    `
    // Implementation
}

func (c *PG18StatsCollector) CollectVacuumStats(ctx context.Context) (*VacuumStats, error) {
    // Collect enhanced vacuum statistics
    query := `
        SELECT 
            schemaname,
            tablename,
            total_vacuum_time,
            total_autovacuum_time,
            total_analyze_time,
            total_autoanalyze_time
        FROM pg_stat_all_tables 
        WHERE total_vacuum_time > 0
    `
    // Implementation
}
```

## Database Migration Plan

### Phase 1: Foundation (Week 1-2)
1. **Enhanced Connection Pool Setup**
   - Configure async I/O settings
   - Implement prepared statement pooling
   - Add connection pool monitoring

2. **Repository Layer Enhancement**
   - Create `EnhancedBaseRepository`
   - Implement `PreparedStatementPool`
   - Add basic metrics collection

### Phase 2: Virtual Columns (Week 3-4)
1. **Add Virtual Generated Columns**
   - Content service search vectors
   - User service computed fields
   - Product service calculations

2. **Update Repository Methods**
   - Modify queries to use virtual columns
   - Remove application-level computations
   - Update unit tests

### Phase 3: Index Optimization (Week 5-6)
1. **Skip Scan Index Implementation**
   - Create optimized composite indexes
   - Update query patterns for skip scans
   - Monitor index usage effectiveness

2. **Partition Strategy Review**
   - Implement campaign-based partitioning for large tables
   - Update maintenance procedures
   - Test partition pruning performance

### Phase 4: Advanced Features (Week 7-8)
1. **Batch Processing Implementation**
   - Event processing optimization
   - Bulk insert operations
   - Transaction batching

2. **Enhanced Monitoring**
   - PostgreSQL 18 statistics integration
   - Performance dashboard updates
   - Alerting configuration

## Testing and Validation

### Performance Benchmarks
1. **Query Performance Tests**
   - Before/after comparison for key queries
   - Skip scan effectiveness measurement
   - Virtual column performance validation

2. **Concurrency Tests**
   - Multi-service transaction testing
   - Lock contention measurement
   - Connection pool efficiency

3. **Monitoring Validation**
   - Statistics accuracy verification
   - Alert threshold tuning
   - Dashboard functionality testing

## Configuration Templates

### PostgreSQL 18 Configuration
```sql
-- postgresql.conf enhancements for our workload

# Async I/O Configuration
io_method = 'io_uring'  # Linux only
effective_io_concurrency = 32
maintenance_io_concurrency = 32

# Connection and Memory
max_connections = 200
shared_buffers = '1GB'
work_mem = '16MB'
maintenance_work_mem = '256MB'

# Vacuum Configuration
autovacuum_worker_slots = 8
autovacuum_vacuum_max_threshold = 1000000
vacuum_max_eager_freeze_failure_rate = 0.1

# Monitoring
track_cost_delay_timing = on
track_wal_io_timing = on
log_lock_failure = on

# Statistics
pg_stat_statements.track = 'all'
pg_stat_statements.track_utility = on
```

### Go Database Configuration
```go
// pkg/database/pg18_config.go
type PG18Config struct {
    // Connection Pool
    MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" default:"50"`
    MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" default:"25"`
    ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" default:"1h"`
    
    // PostgreSQL 18 Features
    AsyncIOEnabled       bool `env:"DB_ASYNC_IO_ENABLED" default:"true"`
    PreparedStmtCaching  bool `env:"DB_PREPARED_STMT_CACHING" default:"true"`
    BatchSize           int  `env:"DB_BATCH_SIZE" default:"1000"`
    
    // Monitoring
    MetricsEnabled      bool `env:"DB_METRICS_ENABLED" default:"true"`
    SlowQueryThreshold  time.Duration `env:"DB_SLOW_QUERY_THRESHOLD" default:"1s"`
}
```

## Success Metrics

### Performance Targets
- 25% reduction in average query response time
- 40% improvement in concurrent connection handling
- 30% reduction in index scan time for campaign-scoped queries
- 50% improvement in bulk insert performance

### Operational Improvements
- Real-time visibility into PostgreSQL 18 I/O statistics
- Automated detection of slow queries and lock contention
- Proactive vacuum and index maintenance scheduling
- Campaign-level performance isolation and monitoring

## Risk Mitigation

### Rollback Strategy
1. **Database Level**
   - Virtual columns can be dropped without data loss
   - Index changes are reversible
   - Configuration rollback procedures

2. **Application Level**
   - Feature flags for new repository methods
   - Gradual migration of services
   - Fallback to previous query patterns

### Monitoring and Alerts
1. **Performance Degradation Detection**
   - Query performance regression alerts
   - Connection pool exhaustion monitoring
   - Lock contention spike detection

2. **Data Integrity Validation**
   - Virtual column computation verification
   - Index consistency checks
   - Transaction rollback monitoring
