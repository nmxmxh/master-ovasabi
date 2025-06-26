package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.uber.org/zap"
)

var (
	// ErrMasterNotFound indicates the master record was not found.
	ErrMasterNotFound = errors.New("master record not found")
	// ErrMasterVersionConflict indicates a version conflict during update.
	ErrMasterVersionConflict = errors.New("master record version conflict")
	// ErrInvalidEntityType indicates an invalid entity type.
	ErrInvalidEntityType = errors.New("invalid entity type")
)

const (
	TTLSearchPattern = 5 * time.Minute  // Pattern search results TTL
	TTLSearchExact   = 30 * time.Minute // Exact search results TTL
	TTLSearchStats   = 1 * time.Hour    // Search statistics TTL
)

// Statement represents a SQL statement with its arguments.
type Statement struct {
	Query string
	Args  []interface{}
}

// (interface is defined in types.go).
type DefaultMasterRepository struct {
	*BaseRepository
	log *zap.Logger
}

// NewMasterRepository creates a new master repository.
func NewMasterRepository(db *sql.DB, log *zap.Logger) *DefaultMasterRepository {
	return &DefaultMasterRepository{
		BaseRepository: NewBaseRepository(db, log),
		log:            log,
	}
}

// validateEntityType checks if the entity type is valid.
func (r *DefaultMasterRepository) validateEntityType(ctx context.Context, entityType EntityType) error {
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			r.log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	var exists bool
	err = tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM entity_types WHERE type = $1)",
		entityType,
	).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		return ErrInvalidEntityType
	}

	return nil
}

// CreateMasterRecord creates a new master record in a new transaction.
// This is for cases where the master record is the only thing being created in a transaction.
func (r *DefaultMasterRepository) CreateMasterRecord(ctx context.Context, entityType, name string) (int64, string, error) {
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return 0, "", err
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
		if err != nil { // Only rollback if an error occurred in the function
			_ = tx.Rollback()
		}
	}()

	id, uuidStr, err := r.Create(ctx, tx, EntityType(entityType), name) // Call the new transactional Create
	if err != nil {
		return 0, "", err
	}

	if err := tx.Commit(); err != nil {
		return 0, "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return id, uuidStr, nil
}

// Create creates a new master record within an existing transaction.
// It returns the generated master_id, master_uuid, and an error if any.
func (r *DefaultMasterRepository) Create(ctx context.Context, tx *sql.Tx, entityType EntityType, name string) (int64, string, error) {
	if err := r.validateEntityType(ctx, entityType); err != nil {
		return 0, "", err
	}

	newUUID := uuid.New()
	var id int64
	err := tx.QueryRowContext(ctx, // Use the provided tx
		`INSERT INTO master (uuid, name, type, created_at, updated_at, is_active, version) 
		 VALUES ($1, $2, $3, NOW(), NOW(), true, 1) 
		 RETURNING id`,
		newUUID, name, entityType).Scan(&id)
	if err != nil {
		pqErr := &pq.Error{}
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" { // unique_violation
				return 0, "", fmt.Errorf("duplicate master record: %w", err)
			}
		}
		return 0, "", fmt.Errorf("failed to create master record: %w", err)
	}
	return id, newUUID.String(), nil
}

// Get retrieves a master record by ID.
func (r *DefaultMasterRepository) Get(ctx context.Context, id int64) (*Master, error) {
	master := &Master{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT id, uuid, name, type, description, version, created_at, updated_at, is_active 
		 FROM master 
		 WHERE id = $1`,
		id).Scan(
		&master.ID, &master.UUID, &master.Name, &master.Type,
		&master.Description, &master.Version, &master.CreatedAt,
		&master.UpdatedAt, &master.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMasterNotFound
		}
		return nil, fmt.Errorf("failed to get master record: %w", err)
	}
	return master, nil
}

// Delete removes a master record.
func (r *DefaultMasterRepository) Delete(ctx context.Context, id int64) error {
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			r.log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	result, err := tx.ExecContext(ctx,
		`DELETE FROM master WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete master record: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return ErrMasterNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// List retrieves a paginated list of master records.
func (r *DefaultMasterRepository) List(ctx context.Context, limit, offset int) ([]*Master, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT id, uuid, name, type, description, version, created_at, updated_at, is_active 
		 FROM master 
		 ORDER BY id 
		 LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list master records: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error("failed to close rows", zap.Error(err))
		}
	}()

	var masters []*Master
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			master := &Master{}
			err := rows.Scan(
				&master.ID, &master.UUID, &master.Name, &master.Type,
				&master.Description, &master.Version, &master.CreatedAt,
				&master.UpdatedAt, &master.IsActive)
			if err != nil {
				return nil, fmt.Errorf("failed to scan master record: %w", err)
			}
			masters = append(masters, master)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating master records: %w", err)
	}

	return masters, nil
}

// GetByUUID retrieves a master record by UUID.
func (r *DefaultMasterRepository) GetByUUID(ctx context.Context, id uuid.UUID) (*Master, error) {
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			r.log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	master := &Master{}
	err = tx.QueryRowContext(ctx,
		`SELECT id, uuid, name, type, description, version, created_at, updated_at, is_active
		FROM master
		WHERE uuid = $1`,
		id,
	).Scan(
		&master.ID, &master.UUID, &master.Name,
		&master.Type, &master.Description, &master.Version,
		&master.CreatedAt, &master.UpdatedAt, &master.IsActive,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMasterNotFound
		}
		return nil, fmt.Errorf("failed to get master record: %w", err)
	}

	return master, nil
}

// Update updates a master record with optimistic locking.
func (r *DefaultMasterRepository) Update(ctx context.Context, master *Master) error {
	if err := r.validateEntityType(ctx, master.Type); err != nil {
		return err
	}

	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			r.log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	result, err := tx.ExecContext(ctx,
		`UPDATE master 
		 SET name = $1, description = $2, is_active = $3, version = version + 1, updated_at = NOW()
		 WHERE id = $4 AND version = $5 AND type = $6`,
		master.Name, master.Description, master.IsActive,
		master.ID, master.Version, master.Type)
	if err != nil {
		pqErr := &pq.Error{}
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" {
				return fmt.Errorf("duplicate master record: %w", err)
			}
		}
		return fmt.Errorf("failed to update master record: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		// Check if the record exists
		exists, err := r.recordExists(ctx, tx, master.ID)
		if err != nil {
			return fmt.Errorf("failed to check record existence: %w", err)
		}
		if !exists {
			return ErrMasterNotFound
		}
		return ErrMasterVersionConflict
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	master.Version++
	return nil
}

// recordExists checks if a master record exists.
func (r *DefaultMasterRepository) recordExists(ctx context.Context, tx *sql.Tx, id int64) (bool, error) {
	if tx == nil {
		var err error
		tx, err = r.GetDB().BeginTx(ctx, nil)
		if err != nil {
			return false, err
		}
		defer func() {
			if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
				r.log.Error("failed to rollback transaction", zap.Error(err))
			}
		}()
	}

	var exists bool
	err := tx.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM master WHERE id = $1)",
		id,
	).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// SearchResult represents a master record with similarity score.
type SearchResult struct {
	*Master
	Similarity float64
}

// SearchByPattern searches for master records matching a pattern.
func (r *DefaultMasterRepository) SearchByPattern(ctx context.Context, pattern string, entityType EntityType, limit int) ([]*SearchResult, error) {
	if err := r.validateEntityType(ctx, entityType); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 10
	}

	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT id, name, entity_type, similarity(name, $1) as sim
		FROM masters
		WHERE entity_type = $2 AND similarity(name, $1) > 0.3
		ORDER BY sim DESC
		LIMIT $3`,
		pattern, entityType, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error("failed to close rows", zap.Error(err))
		}
	}()

	var results []*SearchResult
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			result := &SearchResult{
				Master: &Master{},
			}
			err := rows.Scan(
				&result.ID, &result.Name, &result.Type,
				&result.Similarity,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to scan search result: %w", err)
			}
			results = append(results, result)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}

// SearchByPatternAcrossTypes searches for master records matching a pattern across all entity types.
func (r *DefaultMasterRepository) SearchByPatternAcrossTypes(ctx context.Context, pattern string, limit int) ([]*SearchResult, error) {
	return r.SearchByPattern(ctx, pattern, "", limit)
}

// QuickSearch performs a fast search with default parameters.
func (r *DefaultMasterRepository) QuickSearch(ctx context.Context, pattern string) ([]*SearchResult, error) {
	return r.SearchByPatternAcrossTypes(ctx, pattern, 10)
}

// WithLock executes a function while holding a distributed lock.
func (r *DefaultMasterRepository) WithLock(ctx context.Context, _ EntityType, id interface{}, _ time.Duration, fn func() error) error {
	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  false,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			r.log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	// Try to acquire lock using SELECT FOR UPDATE
	_, err = tx.ExecContext(ctx,
		`SELECT id FROM master WHERE id = $1 FOR UPDATE NOWAIT`,
		id)
	if err != nil {
		pqErr := &pq.Error{}
		if errors.As(err, &pqErr) {
			return fmt.Errorf("failed to acquire lock: already locked")
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Execute the function
	if err := fn(); err != nil {
		return err
	}

	// Commit the transaction to release the lock
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *DefaultMasterRepository) ExecuteInTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			r.log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *DefaultMasterRepository) BatchExecute(ctx context.Context, queries []string) error {
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			r.log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	for _, query := range queries {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *DefaultMasterRepository) BatchExecuteWithArgs(ctx context.Context, statements []Statement) error {
	tx, err := r.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			r.log.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt.Query, stmt.Args...); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateWithTransaction updates a master record within a transaction.
func (r *DefaultMasterRepository) UpdateWithTransaction(ctx context.Context, tx *sql.Tx, master *Master) error {
	if tx == nil {
		var err error
		tx, err = r.GetDB().BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() {
			if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
				r.log.Error("failed to rollback transaction", zap.Error(err))
			}
		}()
	}

	if err := r.validateEntityType(ctx, master.Type); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx,
		`UPDATE master 
		 SET name = $1, description = $2, is_active = $3, version = version + 1, updated_at = NOW()
		 WHERE id = $4 AND version = $5 AND type = $6`,
		master.Name, master.Description, master.IsActive,
		master.ID, master.Version, master.Type)
	if err != nil {
		pqErr := &pq.Error{}
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" {
				return fmt.Errorf("duplicate master record: %w", err)
			}
		}
		return fmt.Errorf("failed to update master record: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		// Check if the record exists
		exists, err := r.recordExists(ctx, tx, master.ID)
		if err != nil {
			return fmt.Errorf("failed to check record existence: %w", err)
		}
		if !exists {
			return ErrMasterNotFound
		}
		return ErrMasterVersionConflict
	}

	master.Version++
	return nil
}
