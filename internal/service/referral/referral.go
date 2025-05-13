package referral

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v1"
	referralrepo "github.com/nmxmxh/master-ovasabi/internal/repository/referral"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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
func (s *ServiceImpl) CreateReferral(ctx context.Context, req *referralpb.CreateReferralRequest) (*referralpb.CreateReferralResponse, error) {
	// 1. Validate metadata
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid metadata: "+err.Error())
	}
	// 2. Generate referral code (simple example, should be more robust in prod)
	referralCode := generateReferralCode()
	// 3. Store referral in DB
	record := &referralrepo.Referral{
		ReferrerMasterID: req.ReferrerMasterId,
		ReferredMasterID: sqlNullString(""), // Not provided at creation
		CampaignID:       sqlNullInt64(req.CampaignId),
		DeviceHash:       sqlNullString(req.DeviceHash),
		ReferralCode:     referralCode,
		Successful:       false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Metadata:         req.Metadata,
	}
	if err := s.repo.Create(record); err != nil {
		s.log.Error("failed to create referral", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create referral")
	}
	// 4. Integration points
	idStr := fmt.Sprint(record.ID)
	if s.cache != nil && record.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.cache, "referral", idStr, record.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "referral", idStr, record.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "referral", idStr, record.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "referral", record.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	// 5. Return referral details
	return &referralpb.CreateReferralResponse{
		Referral: toProtoReferral(record),
		Success:  true,
	}, nil
}

// GetReferralStats retrieves referral statistics.
func (s *ServiceImpl) GetReferralStats(_ context.Context, _ *referralpb.GetReferralStatsRequest) (*referralpb.GetReferralStatsResponse, error) {
	// TODO: Implement GetReferralStats
	// Pseudocode:
	// 1. Validate user/referrer permissions
	// 2. Query DB for referral stats
	// 3. Return stats
	return nil, status.Error(codes.Unimplemented, "GetReferralStats not yet implemented")
}

// GetReferral retrieves a specific referral by code.
func (s *ServiceImpl) GetReferral(_ context.Context, req *referralpb.GetReferralRequest) (*referralpb.GetReferralResponse, error) {
	if req.ReferralCode == "" {
		return nil, status.Error(codes.InvalidArgument, "referral_code is required")
	}
	s.log.Debug("GetReferral called", zap.String("referral_code", req.ReferralCode))
	referral, err := s.repo.GetByCode(req.ReferralCode)
	if err != nil {
		if errors.Is(err, referralrepo.ErrReferralNotFound) {
			return nil, status.Error(codes.NotFound, "referral not found")
		}
		s.log.Error("failed to get referral by code", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get referral")
	}
	return &referralpb.GetReferralResponse{
		Referral: toProtoReferral(referral),
	}, nil
}

// TODO (Amadeus Context): Implement RegisterReferral and RewardReferral following the canonical metadata pattern when the proto definitions are available.
// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
// Steps: Validate, store, notify, and log in Nexus as described in the Amadeus context.

// Helper: map repository Referral to proto Referral.
func toProtoReferral(r *referralrepo.Referral) *referralpb.Referral {
	if r == nil {
		return nil
	}
	return &referralpb.Referral{
		Id:               r.ID,
		ReferrerMasterId: r.ReferrerMasterID,
		ReferredMasterId: r.ReferredMasterID.String,
		CampaignId:       r.CampaignID.Int64,
		DeviceHash:       r.DeviceHash.String,
		ReferralCode:     r.ReferralCode,
		Successful:       r.Successful,
		CreatedAt:        timestamppb.New(r.CreatedAt),
		UpdatedAt:        timestamppb.New(r.UpdatedAt),
		Metadata:         r.Metadata, // direct assignment
	}
}

// Helper functions for null types.
func sqlNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func sqlNullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{Int64: i, Valid: i != 0}
}

// Simple referral code generator (for demo; replace with robust version in prod).
func generateReferralCode() string {
	return fmt.Sprintf("REF-%d", time.Now().UnixNano())
}
