package contentmoderation

import (
	"context"
	"encoding/json"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

type Repository interface {
	SubmitContentForModeration(ctx context.Context, contentID, masterID, masterUUID, userID, contentType, content string, metadata []byte, campaignID int64) (*contentmoderationpb.ModerationResult, error)
	GetModerationResult(ctx context.Context, contentID string) (*contentmoderationpb.ModerationResult, error)
	ListFlaggedContent(ctx context.Context, page, pageSize int, status string, campaignID int64) ([]*contentmoderationpb.ModerationResult, int, error)
	ApproveContent(ctx context.Context, contentID, masterID, masterUUID string, metadata []byte, campaignID int64) (*contentmoderationpb.ModerationResult, error)
	RejectContent(ctx context.Context, contentID, masterID, masterUUID, reason string, metadata []byte, campaignID int64) (*contentmoderationpb.ModerationResult, error)
}

type Service struct {
	contentmoderationpb.UnimplementedContentModerationServiceServer
	log          *zap.Logger
	repo         Repository
	cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
}

func NewContentModerationService(log *zap.Logger, repo Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool) contentmoderationpb.ContentModerationServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

var _ contentmoderationpb.ContentModerationServiceServer = (*Service)(nil)

// Provider/DI Registration Pattern (Modern, Extensible, DRY)
// ---------------------------------------------------------
// This file implements the centralized Provider pattern for service registration and dependency injection (DI) across the platform.
// It also implements the Graceful Orchestration Standard for error and success handling, as required by the OVASABI platform.
// All orchestration (caching, event emission, knowledge graph enrichment, scheduling, audit, etc.) is handled via the graceful package's orchestration config.
// See docs/amadeus/amadeus_context.md for details and compliance checklists.
//
// Canonical Metadata Pattern: All moderation entities use common.Metadata, with service-specific fields under metadata.service_specific.contentmoderation.
// Bad Actor Pattern: All moderation events increment bad_actor.flag_count and update last_flagged_at.
// Accessibility & Compliance: Add accessibility field if moderation includes accessibility checks.
//
// For onboarding and extensibility, see docs/services/metadata.md and docs/services/versioning.md.
func (s *Service) SubmitContentForModeration(ctx context.Context, req *contentmoderationpb.SubmitContentForModerationRequest) (*contentmoderationpb.SubmitContentForModerationResponse, error) {
	if req == nil || req.ContentId == "" || req.UserId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "content_id and user_id are required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	meta := req.GetMetadata()
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "audit", map[string]interface{}{"created_by": req.UserId, "history": []string{"created"}}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set audit in content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "versioning", map[string]interface{}{"system_version": "1.0.0", "service_version": "1.0.0", "moderation_version": "1.0.0", "environment": "prod"}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set versioning in content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "compliance", map[string]interface{}{"policy": "platform_default"}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set compliance in content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	// Bad Actor: increment flag_count and update last_flagged_at
	metaMap := metadata.ProtoToMap(meta)
	flagCount := 0
	if ss, ok := metaMap["service_specific"].(map[string]interface{}); ok {
		if cmMeta, ok := ss["contentmoderation"].(map[string]interface{}); ok {
			if badActor, ok := cmMeta["bad_actor"].(map[string]interface{}); ok {
				if v, ok := badActor["flag_count"].(float64); ok {
					flagCount = int(v)
				}
			}
		}
	}
	flagCount++
	badActorMeta := map[string]interface{}{
		"flag_count":      flagCount,
		"last_flagged_at": time.Now().UTC().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "bad_actor", badActorMeta); err != nil {
		s.log.Warn("Failed to set service-specific metadata field (bad_actor)", zap.Error(err))
	}
	// Accessibility (stub): add if moderation includes accessibility checks
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "accessibility", map[string]interface{}{"checked": false}); err != nil {
		s.log.Warn("Failed to set service-specific metadata field (accessibility)", zap.Error(err))
	}
	// Normalize metadata before persistence/emission
	metaMap = metadata.ProtoToMap(meta)
	normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, "contentmoderation", req.ContentId, nil, "success", "normalize moderation metadata")
	meta = metadata.MapToProto(normMap)
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to marshal content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	var masterID, masterUUID string
	modVars := metadata.ExtractServiceVariables(meta, "contentmoderation")
	if v, ok := modVars["masterID"].(string); ok {
		masterID = v
	}
	if v, ok := modVars["masterUUID"].(string); ok {
		masterUUID = v
	}
	result, err := s.repo.SubmitContentForModeration(ctx, req.ContentId, masterID, masterUUID, req.UserId, req.ContentType, req.Content, metaJSON, req.CampaignId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to submit content for moderation", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &contentmoderationpb.SubmitContentForModerationResponse{Result: result}
	success := graceful.WrapSuccess(ctx, codes.OK, "content submitted for moderation", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "contentmoderation.submitted",
		EventID:      req.ContentId,
		PatternType:  "contentmoderation",
		PatternID:    req.ContentId,
		PatternMeta:  meta,
	})
	return resp, nil
}

func (s *Service) GetModerationResult(ctx context.Context, req *contentmoderationpb.GetModerationResultRequest) (*contentmoderationpb.GetModerationResultResponse, error) {
	if req == nil || req.ContentId == "" {
		return nil, graceful.WrapErr(ctx, codes.InvalidArgument, "content_id is required", nil)
	}
	result, err := s.repo.GetModerationResult(ctx, req.ContentId)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to get moderation result", err)
	}
	return &contentmoderationpb.GetModerationResultResponse{Result: result}, nil
}

func (s *Service) ListFlaggedContent(ctx context.Context, req *contentmoderationpb.ListFlaggedContentRequest) (*contentmoderationpb.ListFlaggedContentResponse, error) {
	if req == nil {
		return nil, graceful.WrapErr(ctx, codes.InvalidArgument, "request is required", nil)
	}
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 20
	}
	status := req.Status
	if status == "" {
		status = "PENDING"
	}
	results, total, err := s.repo.ListFlaggedContent(ctx, page, pageSize, status, req.CampaignId)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to list flagged content", err)
	}
	totalPages := utils.ToInt32((total + pageSize - 1) / pageSize)
	return &contentmoderationpb.ListFlaggedContentResponse{
		Results:    results,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) ApproveContent(ctx context.Context, req *contentmoderationpb.ApproveContentRequest) (*contentmoderationpb.ApproveContentResponse, error) {
	if req == nil || req.ContentId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "content_id is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	meta := req.GetMetadata()
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "audit", map[string]interface{}{"created_by": "moderator", "history": []string{"approved"}}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set audit in content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "versioning", map[string]interface{}{"system_version": "1.0.0", "service_version": "1.0.0", "moderation_version": "1.0.0", "environment": "prod"}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set versioning in content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "compliance", map[string]interface{}{"policy": "platform_default"}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set compliance in content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	// Bad Actor: increment flag_count and update last_flagged_at
	metaMap := metadata.ProtoToMap(meta)
	flagCount := 0
	if ss, ok := metaMap["service_specific"].(map[string]interface{}); ok {
		if cmMeta, ok := ss["contentmoderation"].(map[string]interface{}); ok {
			if badActor, ok := cmMeta["bad_actor"].(map[string]interface{}); ok {
				if v, ok := badActor["flag_count"].(float64); ok {
					flagCount = int(v)
				}
			}
		}
	}
	flagCount++
	badActorMeta := map[string]interface{}{
		"flag_count":      flagCount,
		"last_flagged_at": time.Now().UTC().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "bad_actor", badActorMeta); err != nil {
		s.log.Warn("Failed to set service-specific metadata field (bad_actor)", zap.Error(err))
	}
	// Accessibility (stub): add if moderation includes accessibility checks
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "accessibility", map[string]interface{}{"checked": false}); err != nil {
		s.log.Warn("Failed to set service-specific metadata field (accessibility)", zap.Error(err))
	}
	// Normalize metadata before persistence/emission
	metaMap = metadata.ProtoToMap(meta)
	normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, "contentmoderation", req.ContentId, nil, "success", "normalize moderation metadata")
	meta = metadata.MapToProto(normMap)
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to marshal content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	var masterID, masterUUID string
	modVars := metadata.ExtractServiceVariables(meta, "contentmoderation")
	if v, ok := modVars["masterID"].(string); ok {
		masterID = v
	}
	if v, ok := modVars["masterUUID"].(string); ok {
		masterUUID = v
	}
	result, err := s.repo.ApproveContent(ctx, req.ContentId, masterID, masterUUID, metaJSON, req.CampaignId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to approve content", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &contentmoderationpb.ApproveContentResponse{Result: result}
	success := graceful.WrapSuccess(ctx, codes.OK, "content approved", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "contentmoderation.approved",
		EventID:      req.ContentId,
		PatternType:  "contentmoderation",
		PatternID:    req.ContentId,
		PatternMeta:  meta,
	})
	return resp, nil
}

func (s *Service) RejectContent(ctx context.Context, req *contentmoderationpb.RejectContentRequest) (*contentmoderationpb.RejectContentResponse, error) {
	if req == nil || req.ContentId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "content_id is required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	meta := req.GetMetadata()
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "audit", map[string]interface{}{"created_by": "moderator", "history": []string{"rejected"}, "reason": req.Reason}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set audit in content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "versioning", map[string]interface{}{"system_version": "1.0.0", "service_version": "1.0.0", "moderation_version": "1.0.0", "environment": "prod"}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set versioning in content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "compliance", map[string]interface{}{"policy": "platform_default"}); err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to set compliance in content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	// Bad Actor: increment flag_count and update last_flagged_at
	metaMap := metadata.ProtoToMap(meta)
	flagCount := 0
	if ss, ok := metaMap["service_specific"].(map[string]interface{}); ok {
		if cmMeta, ok := ss["contentmoderation"].(map[string]interface{}); ok {
			if badActor, ok := cmMeta["bad_actor"].(map[string]interface{}); ok {
				if v, ok := badActor["flag_count"].(float64); ok {
					flagCount = int(v)
				}
			}
		}
	}
	flagCount++
	badActorMeta := map[string]interface{}{
		"flag_count":      flagCount,
		"last_flagged_at": time.Now().UTC().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "bad_actor", badActorMeta); err != nil {
		s.log.Warn("Failed to set service-specific metadata field (bad_actor)", zap.Error(err))
	}
	// Accessibility (stub): add if moderation includes accessibility checks
	if err := metadata.SetServiceSpecificField(meta, "contentmoderation", "accessibility", map[string]interface{}{"checked": false}); err != nil {
		s.log.Warn("Failed to set service-specific metadata field (accessibility)", zap.Error(err))
	}
	// Normalize metadata before persistence/emission
	metaMap = metadata.ProtoToMap(meta)
	normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, "contentmoderation", req.ContentId, nil, "success", "normalize moderation metadata")
	meta = metadata.MapToProto(normMap)
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to marshal content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	var masterID, masterUUID string
	modVars := metadata.ExtractServiceVariables(meta, "contentmoderation")
	if v, ok := modVars["masterID"].(string); ok {
		masterID = v
	}
	if v, ok := modVars["masterUUID"].(string); ok {
		masterUUID = v
	}
	result, err := s.repo.RejectContent(ctx, req.ContentId, masterID, masterUUID, req.Reason, metaJSON, req.CampaignId)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to reject content", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &contentmoderationpb.RejectContentResponse{Result: result}
	success := graceful.WrapSuccess(ctx, codes.OK, "content rejected", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "contentmoderation.rejected",
		EventID:      req.ContentId,
		PatternType:  "contentmoderation",
		PatternID:    req.ContentId,
		PatternMeta:  meta,
	})
	return resp, nil
}
