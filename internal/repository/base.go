package repository

import (
	"context"
	"database/sql"
)

// BaseRepository provides common database functionality
type BaseRepository struct {
	db *sql.DB
}

// NewBaseRepository creates a new base repository instance
func NewBaseRepository(db *sql.DB) *BaseRepository {
	return &BaseRepository{
		db: db,
	}
}

// GetDB returns the database connection
func (r *BaseRepository) GetDB() *sql.DB {
	return r.db
}

// BeginTx starts a new transaction
func (r *BaseRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

// CommitTx commits a transaction
func (r *BaseRepository) CommitTx(tx *sql.Tx) error {
	return tx.Commit()
}

// RollbackTx rolls back a transaction
func (r *BaseRepository) RollbackTx(tx *sql.Tx) error {
	return tx.Rollback()
}

// GetContext returns the context, possibly with transaction
func (r *BaseRepository) GetContext(ctx context.Context) context.Context {
	return ctx
}

// WithTx returns a new repository with transaction
func (r *BaseRepository) WithTx(tx *sql.Tx) Repository {
	return &BaseRepository{
		db: r.db,
	}
}
