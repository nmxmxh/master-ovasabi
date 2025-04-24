package quotes

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nmxmxh/master-ovasabi/api/protos/quotes"
	"go.uber.org/zap"
)

var (
	// ErrQuoteNotFound is returned when a quote cannot be found
	ErrQuoteNotFound = errors.New("quote not found")
	// ErrQuoteExists is returned when attempting to save a quote that already exists
	ErrQuoteExists = errors.New("quote already exists")
)

// ServiceImpl implements the QuotesService interface
type ServiceImpl struct {
	quotes.UnimplementedQuotesServiceServer
	log    *zap.Logger
	mu     sync.RWMutex
	quotes map[string]*quotes.GetQuoteResponse
}

// NewQuotesService creates a new instance of QuotesService
func NewQuotesService(log *zap.Logger) *ServiceImpl {
	return &ServiceImpl{
		log:    log,
		quotes: make(map[string]*quotes.GetQuoteResponse),
	}
}

// GenerateQuote implements the GenerateQuote RPC method
func (s *ServiceImpl) GenerateQuote(ctx context.Context, req *quotes.GenerateQuoteRequest) (*quotes.GenerateQuoteResponse, error) {
	s.log.Info("Generating quote",
		zap.String("category", req.Category),
		zap.Any("parameters", req.Parameters))

	quoteID := "quote-" + time.Now().Format("20060102150405")
	quote := &quotes.GetQuoteResponse{
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

	return &quotes.GenerateQuoteResponse{
		QuoteId:  quoteID,
		Content:  quote.Content,
		Category: quote.Category,
		Metadata: quote.Metadata,
	}, nil
}

// SaveQuote implements the SaveQuote RPC method
func (s *ServiceImpl) SaveQuote(ctx context.Context, req *quotes.SaveQuoteRequest) (*quotes.SaveQuoteResponse, error) {
	quoteID := "quote-" + time.Now().Format("20060102150405")
	s.log.Info("Saving quote",
		zap.String("quote_id", quoteID))

	s.mu.Lock()
	defer s.mu.Unlock()

	quote := &quotes.GetQuoteResponse{
		QuoteId:   quoteID,
		Content:   req.Content,
		Category:  req.Category,
		UserId:    req.UserId,
		Metadata:  req.Metadata,
		CreatedAt: time.Now().Unix(),
	}

	s.quotes[quoteID] = quote

	return &quotes.SaveQuoteResponse{
		QuoteId: quoteID,
		Message: "Quote saved successfully",
	}, nil
}

// GetQuote implements the GetQuote RPC method
func (s *ServiceImpl) GetQuote(ctx context.Context, req *quotes.GetQuoteRequest) (*quotes.GetQuoteResponse, error) {
	s.log.Info("Retrieving quote",
		zap.String("quote_id", req.QuoteId))

	s.mu.RLock()
	defer s.mu.RUnlock()

	if quote, exists := s.quotes[req.QuoteId]; exists {
		return quote, nil
	}

	return nil, ErrQuoteNotFound
}
