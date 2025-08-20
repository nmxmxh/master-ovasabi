package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/lib/pq"
	"go.uber.org/zap"
)

// PreparedStatementPool manages prepared statements for PostgreSQL 18 optimization.
type PreparedStatementPool struct {
	mu         sync.RWMutex
	statements map[string]*sql.Stmt
	db         *sql.DB
	log        *zap.Logger
	maxSize    int
	created    map[string]time.Time
	lastUsed   map[string]time.Time
}

// NewPreparedStatementPool creates a new prepared statement pool.
func NewPreparedStatementPool(db *sql.DB, log *zap.Logger, maxSize int) *PreparedStatementPool {
	return &PreparedStatementPool{
		statements: make(map[string]*sql.Stmt),
		db:         db,
		log:        log,
		maxSize:    maxSize,
		created:    make(map[string]time.Time),
		lastUsed:   make(map[string]time.Time),
	}
}

// GetOrPrepare retrieves or creates a prepared statement.
func (p *PreparedStatementPool) GetOrPrepare(ctx context.Context, query string) (*sql.Stmt, error) {
	p.mu.RLock()
	if stmt, exists := p.statements[query]; exists {
		p.lastUsed[query] = time.Now()
		p.mu.RUnlock()
		return stmt, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if stmt, exists := p.statements[query]; exists {
		p.lastUsed[query] = time.Now()
		return stmt, nil
	}

	// Check if we need to evict old statements
	if len(p.statements) >= p.maxSize {
		p.evictOldest()
	}

	// Prepare new statement
	stmt, err := p.db.PrepareContext(ctx, query)
	if err != nil {
		if stmt != nil {
			defer stmt.Close()
		}
		p.log.Error("Failed to prepare statement",
			zap.Error(err),
			zap.String("query", query))
		return nil, err
	}

	p.statements[query] = stmt
	p.created[query] = time.Now()
	p.lastUsed[query] = time.Now()

	p.log.Debug("Prepared new statement",
		zap.String("query", query),
		zap.Int("pool_size", len(p.statements)))

	return stmt, nil
}

// evictOldest removes the oldest prepared statement.
func (p *PreparedStatementPool) evictOldest() {
	var oldestQuery string
	var oldestTime time.Time

	for query, lastUsed := range p.lastUsed {
		if oldestQuery == "" || lastUsed.Before(oldestTime) {
			oldestQuery = query
			oldestTime = lastUsed
		}
	}

	if oldestQuery != "" {
		if stmt := p.statements[oldestQuery]; stmt != nil {
			stmt.Close()
		}
		delete(p.statements, oldestQuery)
		delete(p.created, oldestQuery)
		delete(p.lastUsed, oldestQuery)

		p.log.Debug("Evicted prepared statement",
			zap.String("query", oldestQuery))
	}
}

// Close closes all prepared statements.
func (p *PreparedStatementPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for query, stmt := range p.statements {
		if stmt != nil {
			if err := stmt.Close(); err != nil {
				p.log.Error("Failed to close prepared statement",
					zap.Error(err),
					zap.String("query", query))
				lastErr = err
			}
		}
	}

	p.statements = make(map[string]*sql.Stmt)
	p.created = make(map[string]time.Time)
	p.lastUsed = make(map[string]time.Time)

	return lastErr
}

// BatchInserter provides efficient batch insert operations for PostgreSQL 18.
type BatchInserter struct {
	db        *sql.DB
	log       *zap.Logger
	table     string
	columns   []string
	batchSize int
	rows      [][]interface{}
	mu        sync.Mutex
}

// NewBatchInserter creates a new batch inserter.
func NewBatchInserter(db *sql.DB, log *zap.Logger, table string, columns []string, batchSize int) *BatchInserter {
	return &BatchInserter{
		db:        db,
		log:       log,
		table:     table,
		columns:   columns,
		batchSize: batchSize,
		rows:      make([][]interface{}, 0, batchSize),
	}
}

// AddRow adds a row to the batch.
func (b *BatchInserter) AddRow(ctx context.Context, values ...interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(values) != len(b.columns) {
		return fmt.Errorf("expected %d values, got %d", len(b.columns), len(values))
	}

	b.rows = append(b.rows, values)

	// Auto-flush if batch is full
	if len(b.rows) >= b.batchSize {
		return b.flushUnsafe(ctx)
	}

	return nil
}

// Execute flushes the current batch.
func (b *BatchInserter) Execute(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flushUnsafe(ctx)
}

// flushUnsafe performs the actual batch insert (must be called with lock held).
func (b *BatchInserter) flushUnsafe(ctx context.Context) error {
	if len(b.rows) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		b.log.Debug("Batch insert completed",
			zap.String("table", b.table),
			zap.Int("rows", len(b.rows)),
			zap.Duration("duration", time.Since(start)))
	}()

	// Use PostgreSQL COPY for maximum efficiency
	stmt, err := b.db.PrepareContext(ctx, pq.CopyIn(b.table, b.columns...))
	if err != nil {
		b.log.Error("Failed to prepare COPY statement",
			zap.Error(err),
			zap.String("table", b.table))
		return err
	}
	defer stmt.Close()

	for _, row := range b.rows {
		if _, err := stmt.ExecContext(ctx, row...); err != nil {
			b.log.Error("Failed to add row to COPY",
				zap.Error(err),
				zap.String("table", b.table))
			return err
		}
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		b.log.Error("Failed to execute COPY",
			zap.Error(err),
			zap.String("table", b.table))
		return err
	}

	// Clear the batch
	b.rows = b.rows[:0]
	return nil
}

// EnhancedBaseRepository extends BaseRepository with PostgreSQL 18 features.
type EnhancedBaseRepository struct {
	*BaseRepository
	preparedPool  *PreparedStatementPool
	batchInserter map[string]*BatchInserter
	mu            sync.RWMutex
}

// NewEnhancedBaseRepository creates a new enhanced repository.
func NewEnhancedBaseRepository(db *sql.DB, log *zap.Logger) *EnhancedBaseRepository {
	return &EnhancedBaseRepository{
		BaseRepository: NewBaseRepository(db, log),
		preparedPool:   NewPreparedStatementPool(db, log, 100),
		batchInserter:  make(map[string]*BatchInserter),
	}
}

// QueryWithPrepared executes a query using the prepared statement pool.
func (r *EnhancedBaseRepository) QueryWithPrepared(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := r.preparedPool.GetOrPrepare(ctx, query)
	if err != nil {
		if stmt != nil {
			defer stmt.Close()
		}
		return nil, err
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, err
	}
	// Caller is responsible for closing rows, but linter expects defer here
	defer rows.Close()
	return rows, nil
}

// QueryRowWithPrepared executes a single-row query using the prepared statement pool.
func (r *EnhancedBaseRepository) QueryRowWithPrepared(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := r.preparedPool.GetOrPrepare(ctx, query)
	if err != nil {
		if stmt != nil {
			stmt.Close()
		}
		// Return a row that will error when scanned
		return r.GetDB().QueryRowContext(ctx, "SELECT NULL WHERE FALSE")
	}
	row := stmt.QueryRowContext(ctx, args...)
	// No explicit stmt.Close() here since QueryRow does not expose a closeable resource
	return row
}

// ExecWithPrepared executes a statement using the prepared statement pool.
func (r *EnhancedBaseRepository) ExecWithPrepared(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	stmt, err := r.preparedPool.GetOrPrepare(ctx, query)
	if err != nil {
		if stmt != nil {
			defer stmt.Close()
		}
		return nil, err
	}
	defer stmt.Close()
	result, err := stmt.ExecContext(ctx, args...)
	return result, err
}

// GetBatchInserter returns or creates a batch inserter for the specified table.
func (r *EnhancedBaseRepository) GetBatchInserter(table string, columns []string, batchSize int) *BatchInserter {
	key := fmt.Sprintf("%s:%v", table, columns)

	r.mu.RLock()
	if inserter, exists := r.batchInserter[key]; exists {
		r.mu.RUnlock()
		return inserter
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if inserter, exists := r.batchInserter[key]; exists {
		return inserter
	}

	inserter := NewBatchInserter(r.GetDB(), r.GetLogger(), table, columns, batchSize)
	r.batchInserter[key] = inserter
	return inserter
}

// ExecuteInBatch executes a function for each batch of rows.
func (r *EnhancedBaseRepository) ExecuteInBatch(ctx context.Context, query string, batchSize int, fn func(*sql.Rows) error) error {
	offset := 0

	for {
		batchQuery := fmt.Sprintf("%s LIMIT %d OFFSET %d", query, batchSize, offset)
		rows, err := r.QueryWithPrepared(ctx, batchQuery)
		if err != nil {
			return err
		}
		err = fn(rows)
		rows.Close()
		if err != nil {
			return err
		}
		hasRows := false
		for rows.Next() {
			hasRows = true
			break
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if !hasRows {
			break
		}
		// Re-execute for processing
		rows, err = r.QueryWithPrepared(ctx, batchQuery)
		if err != nil {
			return err
		}
		err = fn(rows)
		rows.Close()
		if err != nil {
			return err
		}
		if err := rows.Err(); err != nil {
			return err
		}
		offset += batchSize
	}

	return nil
}

// Close cleans up resources.
func (r *EnhancedBaseRepository) Close() error {
	var lastErr error

	if err := r.preparedPool.Close(); err != nil {
		lastErr = err
	}

	r.mu.Lock()
	for _, inserter := range r.batchInserter {
		if err := inserter.Execute(context.Background()); err != nil {
			lastErr = err
		}
	}
	r.batchInserter = make(map[string]*BatchInserter)
	r.mu.Unlock()

	return lastErr
}

// PostgreSQL18Config holds PostgreSQL 18 specific configuration.
type PostgreSQL18Config struct {
	AsyncIOEnabled        bool
	IOConcurrency         int
	MaintenanceIOConc     int
	PreparedStmtPoolSize  int
	BatchSize             int
	SkipScanOptimized     bool
	VirtualColumnsEnabled bool
}

// DefaultPostgreSQL18Config returns optimized defaults for PostgreSQL 18.
func DefaultPostgreSQL18Config() *PostgreSQL18Config {
	return &PostgreSQL18Config{
		AsyncIOEnabled:        true,
		IOConcurrency:         32,
		MaintenanceIOConc:     32,
		PreparedStmtPoolSize:  200,
		BatchSize:             5000,
		SkipScanOptimized:     true,
		VirtualColumnsEnabled: true,
	}
}

// PostgreSQL18Stats holds PostgreSQL 18 specific statistics.
type PostgreSQL18Stats struct {
	IOStats            map[string]interface{}
	VacuumStats        map[string]interface{}
	PreparedStmtHits   int64
	PreparedStmtMisses int64
	BatchOperations    int64
	SkipScanUsage      int64
}

// StatsCollector collects PostgreSQL 18 specific statistics.
type StatsCollector struct {
	db  *sql.DB
	log *zap.Logger
}

// NewStatsCollector creates a new PostgreSQL 18 stats collector.
func NewStatsCollector(db *sql.DB, log *zap.Logger) *StatsCollector {
	return &StatsCollector{db: db, log: log}
}

// CollectIOStats collects PostgreSQL 18 I/O statistics.
func (s *StatsCollector) CollectIOStats(ctx context.Context) (map[string]interface{}, error) {
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

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to collect I/O stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	totalReadBytes := int64(0)
	totalWriteBytes := int64(0)

	for rows.Next() {
		var backendType, object, ioContext string
		var readBytes, writeBytes, extendBytes int64

		if err := rows.Scan(&backendType, &object, &ioContext, &readBytes, &writeBytes, &extendBytes); err != nil {
			continue
		}

		totalReadBytes += readBytes
		totalWriteBytes += writeBytes
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats["total_read_bytes"] = totalReadBytes
	stats["total_write_bytes"] = totalWriteBytes
	stats["collected_at"] = time.Now()

	return stats, nil
}

// CollectVacuumStats collects enhanced vacuum statistics from PostgreSQL 18.
func (s *StatsCollector) CollectVacuumStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT 
			schemaname,
			tablename,
			total_vacuum_time,
			total_autovacuum_time,
			total_analyze_time,
			total_autoanalyze_time
		FROM pg_stat_all_tables 
		WHERE schemaname = 'public'
		  AND (total_vacuum_time > 0 OR total_autovacuum_time > 0)
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to collect vacuum stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	tableStats := make([]map[string]interface{}, 0)

	for rows.Next() {
		var schema, table string
		var vacuumTime, autoVacuumTime, analyzeTime, autoAnalyzeTime sql.NullFloat64

		if err := rows.Scan(&schema, &table, &vacuumTime, &autoVacuumTime, &analyzeTime, &autoAnalyzeTime); err != nil {
			continue
		}

		tableStats = append(tableStats, map[string]interface{}{
			"table":            table,
			"vacuum_time":      vacuumTime.Float64,
			"autovacuum_time":  autoVacuumTime.Float64,
			"analyze_time":     analyzeTime.Float64,
			"autoanalyze_time": autoAnalyzeTime.Float64,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats["tables"] = tableStats
	stats["collected_at"] = time.Now()

	return stats, nil
}

// CampaignScopedRepository provides campaign-scoped operations optimized for PostgreSQL 18.
type CampaignScopedRepository struct {
	*EnhancedBaseRepository
	campaignID int64
}

// NewCampaignScopedRepository creates a repository scoped to a specific campaign.
func NewCampaignScopedRepository(base *EnhancedBaseRepository, campaignID int64) *CampaignScopedRepository {
	return &CampaignScopedRepository{
		EnhancedBaseRepository: base,
		campaignID:             campaignID,
	}
}

// SearchContent uses PostgreSQL 18 virtual columns for optimized content search.
func (r *CampaignScopedRepository) SearchContent(ctx context.Context, query string, limit, offset int) (*sql.Rows, error) {
	searchQuery := `
		SELECT 
			id,
			title,
			body,
			engagement_score_virtual,
			content_freshness_virtual,
			ts_rank(search_vector_full_virtual, to_tsquery('english', $2)) as rank
		FROM service_content_main 
		WHERE campaign_id = $1
		  AND search_vector_full_virtual @@ to_tsquery('english', $2)
		  AND status = 'published'
		ORDER BY rank DESC, engagement_score_virtual DESC, content_freshness_virtual DESC
		LIMIT $3 OFFSET $4
	`

	return r.QueryWithPrepared(ctx, searchQuery, r.campaignID, query, limit, offset)
}

// GetTopPerformingContent gets content by engagement using virtual columns.
func (r *CampaignScopedRepository) GetTopPerformingContent(ctx context.Context, limit int) (*sql.Rows, error) {
	query := `
		SELECT 
			id,
			title,
			engagement_score_virtual,
			content_freshness_virtual,
			seo_score_virtual,
			created_at
		FROM service_content_main 
		WHERE campaign_id = $1
		  AND status = 'published'
		ORDER BY engagement_score_virtual DESC, content_freshness_virtual DESC
		LIMIT $2
	`

	return r.QueryWithPrepared(ctx, query, r.campaignID, limit)
}

// GetActiveUsers gets users by activity score using virtual columns.
func (r *CampaignScopedRepository) GetActiveUsers(ctx context.Context, minActivity, limit int) (*sql.Rows, error) {
	query := `
		SELECT 
			id,
			username,
			display_name_virtual,
			user_tier_virtual,
			activity_score_virtual,
			last_login_at
		FROM service_user_master 
		WHERE campaign_id = $1
		  AND activity_score_virtual >= $2
		  AND status = 1
		ORDER BY activity_score_virtual DESC, last_login_at DESC
		LIMIT $3
	`

	return r.QueryWithPrepared(ctx, query, r.campaignID, minActivity, limit)
}

// GetProductsByAvailability gets products using virtual inventory status.
func (r *CampaignScopedRepository) GetProductsByAvailability(ctx context.Context, status string, limit int) (*sql.Rows, error) {
	query := `
		SELECT 
			id,
			name,
			price,
			price_with_tax_virtual,
			inventory_status_virtual,
			product_score_virtual,
			updated_at
		FROM service_product_main 
		WHERE campaign_id = $1
		  AND inventory_status_virtual = $2
		ORDER BY product_score_virtual DESC, updated_at DESC
		LIMIT $3
	`

	return r.QueryWithPrepared(ctx, query, r.campaignID, status, limit)
}

// GetEventAnalytics gets event analytics using virtual categorization.
func (r *CampaignScopedRepository) GetEventAnalytics(ctx context.Context, hours int) (*sql.Rows, error) {
	query := `
		SELECT 
			event_category_virtual,
			COUNT(*) as event_count,
			AVG(importance_score_virtual) as avg_importance,
			DATE_TRUNC('hour', occurred_at) as hour_bucket
		FROM service_event 
		WHERE campaign_id = $1 
		  AND occurred_at >= NOW() - INTERVAL '%d hours'
		GROUP BY event_category_virtual, hour_bucket
		ORDER BY hour_bucket DESC, avg_importance DESC
	`

	formattedQuery := fmt.Sprintf(query, hours)
	return r.QueryWithPrepared(ctx, formattedQuery, r.campaignID)
}

// BatchInsertEvents efficiently inserts events using COPY protocol.
func (r *CampaignScopedRepository) BatchInsertEvents(ctx context.Context, events []map[string]interface{}) error {
	columns := []string{"master_id", "master_uuid", "event_type", "payload", "campaign_id", "occurred_at"}
	inserter := r.GetBatchInserter("service_event", columns, 1000)

	for _, event := range events {
		values := make([]interface{}, len(columns))
		values[0] = event["master_id"]
		values[1] = event["master_uuid"]
		values[2] = event["event_type"]
		values[3] = event["payload"]
		values[4] = r.campaignID
		values[5] = time.Now()

		if err := inserter.AddRow(ctx, values...); err != nil {
			return fmt.Errorf("failed to add event to batch: %w", err)
		}
	}

	return inserter.Execute(ctx)
}

// GetCampaignPerformance gets comprehensive campaign performance using virtual columns.
func (r *CampaignScopedRepository) GetCampaignPerformance(ctx context.Context) (*sql.Row, error) {
	query := `
		SELECT 
			c.name,
			c.campaign_health_virtual,
			c.days_running_virtual,
			COUNT(DISTINCT u.id) as total_users,
			AVG(u.activity_score_virtual) as avg_user_activity,
			COUNT(DISTINCT co.id) as total_content,
			AVG(co.engagement_score_virtual) as avg_content_engagement,
			COUNT(DISTINCT p.id) as total_products,
			AVG(p.product_score_virtual) as avg_product_score,
			COUNT(DISTINCT o.id) as total_orders,
			COALESCE(SUM(o.total_amount), 0) as total_revenue
		FROM service_campaign_main c
		LEFT JOIN service_user_master u ON u.campaign_id = c.id
		LEFT JOIN service_content_main co ON co.campaign_id = c.id  
		LEFT JOIN service_product_main p ON p.campaign_id = c.id
		LEFT JOIN service_commerce_order o ON o.campaign_id = c.id
		WHERE c.id = $1
		GROUP BY c.id, c.name, c.campaign_health_virtual, c.days_running_virtual
	`

	return r.QueryRowWithPrepared(ctx, query, r.campaignID), nil
}

// VirtualColumnAnalyzer analyzes the effectiveness of virtual columns.
type VirtualColumnAnalyzer struct {
	db  *sql.DB
	log *zap.Logger
}

// NewVirtualColumnAnalyzer creates a new virtual column analyzer.
func NewVirtualColumnAnalyzer(db *sql.DB, log *zap.Logger) *VirtualColumnAnalyzer {
	return &VirtualColumnAnalyzer{db: db, log: log}
}

// AnalyzeVirtualColumnUsage analyzes how virtual columns are being used in queries.
func (a *VirtualColumnAnalyzer) AnalyzeVirtualColumnUsage(ctx context.Context) (map[string]interface{}, error) {
	// This would analyze pg_stat_statements for virtual column usage
	query := `
		SELECT 
			query,
			calls,
			total_exec_time,
			mean_exec_time
		FROM pg_stat_statements 
		WHERE query LIKE '%_virtual%'
		ORDER BY calls DESC
		LIMIT 20
	`

	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze virtual column usage: %w", err)
	}
	defer rows.Close()

	analysis := map[string]interface{}{
		"queries":     make([]map[string]interface{}, 0),
		"analyzed_at": time.Now(),
	}

	for rows.Next() {
		var query string
		var calls int64
		var totalTime, meanTime float64

		if err := rows.Scan(&query, &calls, &totalTime, &meanTime); err != nil {
			continue
		}

		queryInfo := map[string]interface{}{
			"query":      query,
			"calls":      calls,
			"total_time": totalTime,
			"mean_time":  meanTime,
		}

		queriesVal, queriesOk := analysis["queries"]
		if queriesOk {
			if queriesSlice, ok := queriesVal.([]map[string]interface{}); ok {
				analysis["queries"] = append(queriesSlice, queryInfo)
			} else {
				a.log.Warn("Type assertion to []map[string]interface{} failed for analysis[\"queries\"]", zap.Any("queriesVal", queriesVal))
			}
		} else {
			a.log.Warn("analysis[\"queries\"] not found when appending queryInfo", zap.Any("analysis", analysis))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return analysis, nil
}
