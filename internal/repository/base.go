package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// BaseRepository provides common database functionality.
type BaseRepository struct {
	db *sql.DB
}

// NewBaseRepository creates a new base repository instance.
func NewBaseRepository(db *sql.DB) *BaseRepository {
	return &BaseRepository{
		db: db,
	}
}

// GetDB returns the database connection.
func (r *BaseRepository) GetDB() *sql.DB {
	return r.db
}

// BeginTx starts a new transaction.
func (r *BaseRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

// CommitTx commits a transaction.
func (r *BaseRepository) CommitTx(tx *sql.Tx) error {
	return tx.Commit()
}

// RollbackTx rolls back a transaction.
func (r *BaseRepository) RollbackTx(tx *sql.Tx) error {
	return tx.Rollback()
}

// GetContext returns the context, possibly with transaction.
func (r *BaseRepository) GetContext(ctx context.Context) context.Context {
	return ctx
}

// WithTx returns a new repository with transaction.
func (r *BaseRepository) WithTx(_ *sql.Tx) Repository {
	return &BaseRepository{
		db: r.db,
	}
}

// GenerateMasterName creates a standardized name for master records.
func (r *BaseRepository) GenerateMasterName(entityType EntityType, identifiers ...string) string {
	// Clean and join identifiers
	cleaned := make([]string, 0, len(identifiers))
	for _, id := range identifiers {
		if id = strings.TrimSpace(id); id != "" {
			cleaned = append(cleaned, id)
		}
	}

	// If no valid identifiers, use timestamp
	if len(cleaned) == 0 {
		cleaned = append(cleaned, time.Now().Format("20060102-150405"))
	}

	// Format: type:identifier1:identifier2...
	return fmt.Sprintf("%s:%s", entityType, strings.Join(cleaned, ":"))
}
