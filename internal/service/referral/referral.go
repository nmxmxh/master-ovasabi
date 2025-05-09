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
	repo  *referralrepo.ReferralRepository
}

// NewReferralService creates a new instance of ReferralService.
func NewReferralService(log *zap.Logger, repo *referralrepo.ReferralRepository, cache *redis.Cache) referralpb.ReferralServiceServer {
	return &ServiceImpl{
		log:   log,
		repo:  repo,
		cache: cache,
	}
}

// CreateReferral creates a new referral code following the Master-Client-Service-Event pattern.
func (s *ServiceImpl) CreateReferral(ctx context.Context, req *referralpb.CreateReferralRequest) (*referralpb.CreateReferralResponse, error) {
	// TODO: Implement CreateReferral in ReferralRepository
	return nil, status.Error(codes.Unimplemented, "CreateReferral repository integration not yet implemented")
}

// GetReferralStats retrieves referral statistics.
func (s *ServiceImpl) GetReferralStats(ctx context.Context, req *referralpb.GetReferralStatsRequest) (*referralpb.GetReferralStatsResponse, error) {
	// TODO: Implement GetReferralStats in ReferralRepository
	return nil, status.Error(codes.Unimplemented, "GetReferralStats repository integration not yet implemented")
}

// GetReferral retrieves a specific referral by code.
func (s *ServiceImpl) GetReferral(ctx context.Context, req *referralpb.GetReferralRequest) (*referralpb.GetReferralResponse, error) {
	// TODO: Implement GetReferral in ReferralRepository
	return nil, status.Error(codes.Unimplemented, "GetReferral repository integration not yet implemented")
}
