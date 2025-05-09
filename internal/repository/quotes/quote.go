package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"go.uber.org/zap"
)

var (
	ErrQuoteNotFound = errors.New("quote not found")
	ErrQuoteExists   = errors.New("quote already exists")
)

var log *zap.Logger

func SetLogger(l *zap.Logger) {
	log = l
}

// Quote represents a financial quote in the service_quote table
type Quote struct {
	ID        int64     `db:"id"`
	MasterID  int64     `db:"master_id"`
	Symbol    string    `db:"symbol"`
	Price     float64   `db:"price"`
	Volume    int64     `db:"volume"`
	Metadata  string    `db:"metadata"`
	Timestamp time.Time `db:"timestamp"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// QuoteRepository handles operations on the service_quote table
type QuoteRepository struct {
	*repository.BaseRepository
	masterRepo repository.MasterRepository
}

// NewQuoteRepository creates a new quote repository instance
func NewQuoteRepository(db *sql.DB, masterRepo repository.MasterRepository) *QuoteRepository {
	return &QuoteRepository{
		BaseRepository: repository.NewBaseRepository(db),
		masterRepo:     masterRepo,
	}
}

// Create inserts a new quote record
func (r *QuoteRepository) Create(ctx context.Context, quote *Quote) (*Quote, error) {
	// Generate a descriptive name for the master record
	masterName := r.GenerateMasterName(repository.EntityTypeQuote,
		quote.Symbol,
		fmt.Sprintf("%.2f", quote.Price))

	masterID, err := r.masterRepo.Create(ctx, repository.EntityTypeQuote, masterName)
	if err != nil {
		return nil, err
	}

	quote.MasterID = masterID
	err = r.GetDB().QueryRowContext(ctx,
		`INSERT INTO service_quote (
			master_id, symbol, price, volume, metadata,
			timestamp, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, NOW(), NOW()
		) RETURNING id, created_at, updated_at`,
		quote.MasterID, quote.Symbol, quote.Price,
		quote.Volume, quote.Metadata, quote.Timestamp,
	).Scan(&quote.ID, &quote.CreatedAt, &quote.UpdatedAt)

	if err != nil {
		// If quote creation fails, clean up the master record
		_ = r.masterRepo.Delete(ctx, masterID)
		return nil, err
	}

	return quote, nil
}

// GetBySymbol retrieves the latest quote for a symbol
func (r *QuoteRepository) GetBySymbol(ctx context.Context, symbol string) (*Quote, error) {
	quote := &Quote{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, symbol, price, volume,
			metadata, timestamp, created_at, updated_at
		FROM service_quote 
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT 1`,
		symbol,
	).Scan(
		&quote.ID, &quote.MasterID, &quote.Symbol,
		&quote.Price, &quote.Volume, &quote.Metadata,
		&quote.Timestamp, &quote.CreatedAt, &quote.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrQuoteNotFound
		}
		return nil, err
	}
	return quote, nil
}

// GetByID retrieves a quote by ID
func (r *QuoteRepository) GetByID(ctx context.Context, id int64) (*Quote, error) {
	quote := &Quote{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, symbol, price, volume,
			metadata, timestamp, created_at, updated_at
		FROM service_quote 
		WHERE id = $1`,
		id,
	).Scan(
		&quote.ID, &quote.MasterID, &quote.Symbol,
		&quote.Price, &quote.Volume, &quote.Metadata,
		&quote.Timestamp, &quote.CreatedAt, &quote.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrQuoteNotFound
		}
		return nil, err
	}
	return quote, nil
}

// ListBySymbol retrieves a paginated list of quotes for a symbol
func (r *QuoteRepository) ListBySymbol(ctx context.Context, symbol string, limit, offset int) ([]*Quote, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, symbol, price, volume,
			metadata, timestamp, created_at, updated_at
		FROM service_quote 
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3`,
		symbol, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			if log != nil {
				log.Error("error closing rows", zap.Error(cerr))
			}
		}
	}()

	var quotes []*Quote
	for rows.Next() {
		quote := &Quote{}
		err := rows.Scan(
			&quote.ID, &quote.MasterID, &quote.Symbol,
			&quote.Price, &quote.Volume, &quote.Metadata,
			&quote.Timestamp, &quote.CreatedAt, &quote.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		quotes = append(quotes, quote)
	}
	return quotes, rows.Err()
}

// GetQuoteHistory retrieves quotes for a symbol within a time range
func (r *QuoteRepository) GetQuoteHistory(ctx context.Context, symbol string, start, end time.Time) ([]*Quote, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, symbol, price, volume,
			metadata, timestamp, created_at, updated_at
		FROM service_quote 
		WHERE symbol = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC`,
		symbol, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			if log != nil {
				log.Error("error closing rows", zap.Error(cerr))
			}
		}
	}()

	var quotes []*Quote
	for rows.Next() {
		quote := &Quote{}
		err := rows.Scan(
			&quote.ID, &quote.MasterID, &quote.Symbol,
			&quote.Price, &quote.Volume, &quote.Metadata,
			&quote.Timestamp, &quote.CreatedAt, &quote.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		quotes = append(quotes, quote)
	}
	return quotes, rows.Err()
}

// Delete removes a quote and its master record
func (r *QuoteRepository) Delete(ctx context.Context, id int64) error {
	quote, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// The master record deletion will cascade to the quote due to foreign key
	return r.masterRepo.Delete(ctx, quote.MasterID)
}

// GetLatestQuotes retrieves the latest quotes for multiple symbols
func (r *QuoteRepository) GetLatestQuotes(ctx context.Context, symbols []string) ([]*Quote, error) {
	query := `
		WITH RankedQuotes AS (
			SELECT 
				id, master_id, symbol, price, volume,
				metadata, timestamp, created_at, updated_at,
				ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY timestamp DESC) as rn
			FROM service_quote 
			WHERE symbol = ANY($1)
		)
		SELECT 
			id, master_id, symbol, price, volume,
			metadata, timestamp, created_at, updated_at
		FROM RankedQuotes 
		WHERE rn = 1
		ORDER BY symbol`

	rows, err := r.GetDB().QueryContext(ctx, query, pq.Array(symbols))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			if log != nil {
				log.Error("error closing rows", zap.Error(cerr))
			}
		}
	}()

	var quotes []*Quote
	for rows.Next() {
		quote := &Quote{}
		err := rows.Scan(
			&quote.ID, &quote.MasterID, &quote.Symbol,
			&quote.Price, &quote.Volume, &quote.Metadata,
			&quote.Timestamp, &quote.CreatedAt, &quote.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		quotes = append(quotes, quote)
	}
	return quotes, rows.Err()
}
