package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// DefaultMasterRepository implements MasterRepository
// (interface is defined in types.go)
type DefaultMasterRepository struct {
	*BaseRepository
}

// NewMasterRepository creates a new master repository instance
func NewMasterRepository(db *sql.DB) MasterRepository {
	return &DefaultMasterRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create creates a new master record
func (r *DefaultMasterRepository) Create(ctx context.Context, entityType EntityType) (int64, error) {
	var id int64
	err := r.GetDB().QueryRowContext(ctx,
		`INSERT INTO master (uuid, type, created_at, updated_at, is_active, version) 
		 VALUES ($1, $2, NOW(), NOW(), true, 1) 
		 RETURNING id`,
		uuid.New(), entityType).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// Get retrieves a master record by ID
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
		return nil, err
	}
	return master, nil
}

// Delete removes a master record
func (r *DefaultMasterRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.GetDB().ExecContext(ctx,
		`DELETE FROM master WHERE id = $1`,
		id,
	)
	return err
}

// List retrieves a paginated list of master records
func (r *DefaultMasterRepository) List(ctx context.Context, limit, offset int) ([]*Master, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT id, uuid, name, type, description, version, created_at, updated_at, is_active 
		 FROM master 
		 ORDER BY id 
		 LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var masters []*Master
	for rows.Next() {
		master := &Master{}
		err := rows.Scan(
			&master.ID, &master.UUID, &master.Name, &master.Type,
			&master.Description, &master.Version, &master.CreatedAt,
			&master.UpdatedAt, &master.IsActive)
		if err != nil {
			return nil, err
		}
		masters = append(masters, master)
	}
	return masters, rows.Err()
}

// GetByUUID retrieves a master record by UUID
func (r *DefaultMasterRepository) GetByUUID(ctx context.Context, uuid uuid.UUID) (*Master, error) {
	master := &Master{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT id, uuid, name, type, description, version, created_at, updated_at, is_active 
		 FROM master 
		 WHERE uuid = $1`,
		uuid).Scan(
		&master.ID, &master.UUID, &master.Name, &master.Type,
		&master.Description, &master.Version, &master.CreatedAt,
		&master.UpdatedAt, &master.IsActive)
	if err != nil {
		return nil, err
	}
	return master, nil
}

// Update updates a master record
func (r *DefaultMasterRepository) Update(ctx context.Context, master *Master) error {
	result, err := r.GetDB().ExecContext(ctx,
		`UPDATE master 
		 SET name = $1, description = $2, is_active = $3, version = version + 1, updated_at = NOW()
		 WHERE id = $4 AND version = $5`,
		master.Name, master.Description, master.IsActive,
		master.ID, master.Version)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	master.Version++
	return nil
}
