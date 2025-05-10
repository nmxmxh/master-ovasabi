package referral

import (
	"context"

	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v1"
	referralrepo "github.com/nmxmxh/master-ovasabi/internal/repository/referral"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ServiceImpl implements the ReferralService interface.
type ServiceImpl struct {
	referralpb.UnimplementedReferralServiceServer
	log   *zap.Logger
	cache *redis.Cache
	repo  *referralrepo.Repository
}

// Compile-time check.
var _ referralpb.ReferralServiceServer = (*ServiceImpl)(nil)

// NewReferralService creates a new instance of ReferralService.
func NewReferralService(log *zap.Logger, repo *referralrepo.Repository, cache *redis.Cache) referralpb.ReferralServiceServer {
	return &ServiceImpl{
		log:   log,
		repo:  repo,
		cache: cache,
	}
}

// CreateReferral creates a new referral code following the Master-Client-Service-Event pattern.
func (s *ServiceImpl) CreateReferral(_ context.Context, _ *referralpb.CreateReferralRequest) (*referralpb.CreateReferralResponse, error) {
	// TODO: Implement CreateReferral
	return nil, status.Error(codes.Unimplemented, "CreateReferral not yet implemented")
}

// GetReferralStats retrieves referral statistics.
func (s *ServiceImpl) GetReferralStats(_ context.Context, _ *referralpb.GetReferralStatsRequest) (*referralpb.GetReferralStatsResponse, error) {
	// TODO: Implement GetReferralStats
	return nil, status.Error(codes.Unimplemented, "GetReferralStats not yet implemented")
}

// GetReferral retrieves a specific referral by code.
func (s *ServiceImpl) GetReferral(_ context.Context, _ *referralpb.GetReferralRequest) (*referralpb.GetReferralResponse, error) {
	// TODO: Implement GetReferral
	return nil, status.Error(codes.Unimplemented, "GetReferral not yet implemented")
}
