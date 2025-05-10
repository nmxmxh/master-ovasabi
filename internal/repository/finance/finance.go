package finance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var ErrTransactionNotFound = errors.New("transaction not found")

// Repository defines the interface for finance operations.
type Repository interface {
	GetBalance(ctx context.Context, userID uuid.UUID) (float64, error)
	UpdateBalance(ctx context.Context, userID uuid.UUID, amount float64) error
	CreateTransaction(ctx context.Context, tx *TransactionModel) error
	ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*TransactionModel, error)
	GetTransaction(ctx context.Context, id uuid.UUID) (*TransactionModel, error)
}

// TransactionModel represents a financial transaction in the database.
type TransactionModel struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	ToUserID    uuid.UUID
	Type        string
	Amount      float64
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// Transaction represents a financial transaction.
type Transaction struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	ToUserID    uuid.UUID
	Type        string
	Amount      float64
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   sql.NullTime
}

type repository struct {
	db  *sql.DB
	log *zap.Logger
}

// New creates a new finance repository.
func New(db *sql.DB, log *zap.Logger) Repository {
	return &repository{
		db:  db,
		log: log,
	}
}

// GetBalance retrieves the current balance for a user.
func (r *repository) GetBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	var balance float64
	err := r.db.QueryRowContext(ctx, `
		SELECT balance
		FROM user_balances
		WHERE user_id = $1
	`, userID).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}
	return balance, nil
}

// UpdateBalance updates a user's balance.
func (r *repository) UpdateBalance(ctx context.Context, userID uuid.UUID, amount float64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE user_balances
		SET balance = balance + $1
		WHERE user_id = $2
	`, amount, userID)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		_, err = r.db.ExecContext(ctx, `
			INSERT INTO user_balances (user_id, balance)
			VALUES ($1, $2)
		`, userID, amount)
		if err != nil {
			return fmt.Errorf("failed to create balance: %w", err)
		}
	}
	return nil
}

// CreateTransaction creates a new financial transaction.
func (r *repository) CreateTransaction(ctx context.Context, tx *TransactionModel) error {
	query := `
		INSERT INTO transactions (id, user_id, to_user_id, type, amount, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		tx.ID, tx.UserID, tx.ToUserID, tx.Type, tx.Amount, tx.Description, tx.Status,
		tx.CreatedAt, tx.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	return nil
}

// GetTransaction retrieves a transaction by ID.
func (r *repository) GetTransaction(ctx context.Context, id uuid.UUID) (*TransactionModel, error) {
	tx := &TransactionModel{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, to_user_id, type, amount, description, status, created_at, updated_at, deleted_at
		FROM transactions 
		WHERE id = $1
	`, id).Scan(
		&tx.ID, &tx.UserID, &tx.ToUserID, &tx.Type, &tx.Amount, &tx.Description,
		&tx.Status, &tx.CreatedAt, &tx.UpdatedAt, &tx.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrTransactionNotFound
	}
	return tx, err
}

// ListTransactions lists transactions for a user with pagination.
func (r *repository) ListTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*TransactionModel, error) {
	query := `
		SELECT id, user_id, to_user_id, type, amount, description, status, created_at, updated_at, deleted_at
		FROM transactions
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			r.log.Error("failed to close rows", zap.Error(err))
		}
	}()

	var transactions []*TransactionModel
	for rows.Next() {
		transaction := &TransactionModel{}
		err := rows.Scan(
			&transaction.ID, &transaction.UserID, &transaction.ToUserID, &transaction.Type,
			&transaction.Amount, &transaction.Description, &transaction.Status,
			&transaction.CreatedAt, &transaction.UpdatedAt, &transaction.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return transactions, nil
}

// LockBalance locks an amount in a user's balance.
func (r *repository) LockBalance(ctx context.Context, userID uuid.UUID, amount float64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE user_balances 
		SET balance = balance - $2,
		    locked_amount = locked_amount + $2
		WHERE user_id = $1 AND balance >= $2
	`, userID, amount)
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
	return nil
}

// UnlockBalance unlocks a previously locked amount in a user's balance.
func (r *repository) UnlockBalance(ctx context.Context, userID uuid.UUID, amount float64) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE user_balances 
		SET locked_amount = locked_amount - $2
		WHERE user_id = $1 AND locked_amount >= $2
	`, userID, amount)
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
	return nil
}
