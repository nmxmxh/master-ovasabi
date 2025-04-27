package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Master struct {
	ID          int
	UUID        uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	IsActive    bool
}

type MasterRepository struct {
	db *sql.DB
}

func NewMasterRepository(db *sql.DB) *MasterRepository {
	return &MasterRepository{db: db}
}

func (r *MasterRepository) Create(ctx context.Context, name, description string) (*Master, error) {
	id := 0
	newUUID := uuid.New()
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO master (uuid, name, description) VALUES ($1, $2, $3) RETURNING id`,
		newUUID, name, description,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *MasterRepository) GetByID(ctx context.Context, id int) (*Master, error) {
	var m Master
	err := r.db.QueryRowContext(ctx,
		`SELECT id, uuid, name, description, created_at, updated_at, is_active FROM master WHERE id = $1`, id,
	).Scan(&m.ID, &m.UUID, &m.Name, &m.Description, &m.CreatedAt, &m.UpdatedAt, &m.IsActive)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MasterRepository) Update(ctx context.Context, id int, name, description string, isActive bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE master SET name = $1, description = $2, is_active = $3, updated_at = NOW() WHERE id = $4`,
		name, description, isActive, id,
	)
	return err
}

func (r *MasterRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM master WHERE id = $1`, id)
	return err
}

func (r *MasterRepository) List(ctx context.Context, limit, offset int) ([]*Master, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, uuid, name, description, created_at, updated_at, is_active FROM master ORDER BY id LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var masters []*Master
	for rows.Next() {
		var m Master
		if err := rows.Scan(&m.ID, &m.UUID, &m.Name, &m.Description, &m.CreatedAt, &m.UpdatedAt, &m.IsActive); err != nil {
			return nil, fmt.Errorf("failed to scan master: %w", err)
		}
		masters = append(masters, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return masters, nil
}
