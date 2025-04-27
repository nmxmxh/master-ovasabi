package referral

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v0"
	"github.com/nmxmxh/master-ovasabi/internal/shared/dbiface"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ServiceImpl implements the ReferralService interface.
type ServiceImpl struct {
	referralpb.UnimplementedReferralServiceServer
	log *zap.Logger
	db  dbiface.DB
}

// NewReferralService creates a new instance of ReferralService.
func NewReferralService(log *zap.Logger, db dbiface.DB) referralpb.ReferralServiceServer {
	return &ServiceImpl{
		log: log,
		db:  db,
	}
}

// CreateReferral creates a new referral code following the Master-Client-Service-Event pattern.
func (s *ServiceImpl) CreateReferral(ctx context.Context, req *referralpb.CreateReferralRequest) (*referralpb.CreateReferralResponse, error) {
	s.log.Info("Creating referral code",
		zap.Int32("referrer_master_id", req.ReferrerMasterId),
		zap.Int32("campaign_id", req.CampaignId))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.log.Warn("failed to rollback tx", zap.Error(err))
		}
	}()

	// 1. Create service_referral record
	referralCode := uuid.New().String()[:8] // Use first 8 chars for shorter code
	var referralID int32
	err = tx.QueryRowContext(ctx,
		`INSERT INTO service_referral 
		 (referrer_master_id, campaign_id, device_hash, referral_code, successful, created_at) 
		 VALUES ($1, $2, $3, $4, false, NOW()) 
		 RETURNING id`,
		req.ReferrerMasterId, req.CampaignId, req.DeviceHash, referralCode).Scan(&referralID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create referral record: %v", err)
	}

	// 2. Log event
	payload, err := json.Marshal(map[string]interface{}{
		"referrer_master_id": req.ReferrerMasterId,
		"campaign_id":        req.CampaignId,
		"device_hash":        req.DeviceHash,
		"referral_code":      referralCode,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal event payload: %v", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO service_event 
		 (master_id, event_type, payload) 
		 VALUES ($1, 'referral_created', $2)`,
		req.ReferrerMasterId, payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to log event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return &referralpb.CreateReferralResponse{
		Referral: &referralpb.Referral{
			Id:               referralID,
			ReferrerMasterId: req.ReferrerMasterId,
			CampaignId:       req.CampaignId,
			ReferralCode:     referralCode,
			Successful:       false,
			CreatedAt:        timestamppb.Now(),
		},
		Success: true,
	}, nil
}

// GetReferralStats retrieves referral statistics.
func (s *ServiceImpl) GetReferralStats(ctx context.Context, req *referralpb.GetReferralStatsRequest) (*referralpb.GetReferralStatsResponse, error) {
	var stats struct {
		TotalReferrals  int32
		ActiveReferrals int32
		TotalRewards    int32
	}

	err := s.db.QueryRowContext(ctx, `
		WITH referral_stats AS (
			SELECT 
				COUNT(*) as total_referrals,
				COUNT(*) FILTER (WHERE successful = true) as active_referrals
			FROM service_referral
			WHERE referrer_master_id = (
				SELECT id FROM master WHERE uuid = $1
			)
		)
		SELECT 
			total_referrals,
			active_referrals,
			COALESCE(
				(SELECT COUNT(*) 
				 FROM service_event 
				 WHERE event_type = 'referral_reward_earned'
				 AND master_id = (SELECT id FROM master WHERE uuid = $1)
				), 0
			) as total_rewards
		FROM referral_stats
	`, req.UserId).Scan(&stats.TotalReferrals, &stats.ActiveReferrals, &stats.TotalRewards)

	if err != nil && err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	// If no records found, return zeros
	if err == sql.ErrNoRows {
		return &referralpb.GetReferralStatsResponse{
			TotalReferrals:  0,
			ActiveReferrals: 0,
			TotalRewards:    0,
		}, nil
	}

	return &referralpb.GetReferralStatsResponse{
		TotalReferrals:  stats.TotalReferrals,
		ActiveReferrals: stats.ActiveReferrals,
		TotalRewards:    stats.TotalRewards,
	}, nil
}

// GetReferral retrieves a specific referral by code.
func (s *ServiceImpl) GetReferral(ctx context.Context, req *referralpb.GetReferralRequest) (*referralpb.GetReferralResponse, error) {
	var ref referralpb.Referral
	var createdAt time.Time

	err := s.db.QueryRowContext(ctx, `
		SELECT id, referrer_master_id, campaign_id, referral_code, successful, created_at
		FROM service_referral
		WHERE referral_code = $1
	`, req.ReferralCode).Scan(
		&ref.Id,
		&ref.ReferrerMasterId,
		&ref.CampaignId,
		&ref.ReferralCode,
		&ref.Successful,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "referral not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	ref.CreatedAt = timestamppb.New(createdAt)

	return &referralpb.GetReferralResponse{
		Referral: &ref,
	}, nil
}
