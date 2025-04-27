// dbwrapper.go: Implements the DB and Tx interfaces for dependency injection
package service

import (
	"context"
	"database/sql"

	"github.com/nmxmxh/master-ovasabi/internal/shared/dbiface"
)

// Exported so all services can use it.
type Tx interface {
	Commit() error
	Rollback() error
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// Exported so all services can use it.
type DB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

type sqlDBWrapper struct {
	db *sql.DB
}

type sqlTxWrapper struct {
	tx *sql.Tx
}

// --- Tx interface implementation ---.
func (t *sqlTxWrapper) Commit() error   { return t.tx.Commit() }
func (t *sqlTxWrapper) Rollback() error { return t.tx.Rollback() }
func (t *sqlTxWrapper) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

func (t *sqlTxWrapper) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

// --- DB interface implementation ---.
func (d *sqlDBWrapper) BeginTx(ctx context.Context, opts *sql.TxOptions) (dbiface.Tx, error) {
	tx, err := d.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &sqlTxWrapper{tx: tx}, nil
}

func (d *sqlDBWrapper) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

func (d *sqlDBWrapper) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

func (d *sqlDBWrapper) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}
