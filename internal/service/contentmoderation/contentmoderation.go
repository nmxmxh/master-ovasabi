package contentmoderation

import (
	"context"
	"encoding/json"
	"errors"

	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
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
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.submit_failed", "", nil)
		}
		return nil, errors.New("content_id and user_id are required")
	}
	meta, err := ExtractAndEnrichContentModerationMetadata(req.GetMetadata(), req.UserId, true)
	if err != nil {
		s.log.Error("Failed to enrich moderation metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.submit_failed", req.ContentId, nil)
		}
		return nil, err
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		s.log.Error("Failed to marshal metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.submit_failed", req.ContentId, nil)
		}
		return nil, err
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
		s.log.Error("Failed to submit content for moderation", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.submit_failed", req.ContentId, nil)
		}
		return nil, err
	}
	if s.eventEnabled && s.eventEmitter != nil {
		events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.submitted", req.ContentId, meta)
	}
	return &contentmoderationpb.SubmitContentForModerationResponse{Result: result}, nil
}

func (s *Service) GetModerationResult(ctx context.Context, req *contentmoderationpb.GetModerationResultRequest) (*contentmoderationpb.GetModerationResultResponse, error) {
	if req == nil || req.ContentId == "" {
		return nil, errors.New("content_id is required")
	}
	result, err := s.repo.GetModerationResult(ctx, req.ContentId)
	if err != nil {
		s.log.Error("Failed to get moderation result", zap.Error(err))
		return nil, err
	}
	return &contentmoderationpb.GetModerationResultResponse{Result: result}, nil
}

func (s *Service) ListFlaggedContent(ctx context.Context, req *contentmoderationpb.ListFlaggedContentRequest) (*contentmoderationpb.ListFlaggedContentResponse, error) {
	if req == nil {
		return nil, errors.New("request is required")
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
		s.log.Error("Failed to list flagged content", zap.Error(err))
		return nil, err
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
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.approve_failed", "", nil)
		}
		return nil, errors.New("content_id is required")
	}
	meta, err := ExtractAndEnrichContentModerationMetadata(req.GetMetadata(), "moderator", false)
	if err != nil {
		s.log.Error("Failed to enrich moderation metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.approve_failed", req.ContentId, nil)
		}
		return nil, err
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		s.log.Error("Failed to marshal metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.approve_failed", req.ContentId, nil)
		}
		return nil, err
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
		s.log.Error("Failed to approve content", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.approve_failed", req.ContentId, nil)
		}
		return nil, err
	}
	if s.eventEnabled && s.eventEmitter != nil {
		events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.approved", req.ContentId, meta)
	}
	return &contentmoderationpb.ApproveContentResponse{Result: result}, nil
}

func (s *Service) RejectContent(ctx context.Context, req *contentmoderationpb.RejectContentRequest) (*contentmoderationpb.RejectContentResponse, error) {
	if req == nil || req.ContentId == "" {
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.reject_failed", "", nil)
		}
		return nil, errors.New("content_id is required")
	}
	meta, err := ExtractAndEnrichContentModerationMetadata(req.GetMetadata(), "moderator", false)
	if err != nil {
		s.log.Error("Failed to enrich moderation metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.reject_failed", req.ContentId, nil)
		}
		return nil, err
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		s.log.Error("Failed to marshal metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.reject_failed", req.ContentId, nil)
		}
		return nil, err
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
		s.log.Error("Failed to reject content", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.reject_failed", req.ContentId, nil)
		}
		return nil, err
	}
	if s.eventEnabled && s.eventEmitter != nil {
		events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "contentmoderation.rejected", req.ContentId, meta)
	}
	return &contentmoderationpb.RejectContentResponse{Result: result}, nil
}
