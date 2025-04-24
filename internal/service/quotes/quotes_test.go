package quotes

import (
	"context"
	"testing"
	"time"

	"github.com/ovasabi/master-ovasabi/api/protos/quotes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates service with dependencies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewQuotesService(logger)

			assert.NotNil(t, svc)
			assert.NotNil(t, svc.log)
			assert.NotNil(t, svc.quotes)
		})
	}
}

func TestService_GenerateQuote(t *testing.T) {
	tests := []struct {
		name          string
		request       *quotes.GenerateQuoteRequest
		expectedError error
	}{
		{
			name: "successful quote generation",
			request: &quotes.GenerateQuoteRequest{
				Category: "test-category",
				Parameters: map[string]string{
					"key": "value",
				},
			},
			expectedError: nil,
		},
		{
			name: "empty category",
			request: &quotes.GenerateQuoteRequest{
				Category: "",
				Parameters: map[string]string{
					"key": "value",
				},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewQuotesService(zap.NewNop())

			resp, err := svc.GenerateQuote(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.QuoteId)
				assert.NotEmpty(t, resp.Content)
				assert.Equal(t, tt.request.Category, resp.Category)
				assert.Equal(t, tt.request.Parameters, resp.Metadata)
			}
		})
	}
}

func TestService_SaveQuote(t *testing.T) {
	tests := []struct {
		name          string
		request       *quotes.SaveQuoteRequest
		expectedError error
	}{
		{
			name: "successful quote save",
			request: &quotes.SaveQuoteRequest{
				Content:  "test quote",
				Category: "test-category",
				UserId:   "test-user",
				Metadata: map[string]string{
					"key": "value",
				},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewQuotesService(zap.NewNop())

			resp, err := svc.SaveQuote(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, resp.QuoteId)
				assert.Equal(t, "Quote saved successfully", resp.Message)

				// Verify the quote was saved
				saved, err := svc.GetQuote(context.Background(), &quotes.GetQuoteRequest{
					QuoteId: resp.QuoteId,
				})
				require.NoError(t, err)
				assert.Equal(t, tt.request.Content, saved.Content)
				assert.Equal(t, tt.request.Category, saved.Category)
				assert.Equal(t, tt.request.UserId, saved.UserId)
				assert.Equal(t, tt.request.Metadata, saved.Metadata)
				assert.NotZero(t, saved.CreatedAt)
			}
		})
	}
}

func TestService_GetQuote(t *testing.T) {
	tests := []struct {
		name          string
		setupQuote    *quotes.SaveQuoteRequest
		request       *quotes.GetQuoteRequest
		expectedError error
	}{
		{
			name: "successful quote retrieval",
			setupQuote: &quotes.SaveQuoteRequest{
				Content:  "test quote",
				Category: "test-category",
				UserId:   "test-user",
				Metadata: map[string]string{
					"key": "value",
				},
			},
			request:       &quotes.GetQuoteRequest{}, // QuoteId will be set after saving
			expectedError: nil,
		},
		{
			name:       "quote not found",
			setupQuote: nil,
			request: &quotes.GetQuoteRequest{
				QuoteId: "nonexistent",
			},
			expectedError: ErrQuoteNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewQuotesService(zap.NewNop())

			var quoteID string
			if tt.setupQuote != nil {
				// Save a quote first
				resp, err := svc.SaveQuote(context.Background(), tt.setupQuote)
				require.NoError(t, err)
				quoteID = resp.QuoteId
				tt.request.QuoteId = quoteID
			}

			resp, err := svc.GetQuote(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				assert.Equal(t, quoteID, resp.QuoteId)
				assert.Equal(t, tt.setupQuote.Content, resp.Content)
				assert.Equal(t, tt.setupQuote.Category, resp.Category)
				assert.Equal(t, tt.setupQuote.UserId, resp.UserId)
				assert.Equal(t, tt.setupQuote.Metadata, resp.Metadata)
				assert.NotZero(t, resp.CreatedAt)
				assert.True(t, resp.CreatedAt <= time.Now().Unix())
			}
		})
	}
}
