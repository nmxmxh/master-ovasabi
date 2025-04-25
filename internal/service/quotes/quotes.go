package quotes

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	quotespb "github.com/nmxmxh/master-ovasabi/api/protos/quotes"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrQuoteNotFound is returned when a quote cannot be found
	ErrQuoteNotFound = errors.New("quote not found")
	// ErrQuoteExists is returned when attempting to save a quote that already exists
	ErrQuoteExists = errors.New("quote already exists")
)

// ServiceImpl implements the QuotesService interface
type ServiceImpl struct {
	quotespb.UnimplementedQuotesServiceServer
	log    *zap.Logger
	mu     sync.RWMutex
	quotes map[string]*quotespb.GetQuoteResponse
}

// NewQuotesService creates a new instance of QuotesService
func NewQuotesService(log *zap.Logger) *ServiceImpl {
	return &ServiceImpl{
		log:    log,
		quotes: make(map[string]*quotespb.GetQuoteResponse),
	}
}

// GenerateQuote implements the GenerateQuote RPC method
func (s *ServiceImpl) GenerateQuote(ctx context.Context, req *quotespb.GenerateQuoteRequest) (*quotespb.GenerateQuoteResponse, error) {
	s.log.Info("Generating quote",
		zap.String("category", req.Category),
		zap.Any("parameters", req.Parameters))

	quoteID := "quote-" + time.Now().Format("20060102150405")
	quote := &quotespb.GetQuoteResponse{
		QuoteId:   quoteID,
		Content:   "This is a mock quote",
		Category:  req.Category,
		UserId:    "mock-user-id",
		Metadata:  req.Parameters,
		CreatedAt: time.Now().Unix(),
	}

	s.mu.Lock()
	s.quotes[quoteID] = quote
	s.mu.Unlock()

	return &quotespb.GenerateQuoteResponse{
		QuoteId:  quoteID,
		Content:  quote.Content,
		Category: quote.Category,
		Metadata: quote.Metadata,
	}, nil
}

// SaveQuote implements the SaveQuote RPC method
func (s *ServiceImpl) SaveQuote(ctx context.Context, req *quotespb.SaveQuoteRequest) (*quotespb.SaveQuoteResponse, error) {
	quoteID := "quote-" + time.Now().Format("20060102150405")
	s.log.Info("Saving quote",
		zap.String("quote_id", quoteID))

	s.mu.Lock()
	defer s.mu.Unlock()

	quote := &quotespb.GetQuoteResponse{
		QuoteId:   quoteID,
		Content:   req.Content,
		Category:  req.Category,
		UserId:    req.UserId,
		Metadata:  req.Metadata,
		CreatedAt: time.Now().Unix(),
	}

	s.quotes[quoteID] = quote

	return &quotespb.SaveQuoteResponse{
		QuoteId: quoteID,
		Message: "Quote saved successfully",
	}, nil
}

// GetQuote implements the GetQuote RPC method
func (s *ServiceImpl) GetQuote(ctx context.Context, req *quotespb.GetQuoteRequest) (*quotespb.GetQuoteResponse, error) {
	if req.QuoteId == "" {
		s.log.Error("Invalid quote ID",
			zap.Error(status.Error(codes.InvalidArgument, "quote_id cannot be empty")))
		return nil, status.Error(codes.InvalidArgument, "quote_id cannot be empty")
	}

	s.log.Info("Retrieving quote",
		zap.String("quote_id", req.QuoteId))

	s.mu.RLock()
	defer s.mu.RUnlock()

	if quote, exists := s.quotes[req.QuoteId]; exists {
		return quote, nil
	}

	return nil, ErrQuoteNotFound
}

func (s *ServiceImpl) CreateQuote(ctx context.Context, req *quotespb.CreateQuoteRequest) (*quotespb.CreateQuoteResponse, error) {
	if req.Text == "" {
		s.log.Error("Invalid quote text",
			zap.Error(status.Error(codes.InvalidArgument, "quote text cannot be empty")))
		return nil, status.Error(codes.InvalidArgument, "quote text cannot be empty")
	}

	if req.Author == "" {
		s.log.Error("Invalid quote author",
			zap.Error(status.Error(codes.InvalidArgument, "quote author cannot be empty")))
		return nil, status.Error(codes.InvalidArgument, "quote author cannot be empty")
	}

	quote := &quotespb.Quote{
		Id:        uuid.New().String(),
		Text:      req.Text,
		Author:    req.Author,
		CreatedAt: time.Now().Unix(),
	}

	s.log.Info("Created new quote",
		zap.String("quote_id", quote.Id),
		zap.String("author", quote.Author))

	return &quotespb.CreateQuoteResponse{
		Quote: quote,
	}, nil
}
