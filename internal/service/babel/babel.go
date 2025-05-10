package babel

import (
	"context"
	"time"

	babelpb "github.com/nmxmxh/master-ovasabi/api/protos/babel/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository/babel"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PricingRule struct {
	ID            int64
	CountryCode   string
	Region        string
	City          string
	CurrencyCode  string
	AffluenceTier string
	DemandLevel   string
	Multiplier    float64
	BasePrice     float64
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	KGEntityID    string
	Notes         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Service struct {
	babelpb.UnimplementedBabelServiceServer
	repo  *babel.Repository
	cache *redis.Cache
	log   *zap.Logger
}

func NewService(repo *babel.Repository, cache *redis.Cache, log *zap.Logger) *Service {
	return &Service{repo: repo, cache: cache, log: log}
}

// Compile-time check.
var _ babelpb.BabelServiceServer = (*Service)(nil)

// GetLocationContext implements the gRPC method for BabelService.
func (s *Service) GetLocationContext(ctx context.Context, req *babelpb.GetLocationContextRequest) (*babelpb.GetLocationContextResponse, error) {
	rule, err := s.GetLocationContextInternal(ctx, req.CountryCode, req.Region, req.City)
	if err != nil {
		return nil, err
	}
	return &babelpb.GetLocationContextResponse{Rule: toProtoPricingRule(rule)}, nil
}

// Translate implements the gRPC method for BabelService.
func (s *Service) Translate(ctx context.Context, req *babelpb.TranslateRequest) (*babelpb.TranslateResponse, error) {
	val, err := s.TranslateInternal(ctx, req.Key, req.Locale)
	if err != nil {
		return nil, err
	}
	return &babelpb.TranslateResponse{Value: val}, nil
}

// Internal logic for location context (used by gRPC and internal calls).
func (s *Service) GetLocationContextInternal(ctx context.Context, country, region, city string) (*babel.PricingRule, error) {
	cacheKey := "babel:pricing:" + country + ":" + region + ":" + city
	var rule babel.PricingRule
	if err := s.cache.Get(ctx, cacheKey, "", &rule); err == nil {
		return &rule, nil
	}
	rulePtr, err := s.repo.FindBestRule(ctx, country, region, city, time.Now())
	if err != nil {
		s.log.Warn("No pricing rule found", zap.String("country", country), zap.String("region", region), zap.String("city", city), zap.Error(err))
		return nil, err
	}
	if err := s.cache.Set(ctx, cacheKey, "", rulePtr, 10*time.Minute); err != nil {
		s.log.Error("failed to cache pricing rule", zap.String("cacheKey", cacheKey), zap.Error(err))
	}
	return rulePtr, nil
}

// Internal logic for translation (used by gRPC and internal calls).
func (s *Service) TranslateInternal(_ context.Context, _, _ string) (string, error) {
	// TODO: Integrate with translation backend
	return "", nil
}

// Helper to convert internal PricingRule to proto.
func toProtoPricingRule(rule *babel.PricingRule) *babelpb.PricingRule {
	if rule == nil {
		return nil
	}
	var effectiveTo *timestamppb.Timestamp
	if rule.EffectiveTo != nil {
		effectiveTo = timestamppb.New(*rule.EffectiveTo)
	}
	return &babelpb.PricingRule{
		Id:            rule.ID,
		CountryCode:   rule.CountryCode,
		Region:        rule.Region,
		City:          rule.City,
		CurrencyCode:  rule.CurrencyCode,
		AffluenceTier: rule.AffluenceTier,
		DemandLevel:   rule.DemandLevel,
		Multiplier:    rule.Multiplier,
		BasePrice:     rule.BasePrice,
		EffectiveFrom: timestamppb.New(rule.EffectiveFrom),
		EffectiveTo:   effectiveTo,
		KgEntityId:    rule.KGEntityID,
		Notes:         rule.Notes,
		CreatedAt:     timestamppb.New(rule.CreatedAt),
		UpdatedAt:     timestamppb.New(rule.UpdatedAt),
	}
}

// HealthCheck returns nil if the service is healthy.
func (s *Service) HealthCheck(ctx context.Context) error {
	// Check DB connectivity
	if err := s.repo.DB.PingContext(ctx); err != nil {
		return err
	}
	// Optionally, check Redis by setting and getting a test key
	testKey := "babel:healthcheck"
	testVal := "ok"
	if err := s.cache.Set(ctx, testKey, "", testVal, 1*time.Minute); err != nil {
		return err
	}
	var val string
	if err := s.cache.Get(ctx, testKey, "", &val); err != nil || val != testVal {
		return err
	}
	return nil
}

// Future: Add KG enrichment, rule management, analytics, etc.
