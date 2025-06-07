package referral

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service struct implements the ReferralService interface.
type Service struct {
	referralpb.UnimplementedReferralServiceServer
	log          *zap.Logger
	cache        *redis.Cache
	repo         *Repository
	eventEmitter events.EventEmitter
	eventEnabled bool
}

// Compile-time check.
var _ referralpb.ReferralServiceServer = (*Service)(nil)

// NewService creates a new instance of ReferralService.
func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool) referralpb.ReferralServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

// CreateReferral creates a new referral code following the Master-Client-Service-Event pattern.
func (s *Service) CreateReferral(ctx context.Context, req *referralpb.CreateReferralRequest) (*referralpb.CreateReferralResponse, error) {
	metadata.MigrateMetadata(req.Metadata)
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "context", codes.InvalidArgument))
	}
	referralCode := generateReferralCode()
	referrerMasterID, err := strconv.ParseInt(req.ReferrerMasterId, 10, 64)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "context", codes.InvalidArgument))
	}
	record := &Referral{
		ReferrerMasterID: referrerMasterID,
		ReferredMasterID: 0,
		CampaignID:       sqlNullInt64(req.CampaignId),
		DeviceHash:       sqlNullString(req.DeviceHash),
		ReferralCode:     referralCode,
		Successful:       false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Metadata:         req.Metadata,
	}
	if err := s.repo.Create(record); err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "context", codes.Internal))
	}
	idStr := fmt.Sprint(record.ID)
	success := graceful.WrapSuccess(ctx, codes.OK, "referral created", record, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.cache, CacheKey: idStr, CacheValue: record, CacheTTL: 10 * time.Minute, Metadata: record.Metadata, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "referral.created", EventID: idStr, PatternType: "referral", PatternID: idStr, PatternMeta: record.Metadata})
	return &referralpb.CreateReferralResponse{
		Referral: toProtoReferral(record),
		Success:  true,
	}, nil
}

// GetReferralStats retrieves referral statistics.
func (s *Service) GetReferralStats(ctx context.Context, req *referralpb.GetReferralStatsRequest) (*referralpb.GetReferralStatsResponse, error) {
	if req == nil || req.MasterId == 0 {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "context", codes.InvalidArgument))
	}
	stats, err := s.repo.GetStats(req.MasterId)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "context", codes.Internal))
	}
	resp := &referralpb.GetReferralStatsResponse{
		TotalReferrals:  utils.ToInt32(int(stats.TotalReferrals)),
		ActiveReferrals: utils.ToInt32(int(stats.SuccessfulReferrals)),
		GeneratedAt:     timestamppb.Now(),
	}
	return resp, nil
}

// GetReferral retrieves a specific referral by code.
func (s *Service) GetReferral(ctx context.Context, req *referralpb.GetReferralRequest) (*referralpb.GetReferralResponse, error) {
	if req.ReferralCode == "" {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "context", codes.InvalidArgument))
	}
	s.log.Debug("GetReferral called", zap.String("referral_code", req.ReferralCode))
	referral, err := s.repo.GetByCode(req.ReferralCode)
	if err != nil {
		if errors.Is(err, ErrReferralNotFound) {
			return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "context", codes.NotFound))
		}
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "context", codes.Internal))
	}
	return &referralpb.GetReferralResponse{
		Referral: toProtoReferral(referral),
	}, nil
}

// RegisterReferral registers a new referral and emits an event.
func (s *Service) RegisterReferral(ctx context.Context, req *referralpb.RegisterReferralRequest) (*referralpb.RegisterReferralResponse, error) {
	metadata.MigrateMetadata(req.Metadata)
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "invalid metadata", codes.InvalidArgument))
	}
	referrerMasterID, err := strconv.ParseInt(req.ReferrerMasterId, 10, 64)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "invalid referrer_master_id", codes.InvalidArgument))
	}
	record := &Referral{
		ReferrerMasterID: referrerMasterID,
		ReferredMasterID: 0,
		CampaignID:       sqlNullInt64(req.CampaignId),
		DeviceHash:       sqlNullString(req.DeviceHash),
		ReferralCode:     generateReferralCode(),
		Successful:       false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Metadata:         req.Metadata,
	}
	if err := s.repo.Create(record); err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to register referral", codes.Internal))
	}
	idStr := fmt.Sprint(record.ID)
	success := graceful.WrapSuccess(ctx, codes.OK, "referral registered", record, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.cache,
		CacheKey:     idStr,
		CacheValue:   record,
		CacheTTL:     10 * time.Minute,
		Metadata:     record.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "referral.registered",
		EventID:      idStr,
		PatternType:  "referral",
		PatternID:    idStr,
		PatternMeta:  record.Metadata,
	})
	return &referralpb.RegisterReferralResponse{
		Referral: toProtoReferral(record),
		Success:  true,
	}, nil
}

// RewardReferral rewards a referral and emits an event.
func (s *Service) RewardReferral(ctx context.Context, req *referralpb.RewardReferralRequest) (*referralpb.RewardReferralResponse, error) {
	metadata.MigrateMetadata(req.Metadata)
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "invalid metadata", codes.InvalidArgument))
	}
	referral, err := s.repo.GetByCode(req.ReferralCode)
	if err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "referral not found", codes.NotFound))
	}
	// Mark as successful and update metadata with reward info
	referral.Successful = true
	referral.UpdatedAt = time.Now()
	referral.Metadata = req.Metadata
	if err := s.repo.Update(referral); err != nil {
		return nil, graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "failed to reward referral", codes.Internal))
	}
	idStr := fmt.Sprint(referral.ID)
	success := graceful.WrapSuccess(ctx, codes.OK, "referral rewarded", referral, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.cache,
		CacheKey:     idStr,
		CacheValue:   referral,
		CacheTTL:     10 * time.Minute,
		Metadata:     referral.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "referral.rewarded",
		EventID:      idStr,
		PatternType:  "referral",
		PatternID:    idStr,
		PatternMeta:  referral.Metadata,
	})
	return &referralpb.RewardReferralResponse{
		Referral: toProtoReferral(referral),
		Success:  true,
	}, nil
}

// Helper: map repository Referral to proto Referral.
func toProtoReferral(r *Referral) *referralpb.Referral {
	if r == nil {
		return nil
	}
	return &referralpb.Referral{
		Id:               r.ID,
		ReferrerMasterId: strconv.FormatInt(r.ReferrerMasterID, 10),
		ReferredMasterId: strconv.FormatInt(r.ReferredMasterID, 10),
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

// UpdateReferredMasterID updates the referred master ID for a referral.
func (s *Service) UpdateReferredMasterID(ctx context.Context, referralCode, referredMasterID string) error {
	if referralCode == "" {
		return graceful.ToStatusError(graceful.MapAndWrapErr(ctx, nil, "context", codes.InvalidArgument))
	}
	var referredMasterIDInt int64
	if referredMasterID != "" {
		var err error
		referredMasterIDInt, err = strconv.ParseInt(referredMasterID, 10, 64)
		if err != nil {
			return graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "context", codes.InvalidArgument))
		}
	}
	if err := s.repo.UpdateReferredMasterID(referralCode, referredMasterIDInt); err != nil {
		return graceful.ToStatusError(graceful.MapAndWrapErr(ctx, err, "context", codes.Internal))
	}
	return nil
}
