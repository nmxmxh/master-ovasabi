package i18n

import (
	"context"

	"github.com/ovasabi/master-ovasabi/api/protos/i18n"
	"go.uber.org/zap"
)

// ServiceImpl implements the I18nService interface
type ServiceImpl struct {
	i18n.UnimplementedI18NServiceServer
	log              *zap.Logger
	supportedLocales []string
	defaultLocale    string
}

// NewService creates a new instance of I18nService
func NewService(log *zap.Logger) *ServiceImpl {
	return &ServiceImpl{
		log:              log,
		supportedLocales: []string{"en", "es", "fr"},
		defaultLocale:    "en",
	}
}

// GetTranslation implements the GetTranslation RPC method
func (s *ServiceImpl) GetTranslation(ctx context.Context, req *i18n.GetTranslationRequest) (*i18n.GetTranslationResponse, error) {
	// TODO: Implement proper translation lookup
	// For now, just return the key as the translation
	return &i18n.GetTranslationResponse{
		Text:   req.Key,
		Locale: req.Locale,
	}, nil
}

// GetSupportedLocales implements the GetSupportedLocales RPC method
func (s *ServiceImpl) GetSupportedLocales(ctx context.Context, req *i18n.GetSupportedLocalesRequest) (*i18n.GetSupportedLocalesResponse, error) {
	return &i18n.GetSupportedLocalesResponse{
		Locales:       s.supportedLocales,
		DefaultLocale: s.defaultLocale,
	}, nil
}
