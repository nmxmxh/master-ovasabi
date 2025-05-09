package dbiface

import (
	"context"
	"database/sql"
)

type Tx interface {
	Commit() error
	Rollback() error
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type DB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}
