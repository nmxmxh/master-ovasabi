package i18n

import (
	"context"

	"github.com/nmxmxh/master-ovasabi/api/protos/i18n/v0"
	i18nrepo "github.com/nmxmxh/master-ovasabi/internal/repository/i18n"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ServiceImpl implements the I18nService interface.
type ServiceImpl struct {
	i18n.UnimplementedI18NServiceServer
	log              *zap.Logger
	cache            *redis.Cache
	repo             *i18nrepo.Repository
	supportedLocales []string
	defaultLocale    string
}

// NewService creates a new instance of I18nService.
func NewService(log *zap.Logger, repo *i18nrepo.Repository, cache *redis.Cache) *ServiceImpl {
	return &ServiceImpl{
		log:              log,
		repo:             repo,
		cache:            cache,
		supportedLocales: []string{"en", "es", "fr"},
		defaultLocale:    "en",
	}
}

func (s *ServiceImpl) CreateTranslation(ctx context.Context, req *i18n.CreateTranslationRequest) (*i18n.CreateTranslationResponse, error) {
	// TODO: Implement CreateTranslation in I18nRepository
	return nil, status.Error(codes.Unimplemented, "CreateTranslation repository integration not yet implemented")
}

func (s *ServiceImpl) GetTranslation(ctx context.Context, req *i18n.GetTranslationRequest) (*i18n.GetTranslationResponse, error) {
	// TODO: Implement GetTranslation in I18nRepository
	return nil, status.Error(codes.Unimplemented, "GetTranslation repository integration not yet implemented")
}

func (s *ServiceImpl) ListTranslations(ctx context.Context, req *i18n.ListTranslationsRequest) (*i18n.ListTranslationsResponse, error) {
	// TODO: Implement ListTranslations in I18nRepository
	return nil, status.Error(codes.Unimplemented, "ListTranslations repository integration not yet implemented")
}
