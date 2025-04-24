package referral

import (
	"context"
	"time"

	"github.com/nmxmxh/master-ovasabi/api/protos/referral"
	"go.uber.org/zap"
)

// ReferralServiceImpl implements the ReferralService interface
type ReferralServiceImpl struct {
	referral.UnimplementedReferralServiceServer
	log *zap.Logger
}

// NewReferralService creates a new instance of ReferralService
func NewReferralService(log *zap.Logger) *ReferralServiceImpl {
	return &ReferralServiceImpl{
		log: log,
	}
}

// CreateReferral implements the CreateReferral RPC method
func (s *ReferralServiceImpl) CreateReferral(ctx context.Context, req *referral.CreateReferralRequest) (*referral.CreateReferralResponse, error) {
	// TODO: Implement proper referral code generation
	// For now, just return a mock referral code
	return &referral.CreateReferralResponse{
		ReferralCode: "MOCK-REF-CODE",
		Message:      "Referral code created successfully",
	}, nil
}

// ApplyReferral implements the ApplyReferral RPC method
func (s *ReferralServiceImpl) ApplyReferral(ctx context.Context, req *referral.ApplyReferralRequest) (*referral.ApplyReferralResponse, error) {
	// TODO: Implement proper referral application
	// For now, just return success with mock rewards
	return &referral.ApplyReferralResponse{
		Success:      true,
		Message:      "Referral applied successfully",
		RewardPoints: 100,
	}, nil
}

// GetReferralStats implements the GetReferralStats RPC method
func (s *ReferralServiceImpl) GetReferralStats(ctx context.Context, req *referral.GetReferralStatsRequest) (*referral.GetReferralStatsResponse, error) {
	// TODO: Implement proper stats calculation
	// For now, just return mock stats
	return &referral.GetReferralStatsResponse{
		TotalReferrals:  5,
		ActiveReferrals: 3,
		TotalRewards:    500,
		Referrals: []*referral.ReferralDetail{
			{
				ReferralCode:   "MOCK-REF-1",
				ReferredUserId: "mock-user-1",
				CreatedAt:      time.Now().Unix(),
				IsActive:       true,
				RewardPoints:   100,
			},
		},
	}, nil
}
