package referral

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	referralpb "github.com/nmxmxh/master-ovasabi/api/protos/referral/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventEmitter defines the interface for emitting events in the referral service.
type EventEmitter interface {
	EmitEventWithLogging(ctx context.Context, emitter interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool)
}

// Service struct implements the ReferralService interface.
type Service struct {
	referralpb.UnimplementedReferralServiceServer
	log          *zap.Logger
	cache        *redis.Cache
	repo         *Repository
	eventEmitter EventEmitter
	eventEnabled bool
}

// Compile-time check.
var _ referralpb.ReferralServiceServer = (*Service)(nil)

// NewService creates a new instance of ReferralService.
func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) referralpb.ReferralServiceServer {
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
	fraudSignals := map[string]interface{}{"device_hash": req.DeviceHash}
	audit := map[string]interface{}{"created_by": req.ReferrerMasterId, "created_at": time.Now().Format(time.RFC3339)}
	meta, err := metadata.BuildReferralMetadata(fraudSignals, nil, audit, nil, nil)
	if err == nil {
		req.Metadata = meta
	}
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "invalid referral metadata", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	referralCode := generateReferralCode()
	referrerMasterID, err := strconv.ParseInt(req.ReferrerMasterId, 10, 64)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "invalid ReferrerMasterId", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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
		err = graceful.WrapErr(ctx, codes.Internal, "failed to create referral", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "MasterId is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	stats, err := s.repo.GetStats(req.MasterId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to get referral stats", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "referral_code is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	s.log.Debug("GetReferral called", zap.String("referral_code", req.ReferralCode))
	referral, err := s.repo.GetByCode(req.ReferralCode)
	if err != nil {
		if errors.Is(err, ErrReferralNotFound) {
			err = graceful.WrapErr(ctx, codes.NotFound, "referral not found", err)
			var ce *graceful.ContextError
			if errors.As(err, &ce) {
				ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
			}
			return nil, graceful.ToStatusError(err)
		}
		err = graceful.WrapErr(ctx, codes.Internal, "failed to get referral", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	return &referralpb.GetReferralResponse{
		Referral: toProtoReferral(referral),
	}, nil
}

// TODO (Amadeus Context): Implement RegisterReferral and RewardReferral following the canonical metadata pattern when the proto definitions are available.
// Reference: docs/amadeus/amadeus_context.md, section 'Canonical Metadata Integration Pattern (System-Wide)'.
// Steps: Validate, store, notify, and log in Nexus as described in the Amadeus context.

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
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "referral_code is required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return graceful.ToStatusError(err)
	}
	var referredMasterIDInt int64
	if referredMasterID != "" {
		var err error
		referredMasterIDInt, err = strconv.ParseInt(referredMasterID, 10, 64)
		if err != nil {
			err = graceful.WrapErr(ctx, codes.InvalidArgument, "invalid referredMasterID", err)
			var ce *graceful.ContextError
			if errors.As(err, &ce) {
				ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
			}
			return graceful.ToStatusError(err)
		}
	}
	if err := s.repo.UpdateReferredMasterID(referralCode, referredMasterIDInt); err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to update referred master id", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return graceful.ToStatusError(err)
	}
	return nil
}
