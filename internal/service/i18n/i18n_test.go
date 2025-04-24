package i18n

import (
	"context"
	"testing"

	"github.com/nmxmxh/master-ovasabi/api/protos/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name           string
		expectedConfig *ServiceImpl
	}{
		{
			name: "creates service with default configuration",
			expectedConfig: &ServiceImpl{
				supportedLocales: []string{"en", "es", "fr"},
				defaultLocale:    "en",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewService(logger)

			assert.Equal(t, tt.expectedConfig.supportedLocales, svc.supportedLocales)
			assert.Equal(t, tt.expectedConfig.defaultLocale, svc.defaultLocale)
			assert.NotNil(t, svc.log)
		})
	}
}

func TestService_GetTranslation(t *testing.T) {
	tests := []struct {
		name          string
		request       *i18n.GetTranslationRequest
		expectedResp  *i18n.GetTranslationResponse
		expectedError error
	}{
		{
			name: "successful translation lookup",
			request: &i18n.GetTranslationRequest{
				Key:    "test.key",
				Locale: "en",
				Params: map[string]string{
					"param1": "value1",
				},
			},
			expectedResp: &i18n.GetTranslationResponse{
				Text:   "test.key",
				Locale: "en",
			},
			expectedError: nil,
		},
		{
			name: "empty key returns key as translation",
			request: &i18n.GetTranslationRequest{
				Key:    "",
				Locale: "en",
			},
			expectedResp: &i18n.GetTranslationResponse{
				Text:   "",
				Locale: "en",
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(zap.NewNop())

			resp, err := svc.GetTranslation(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResp.Text, resp.Text)
				assert.Equal(t, tt.expectedResp.Locale, resp.Locale)
			}
		})
	}
}

func TestService_GetSupportedLocales(t *testing.T) {
	tests := []struct {
		name          string
		request       *i18n.GetSupportedLocalesRequest
		expectedResp  *i18n.GetSupportedLocalesResponse
		expectedError error
	}{
		{
			name:    "returns all supported locales",
			request: &i18n.GetSupportedLocalesRequest{},
			expectedResp: &i18n.GetSupportedLocalesResponse{
				Locales:       []string{"en", "es", "fr"},
				DefaultLocale: "en",
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(zap.NewNop())

			resp, err := svc.GetSupportedLocales(context.Background(), tt.request)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResp.Locales, resp.Locales)
				assert.Equal(t, tt.expectedResp.DefaultLocale, resp.DefaultLocale)
			}
		})
	}
}
