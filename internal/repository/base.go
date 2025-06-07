package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/nmxmxh/master-ovasabi/pkg/json"
	"go.uber.org/zap"
)

// BaseRepository provides common database functionality.
type BaseRepository struct {
	db  *sql.DB
	log *zap.Logger
}

// NewBaseRepository creates a new base repository instance.
func NewBaseRepository(db *sql.DB, log *zap.Logger) *BaseRepository {
	return &BaseRepository{
		db:  db,
		log: log,
	}
}

// GetDB returns the underlying database connection.
func (r *BaseRepository) GetDB() *sql.DB {
	return r.db
}

// GetLogger returns the logger instance.
func (r *BaseRepository) GetLogger() *zap.Logger {
	return r.log
}

// BeginTx starts a new transaction with context.
func (r *BaseRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		r.log = r.log.With(zap.String("request_id", requestID))
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		if r.log != nil {
			if requestID, ok := ctx.Value("request_id").(string); ok {
				r.log.Error("Failed to begin transaction",
					zap.Error(err),
					zap.String("request_id", requestID),
				)
			} else {
				r.log.Error("Failed to begin transaction",
					zap.Error(err),
				)
			}
		}
		return nil, err
	}
	return tx, nil
}

// CommitTx commits a transaction with context.
func (r *BaseRepository) CommitTx(ctx context.Context, tx *sql.Tx) error {
	if err := tx.Commit(); err != nil {
		if r.log != nil {
			if requestID, ok := ctx.Value("request_id").(string); ok {
				r.log.Error("Failed to commit transaction",
					zap.Error(err),
					zap.String("request_id", requestID),
				)
			} else {
				r.log.Error("Failed to commit transaction",
					zap.Error(err),
				)
			}
		}
		return err
	}
	return nil
}

// RollbackTx rolls back a transaction with context.
func (r *BaseRepository) RollbackTx(ctx context.Context, tx *sql.Tx) error {
	if err := tx.Rollback(); err != nil {
		if r.log != nil {
			if requestID, ok := ctx.Value("request_id").(string); ok {
				r.log.Error("Failed to rollback transaction",
					zap.Error(err),
					zap.String("request_id", requestID),
				)
			} else {
				r.log.Error("Failed to rollback transaction",
					zap.Error(err),
				)
			}
		}
		return err
	}
	return nil
}

// GetContext returns the context with transaction if present.
func (r *BaseRepository) GetContext(ctx context.Context) context.Context {
	return ctx
}

// WithTx returns a new repository instance with the given transaction.
// WithTx returns a new repository with transaction.
func (r *BaseRepository) WithTx(_ *sql.Tx) Repository {
	return &BaseRepository{
		db:  r.db,
		log: r.log,
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

// ToJSONB marshals a map to JSONB ([]byte) for Postgres.
func ToJSONB(m map[string]interface{}) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

// FromJSONB unmarshals JSONB ([]byte) from Postgres to a map.
func FromJSONB(b []byte) (map[string]interface{}, error) {
	if len(b) == 0 {
		return map[string]interface{}{}, nil
	}
	var m map[string]interface{}
	err := json.Unmarshal(b, &m)
	return m, err
}
