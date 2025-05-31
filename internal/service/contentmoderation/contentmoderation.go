package contentmoderation

import (
	"context"
	"encoding/json"

	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
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
	eventEmitter EventEmitter
	eventEnabled bool
}

func NewContentModerationService(log *zap.Logger, repo Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) contentmoderationpb.ContentModerationServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

var _ contentmoderationpb.ContentModerationServiceServer = (*Service)(nil)

func (s *Service) SubmitContentForModeration(ctx context.Context, req *contentmoderationpb.SubmitContentForModerationRequest) (*contentmoderationpb.SubmitContentForModerationResponse, error) {
	if req == nil || req.ContentId == "" || req.UserId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "content_id and user_id are required", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	meta, err := ExtractAndEnrichContentModerationMetadata(req.GetMetadata(), req.UserId, true)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to extract and enrich content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to marshal content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	var masterID, masterUUID string
	if meta != nil && meta.ServiceSpecific != nil {
		ssMap := meta.ServiceSpecific.AsMap()
		if cm, ok := ssMap["contentmoderation"].(map[string]interface{}); ok {
			if v, ok := cm["masterID"].(string); ok {
				masterID = v
			}
			if v, ok := cm["masterUUID"].(string); ok {
				masterUUID = v
			}
		}
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
	meta, err := ExtractAndEnrichContentModerationMetadata(req.GetMetadata(), "moderator", false)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to extract and enrich content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to marshal content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	var masterID, masterUUID string
	if meta != nil && meta.ServiceSpecific != nil {
		ssMap := meta.ServiceSpecific.AsMap()
		if cm, ok := ssMap["contentmoderation"].(map[string]interface{}); ok {
			if v, ok := cm["masterID"].(string); ok {
				masterID = v
			}
			if v, ok := cm["masterUUID"].(string); ok {
				masterUUID = v
			}
		}
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
	meta, err := ExtractAndEnrichContentModerationMetadata(req.GetMetadata(), "moderator", false)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to extract and enrich content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to marshal content moderation metadata", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	var masterID, masterUUID string
	if meta != nil && meta.ServiceSpecific != nil {
		ssMap := meta.ServiceSpecific.AsMap()
		if cm, ok := ssMap["contentmoderation"].(map[string]interface{}); ok {
			if v, ok := cm["masterID"].(string); ok {
				masterID = v
			}
			if v, ok := cm["masterUUID"].(string); ok {
				masterUUID = v
			}
		}
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
