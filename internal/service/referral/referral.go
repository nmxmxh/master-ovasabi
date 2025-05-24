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
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventEmitter defines the interface for emitting events in the referral service.
type EventEmitter interface {
	EmitEvent(ctx context.Context, eventType, entityID string, metadata *commonpb.Metadata) error
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
	// Always build and enrich metadata before validation and storage
	fraudSignals := map[string]interface{}{"device_hash": req.DeviceHash}
	audit := map[string]interface{}{"created_by": req.ReferrerMasterId, "created_at": time.Now().Format(time.RFC3339)}
	// You can add more enrichment here (e.g., rewards, campaign, device info)
	meta, err := metadata.BuildReferralMetadata(fraudSignals, nil, audit, nil, nil)
	if err == nil {
		req.Metadata = meta
	}
	// 1. Validate metadata using the shared helper
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		s.log.Error("invalid referral metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for referral.create_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "referral.create_failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit referral.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Error(codes.InvalidArgument, "invalid metadata: "+err.Error())
	}
	// 2. Generate referral code (simple example, should be more robust in prod)
	referralCode := generateReferralCode()
	// 3. Store referral in DB
	referrerMasterID, err := strconv.ParseInt(req.ReferrerMasterId, 10, 64)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid ReferrerMasterId")
	}
	record := &Referral{
		ReferrerMasterID: referrerMasterID,
		ReferredMasterID: 0, // Not provided at creation
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
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for referral.create_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "referral.create_failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit referral.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Error(codes.Internal, "failed to create referral")
	}
	// 4. Integration points (all use the improved metadata structure)
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
	record.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "referral.created", idStr, record.Metadata)
	// 5. Return referral details
	return &referralpb.CreateReferralResponse{
		Referral: toProtoReferral(record),
		Success:  true,
	}, nil
}

// GetReferralStats retrieves referral statistics.
func (s *Service) GetReferralStats(_ context.Context, _ *referralpb.GetReferralStatsRequest) (*referralpb.GetReferralStatsResponse, error) {
	// TODO: Implement GetReferralStats
	// Pseudocode:
	// 1. Validate user/referrer permissions
	// 2. Query DB for referral stats
	// 3. Return stats
	return nil, status.Error(codes.Unimplemented, "GetReferralStats not yet implemented")
}

// GetReferral retrieves a specific referral by code.
func (s *Service) GetReferral(_ context.Context, req *referralpb.GetReferralRequest) (*referralpb.GetReferralResponse, error) {
	if req.ReferralCode == "" {
		return nil, status.Error(codes.InvalidArgument, "referral_code is required")
	}
	s.log.Debug("GetReferral called", zap.String("referral_code", req.ReferralCode))
	referral, err := s.repo.GetByCode(req.ReferralCode)
	if err != nil {
		if errors.Is(err, ErrReferralNotFound) {
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
func (s *Service) UpdateReferredMasterID(_ context.Context, referralCode, referredMasterID string) error {
	if referralCode == "" {
		return status.Error(codes.InvalidArgument, "referral_code is required")
	}
	var referredMasterIDInt int64
	if referredMasterID != "" {
		var err error
		referredMasterIDInt, err = strconv.ParseInt(referredMasterID, 10, 64)
		if err != nil {
			return status.Error(codes.InvalidArgument, "invalid referredMasterID")
		}
	}
	if err := s.repo.UpdateReferredMasterID(referralCode, referredMasterIDInt); err != nil {
		s.log.Error("failed to update referred master id", zap.Error(err))
		return status.Error(codes.Internal, "failed to update referred master id")
	}
	return nil
}
