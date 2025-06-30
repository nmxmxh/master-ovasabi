package crawler

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	crawlerpb "github.com/nmxmxh/master-ovasabi/api/protos/crawler/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
)

// Repository manages crawler data in the database, ensuring transactional integrity
// for all operations that span multiple tables (e.g., master_records and service_crawler_tasks).
// This approach adheres to the database practices outlined in the project's gemini.md guide.
type Repository struct {
	db         *sql.DB
	log        *zap.Logger
	masterRepo repository.MasterRepository
}

// NewRepository creates a new crawler repository.
func NewRepository(db *sql.DB, log *zap.Logger, masterRepo repository.MasterRepository) *Repository {
	return &Repository{
		db:         db,
		log:        log,
		masterRepo: masterRepo,
	}
}

// CreateCrawlTask inserts a new crawl task and its corresponding master record within a single transaction.
func (r *Repository) CreateCrawlTask(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlTask, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback() // Rollback is a no-op if the transaction has been committed.

	// Create a master record for the task. The name is derived from the target for easy identification.
	masterID, masterUUID, err := r.masterRepo.CreateMasterRecord(ctx, "task", task.Target)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create master record")
	}
	task.MasterId = masterID
	task.MasterUuid = masterUUID

	// Generate a new service-specific UUID for the task.
	task.Uuid = uuid.New().String()

	meta, err := metadata.MarshalCanonical(task.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal metadata")
	}
	if err := metadata.ValidateMetadata(task.Metadata); err != nil {
		return nil, errors.Wrap(err, "invalid metadata")
	}

	query := `
		INSERT INTO service_crawler_tasks (uuid, master_id, master_uuid, task_type, target, depth, filters, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id, uuid, master_id, master_uuid, task_type, target, depth, filters, status, metadata, created_at, updated_at
	`

	row := tx.QueryRowContext(ctx, query,
		task.Uuid,
		task.MasterId,
		task.MasterUuid,
		task.Type,
		task.Target,
		task.Depth,
		pq.Array(task.Filters),
		task.Status,
		meta,
	)

	createdTask, err := r.scanCrawlTask(row)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan created crawl task")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	return createdTask, nil
}

// GetCrawlTask retrieves a crawl task by its UUID. This is a read-only operation.
func (r *Repository) GetCrawlTask(ctx context.Context, uuid string) (*crawlerpb.CrawlTask, error) {
	query := `
		SELECT id, uuid, master_id, master_uuid, task_type, target, depth, filters, status, metadata, created_at, updated_at
		FROM service_crawler_tasks
		WHERE uuid = $1
	`
	row := r.db.QueryRowContext(ctx, query, uuid)
	task, err := r.scanCrawlTask(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Wrapf(err, "crawl task with uuid %s not found", uuid)
		}
		return nil, errors.Wrap(err, "failed to get crawl task")
	}
	return task, nil
}

// UpdateCrawlTask updates an existing crawl task.
func (r *Repository) UpdateCrawlTask(ctx context.Context, task *crawlerpb.CrawlTask) (*crawlerpb.CrawlTask, error) {
	meta, err := metadata.MarshalCanonical(task.Metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal metadata")
	}
	if err := metadata.ValidateMetadata(task.Metadata); err != nil {
		return nil, errors.Wrap(err, "invalid metadata")
	}

	query := `
		UPDATE service_crawler_tasks
		SET status = $2, metadata = $3, updated_at = NOW()
		WHERE uuid = $1
		RETURNING id, uuid, master_id, master_uuid, task_type, target, depth, filters, status, metadata, created_at, updated_at
	`
	row := r.db.QueryRowContext(ctx, query, task.Uuid, task.Status, meta)
	updatedTask, err := r.scanCrawlTask(row)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update crawl task")
	}
	return updatedTask, nil
}

// ListCrawlTasks retrieves a paginated and filtered list of crawl tasks.
func (r *Repository) ListCrawlTasks(ctx context.Context, page, pageSize int32, filters map[string]*structpb.Value) ([]*crawlerpb.CrawlTask, int, error) {
	// ... implementation remains the same as it's a read-only operation ...
	whereClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for key, val := range filters {
		switch key {
		case "status":
			whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
			args = append(args, int32(val.GetNumberValue()))
			argIdx++
		case "type":
			whereClauses = append(whereClauses, fmt.Sprintf("task_type = $%d", argIdx))
			args = append(args, int32(val.GetNumberValue()))
			argIdx++
		}
	}

	where := ""
	if len(whereClauses) > 0 {
		where = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM service_crawler_tasks %s", where)
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, errors.Wrap(err, "failed to count crawl tasks")
	}

	query := fmt.Sprintf(`
		SELECT id, uuid, master_id, master_uuid, task_type, target, depth, filters, status, metadata, created_at, updated_at
		FROM service_crawler_tasks
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	offset := page * pageSize
	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to list crawl tasks")
	}
	defer rows.Close()

	tasks := make([]*crawlerpb.CrawlTask, 0, pageSize)
	for rows.Next() {
		task, err := r.scanCrawlTask(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}

	return tasks, total, rows.Err()
}

// StoreCrawlResult saves the result of a completed crawl task and its master record in a transaction.
func (r *Repository) StoreCrawlResult(ctx context.Context, result *crawlerpb.CrawlResult) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	task, err := r.GetCrawlTask(ctx, result.TaskUuid)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve parent task for result")
	}
	result.MasterId = task.MasterId
	result.MasterUuid = task.MasterUuid

	// Generate a new service-specific UUID for the result.
	result.Uuid = uuid.New().String()

	meta, err := metadata.MarshalCanonical(result.Metadata)
	if err != nil {
		return errors.Wrap(err, "failed to marshal result metadata")
	}

	query := `
		INSERT INTO service_crawler_results (uuid, master_id, master_uuid, task_uuid, status, extracted_content, extracted_links, error_message, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (task_uuid) DO UPDATE SET
			status = EXCLUDED.status,
			extracted_content = EXCLUDED.extracted_content,
			extracted_links = EXCLUDED.extracted_links,
			error_message = EXCLUDED.error_message,
			metadata = EXCLUDED.metadata,
			master_id = EXCLUDED.master_id,
			master_uuid = EXCLUDED.master_uuid
	`
	_, err = tx.ExecContext(ctx, query,
		result.Uuid,
		result.MasterId,
		result.MasterUuid,
		result.TaskUuid,
		result.Status,
		result.ExtractedContent,
		pq.Array(result.ExtractedLinks),
		result.ErrorMessage,
		meta,
	)
	if err != nil {
		return errors.Wrap(err, "failed to store crawl result")
	}

	return tx.Commit()
}

// scanCrawlTask is a helper function to scan a database row into a CrawlTask protobuf message.
func (r *Repository) scanCrawlTask(row interface{ Scan(...interface{}) error }) (*crawlerpb.CrawlTask, error) {
	var task crawlerpb.CrawlTask
	var metaRaw []byte
	var filters pq.StringArray
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&task.Id,
		&task.Uuid,
		&task.MasterId,
		&task.MasterUuid,
		&task.Type,
		&task.Target,
		&task.Depth,
		&filters,
		&task.Status,
		&metaRaw,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err // Return raw error for wrapping
	}

	task.Filters = filters
	task.CreatedAt = timestamppb.New(createdAt)
	task.UpdatedAt = timestamppb.New(updatedAt)

	if len(metaRaw) > 0 {
		task.Metadata = &commonpb.Metadata{}
		if err := protojson.Unmarshal(metaRaw, task.Metadata); err != nil {
			r.log.Warn("failed to unmarshal crawl task metadata, leaving it nil", zap.String("uuid", task.Uuid), zap.Error(err))
			task.Metadata = nil
		}
	}

	return &task, nil
}

// GetCrawlResult retrieves a crawl result by its task UUID.
func (r *Repository) GetCrawlResult(ctx context.Context, taskUUID string) (*crawlerpb.CrawlResult, error) {
	query := `
		SELECT id, uuid, master_id, master_uuid, task_uuid, status, extracted_content, extracted_links, error_message, metadata, created_at, updated_at
		FROM service_crawler_results
		WHERE task_uuid = $1
	`
	row := r.db.QueryRowContext(ctx, query, taskUUID)
	result, err := r.scanCrawlResult(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Wrapf(err, "crawl result for task_uuid %s not found", taskUUID)
		}
		return nil, errors.Wrap(err, "failed to get crawl result")
	}
	return result, nil
}

// scanCrawlResult is a helper function to scan a database row into a CrawlResult protobuf message.
func (r *Repository) scanCrawlResult(row interface{ Scan(...interface{}) error }) (*crawlerpb.CrawlResult, error) {
	var result crawlerpb.CrawlResult
	var metaRaw []byte
	var extractedLinks pq.StringArray
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&result.Id,
		&result.Uuid,
		&result.MasterId,
		&result.MasterUuid,
		&result.TaskUuid,
		&result.Status,
		&result.ExtractedContent,
		&extractedLinks,
		&result.ErrorMessage,
		&metaRaw,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err // Return raw error for wrapping
	}

	result.ExtractedLinks = extractedLinks
	result.CreatedAt = timestamppb.New(createdAt)
	result.UpdatedAt = timestamppb.New(updatedAt)

	if len(metaRaw) > 0 {
		result.Metadata = &commonpb.Metadata{}
		if err := protojson.Unmarshal(metaRaw, result.Metadata); err != nil {
			r.log.Warn("failed to unmarshal crawl result metadata, leaving it nil", zap.String("uuid", result.Uuid), zap.Error(err))
			result.Metadata = nil
		}
	}

	return &result, nil
}
