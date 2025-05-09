package quotes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	quotespb "github.com/nmxmxh/master-ovasabi/api/protos/quotes/v1"
	quotesrepo "github.com/nmxmxh/master-ovasabi/internal/repository/quotes"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	// ErrQuoteNotFound is returned when a quote cannot be found.
	ErrQuoteNotFound = errors.New("quote not found")
	// ErrQuoteExists is returned when attempting to save a quote that already exists.
	ErrQuoteExists = errors.New("quote already exists")
)

// TTL constants for quotes caching
const (
	TTLQuote     = 30 * time.Minute
	TTLQuoteList = 5 * time.Minute
)

// SafeInt32 converts an int to int32 with overflow checking.
func SafeInt32(i int) (int32, error) {
	if i > math.MaxInt32 || i < math.MinInt32 {
		return 0, fmt.Errorf("integer overflow: value %d out of int32 range", i)
	}
	return int32(i), nil
}

// ServiceImpl implements the QuotesService interface.
type ServiceImpl struct {
	quotespb.UnimplementedQuotesServiceServer
	log   *zap.Logger
	db    *quotesrepo.QuoteRepository
	cache *redis.Cache
}

// NewQuotesService creates a new instance of QuotesService.
func NewQuotesService(log *zap.Logger, db *quotesrepo.QuoteRepository, cache *redis.Cache) quotespb.QuotesServiceServer {
	return &ServiceImpl{
		log:   log,
		db:    db,
		cache: cache,
	}
}

func (s *ServiceImpl) CreateQuote(ctx context.Context, req *quotespb.CreateQuoteRequest) (*quotespb.CreateQuoteResponse, error) {
	s.log.Info("Creating quote",
		zap.Int32("master_id", req.MasterId),
		zap.Int32("campaign_id", req.CampaignId))

	tx, err := s.db.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.log.Warn("failed to rollback tx", zap.Error(err))
		}
	}()

	// Marshal metadata to JSONB
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal metadata: %v", err)
	}

	// Create service_quote record
	var quote quotespb.BillingQuote
	var createdAt time.Time
	err = tx.QueryRowContext(ctx,
		`INSERT INTO service_quote 
		(master_id, campaign_id, description, author, metadata, created_at) 
		VALUES ($1, $2, $3, $4, $5, NOW()) 
		RETURNING id, master_id, campaign_id, description, author, created_at`,
		req.MasterId, req.CampaignId, req.Description, req.Author, metadata).
		Scan(&quote.Id, &quote.MasterId, &quote.CampaignId, &quote.Description, &quote.Author, &createdAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create quote: %v", err)
	}

	// Set the created_at timestamp
	quote.CreatedAt = timestamppb.New(createdAt)
	quote.Metadata = req.Metadata

	// Log event
	_, err = tx.ExecContext(ctx,
		`INSERT INTO service_event 
		(master_id, event_type, payload) 
		VALUES ($1, 'quote_created', $2)`,
		req.MasterId, metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to log event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	// Invalidate related caches
	cacheKeys := []string{
		fmt.Sprintf("quote:%d", quote.Id),
		fmt.Sprintf("quotes:campaign:%d:page:0", req.CampaignId),
	}

	for _, key := range cacheKeys {
		if err := s.cache.Delete(ctx, key, ""); err != nil {
			s.log.Error("Failed to invalidate cache",
				zap.String("cache_key", key),
				zap.Error(err))
			// Continue even if cache invalidation fails
		}
	}

	return &quotespb.CreateQuoteResponse{
		Quote:   &quote,
		Success: true,
	}, nil
}

func (s *ServiceImpl) GetQuote(ctx context.Context, req *quotespb.GetQuoteRequest) (*quotespb.GetQuoteResponse, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("quote:%d", req.QuoteId)
	var quote quotespb.BillingQuote
	if err := s.cache.Get(ctx, cacheKey, "", &quote); err == nil {
		return &quotespb.GetQuoteResponse{
			Quote: &quote,
		}, nil
	}

	var metadataBytes []byte
	var createdAt time.Time

	err := s.db.GetDB().QueryRowContext(ctx, `
		SELECT id, master_id, campaign_id, description, author, metadata, amount, currency, created_at
		FROM service_quote
		WHERE id = $1`,
		req.QuoteId).
		Scan(&quote.Id, &quote.MasterId, &quote.CampaignId, &quote.Description, &quote.Author,
			&metadataBytes, &quote.Amount, &quote.Currency, &createdAt)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "quote not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	// Parse metadata
	if err := json.Unmarshal(metadataBytes, &quote.Metadata); err != nil {
		s.log.Warn("failed to unmarshal quote metadata",
			zap.Int32("quote_id", quote.Id),
			zap.Error(err))
	}

	quote.CreatedAt = timestamppb.New(createdAt)

	// Cache the quote
	if err := s.cache.Set(ctx, cacheKey, "", &quote, TTLQuote); err != nil {
		s.log.Error("Failed to cache quote",
			zap.Int32("quote_id", quote.Id),
			zap.Error(err))
		// Don't fail the get if caching fails
	}

	return &quotespb.GetQuoteResponse{
		Quote: &quote,
	}, nil
}

func (s *ServiceImpl) ListQuotes(ctx context.Context, req *quotespb.ListQuotesRequest) (*quotespb.ListQuotesResponse, error) {
	// Try to get from cache first
	cacheKey := fmt.Sprintf("quotes:campaign:%d:page:%d", req.CampaignId, req.Page)
	var response quotespb.ListQuotesResponse
	if err := s.cache.Get(ctx, cacheKey, "", &response); err == nil {
		return &response, nil
	}

	query := `
		SELECT id, master_id, campaign_id, description, author, metadata, amount, currency, created_at,
		       COUNT(*) OVER() as total_count
		FROM service_quote
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply campaign filter
	if req.CampaignId != 0 {
		query += fmt.Sprintf(" AND campaign_id = $%d", argPos)
		args = append(args, req.CampaignId)
		argPos++
	}

	// Add pagination
	pageSize := int32(10)
	if req.PageSize > 0 {
		pageSize = req.PageSize
	}
	offset := req.Page * pageSize

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.log.Warn("failed to close rows", zap.Error(err))
		}
	}()

	var quotes []*quotespb.BillingQuote
	var totalCount int32

	for rows.Next() {
		var quote quotespb.BillingQuote
		var metadataBytes []byte
		var createdAt time.Time

		err := rows.Scan(
			&quote.Id,
			&quote.MasterId,
			&quote.CampaignId,
			&quote.Description,
			&quote.Author,
			&metadataBytes,
			&quote.Amount,
			&quote.Currency,
			&createdAt,
			&totalCount,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan quote: %v", err)
		}

		// Parse metadata
		if err := json.Unmarshal(metadataBytes, &quote.Metadata); err != nil {
			s.log.Warn("failed to unmarshal quote metadata",
				zap.Int32("quote_id", quote.Id),
				zap.Error(err))
		}

		quote.CreatedAt = timestamppb.New(createdAt)
		quotes = append(quotes, &quote)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	response = quotespb.ListQuotesResponse{
		Quotes:     quotes,
		TotalCount: totalCount,
		Page:       req.Page,
		TotalPages: totalPages,
	}

	// Cache the response
	if err := s.cache.Set(ctx, cacheKey, "", &response, TTLQuoteList); err != nil {
		s.log.Error("Failed to cache quotes list",
			zap.Int32("campaign_id", req.CampaignId),
			zap.Int32("page", req.Page),
			zap.Error(err))
		// Don't fail the list if caching fails
	}

	return &response, nil
}
