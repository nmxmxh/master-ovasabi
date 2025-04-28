package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/nmxmxh/master-ovasabi/internal/repository"
)

var (
	ErrBroadcastNotFound = errors.New("broadcast not found")
	ErrBroadcastExists   = errors.New("broadcast already exists")
)

// Broadcast represents a broadcast message in the service_broadcast table
type Broadcast struct {
	ID          int64      `db:"id"`
	MasterID    int64      `db:"master_id"`
	Title       string     `db:"title"`
	Content     string     `db:"content"`
	Type        string     `db:"type"`
	Status      string     `db:"status"`
	ScheduledAt *time.Time `db:"scheduled_at"`
	SentAt      *time.Time `db:"sent_at"`
	Metadata    string     `db:"metadata"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// BroadcastRepository handles operations on the service_broadcast table
type BroadcastRepository struct {
	*repository.BaseRepository
	masterRepo repository.MasterRepository
}

// NewBroadcastRepository creates a new broadcast repository instance
func NewBroadcastRepository(db *sql.DB, masterRepo repository.MasterRepository) *BroadcastRepository {
	return &BroadcastRepository{
		BaseRepository: repository.NewBaseRepository(db),
		masterRepo:     masterRepo,
	}
}

// Create inserts a new broadcast record
func (r *BroadcastRepository) Create(ctx context.Context, broadcast *Broadcast) (*Broadcast, error) {
	masterID, err := r.masterRepo.Create(ctx, repository.EntityTypeBroadcast)
	if err != nil {
		return nil, err
	}

	broadcast.MasterID = masterID
	err = r.GetDB().QueryRowContext(ctx,
		`INSERT INTO service_broadcast (
			master_id, title, content, type, status,
			scheduled_at, sent_at, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
		) RETURNING id, created_at, updated_at`,
		broadcast.MasterID, broadcast.Title, broadcast.Content,
		broadcast.Type, broadcast.Status, broadcast.ScheduledAt,
		broadcast.SentAt, broadcast.Metadata,
	).Scan(&broadcast.ID, &broadcast.CreatedAt, &broadcast.UpdatedAt)

	if err != nil {
		_ = r.masterRepo.Delete(ctx, masterID)
		return nil, err
	}

	return broadcast, nil
}

// GetByID retrieves a broadcast by ID
func (r *BroadcastRepository) GetByID(ctx context.Context, id int64) (*Broadcast, error) {
	broadcast := &Broadcast{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, title, content, type,
			status, scheduled_at, sent_at, metadata,
			created_at, updated_at
		FROM service_broadcast 
		WHERE id = $1`,
		id,
	).Scan(
		&broadcast.ID, &broadcast.MasterID, &broadcast.Title,
		&broadcast.Content, &broadcast.Type, &broadcast.Status,
		&broadcast.ScheduledAt, &broadcast.SentAt, &broadcast.Metadata,
		&broadcast.CreatedAt, &broadcast.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrBroadcastNotFound
		}
		return nil, err
	}
	return broadcast, nil
}

// Update updates a broadcast record
func (r *BroadcastRepository) Update(ctx context.Context, broadcast *Broadcast) error {
	result, err := r.GetDB().ExecContext(ctx,
		`UPDATE service_broadcast 
		SET title = $1, content = $2, type = $3,
			status = $4, scheduled_at = $5, sent_at = $6,
			metadata = $7, updated_at = NOW()
		WHERE id = $8`,
		broadcast.Title, broadcast.Content, broadcast.Type,
		broadcast.Status, broadcast.ScheduledAt, broadcast.SentAt,
		broadcast.Metadata, broadcast.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrBroadcastNotFound
	}

	return nil
}

// Delete removes a broadcast and its master record
func (r *BroadcastRepository) Delete(ctx context.Context, id int64) error {
	broadcast, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return r.masterRepo.Delete(ctx, broadcast.MasterID)
}

// List retrieves a paginated list of broadcasts
func (r *BroadcastRepository) List(ctx context.Context, limit, offset int) ([]*Broadcast, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, title, content, type,
			status, scheduled_at, sent_at, metadata,
			created_at, updated_at
		FROM service_broadcast 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var broadcasts []*Broadcast
	for rows.Next() {
		broadcast := &Broadcast{}
		err := rows.Scan(
			&broadcast.ID, &broadcast.MasterID, &broadcast.Title,
			&broadcast.Content, &broadcast.Type, &broadcast.Status,
			&broadcast.ScheduledAt, &broadcast.SentAt, &broadcast.Metadata,
			&broadcast.CreatedAt, &broadcast.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		broadcasts = append(broadcasts, broadcast)
	}
	return broadcasts, rows.Err()
}

// ListPending retrieves a list of pending broadcasts
func (r *BroadcastRepository) ListPending(ctx context.Context) ([]*Broadcast, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, title, content, type,
			status, scheduled_at, sent_at, metadata,
			created_at, updated_at
		FROM service_broadcast 
		WHERE status = 'pending'
			AND (scheduled_at IS NULL OR scheduled_at <= NOW())
		ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var broadcasts []*Broadcast
	for rows.Next() {
		broadcast := &Broadcast{}
		err := rows.Scan(
			&broadcast.ID, &broadcast.MasterID, &broadcast.Title,
			&broadcast.Content, &broadcast.Type, &broadcast.Status,
			&broadcast.ScheduledAt, &broadcast.SentAt, &broadcast.Metadata,
			&broadcast.CreatedAt, &broadcast.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		broadcasts = append(broadcasts, broadcast)
	}
	return broadcasts, rows.Err()
}
