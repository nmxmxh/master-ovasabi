package repository

import (
	"context"
	"database/sql"

	"go.uber.org/zap"
)

// TxFn represents a function that will be executed within a transaction.
type TxFn func(*sql.Tx) error

// WithTransaction executes the given function within a transaction.
func WithTransaction(ctx context.Context, db *sql.DB, log *zap.Logger, fn TxFn) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil {
				if log != nil {
					log.Error("transaction rollback failed during panic recovery", zap.Error(err))
				}
			}
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit()
}

// DBTX represents a database connection that can execute queries or a transaction.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
