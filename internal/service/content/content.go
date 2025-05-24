package content

import (
	context "context"
	"time"

	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	events "github.com/nmxmxh/master-ovasabi/pkg/events"
)

type Service struct {
	contentpb.UnimplementedContentServiceServer
	repo         *Repository
	log          *zap.Logger
	Cache        *redis.Cache
	eventEmitter EventEmitter
	eventEnabled bool
}

func NewService(
	log *zap.Logger,
	repo *Repository,
	cache *redis.Cache,
	eventEmitter EventEmitter,
	eventEnabled bool,
) contentpb.ContentServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

func (s *Service) CreateContent(ctx context.Context, req *contentpb.CreateContentRequest) (*contentpb.ContentResponse, error) {
	content := req.Content
	content.CampaignId = req.CampaignId
	// Extract fields to translate
	title := content.Title
	body := content.Body

	// Supported locales (could be dynamic/configurable)
	supportedLocales := []string{"en", "fr", "es", "ar"}
	translations := map[string]map[string]string{} // locale -> field -> value

	// Instead of direct localizationClient call, emit translation_requested events
	if s.eventEnabled && s.eventEmitter != nil {
		for _, locale := range supportedLocales {
			meta := &commonpb.Metadata{}
			metaStruct, err := structpb.NewStruct(map[string]interface{}{
				"title":      title,
				"body":       body,
				"locale":     locale,
				"content_id": "", // Not yet created
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for content.translation_requested event", zap.Error(err))
				continue
			}
			meta.ServiceSpecific = metaStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "content.translation_requested", "", meta)
			if errEmit != nil {
				s.log.Warn("Failed to emit content.translation_requested event", zap.Error(errEmit))
			}
		}
	}

	// Build content metadata (add translations under service_specific.content.translations)
	serviceSpecific := map[string]interface{}{
		"translations": translations,
	}
	meta, err := BuildContentMetadata(
		nil,             // accessibility
		nil,             // localization
		nil,             // moderation
		nil,             // aiEnrichment
		nil,             // audit
		nil,             // compliance
		content.Tags,    // tags
		serviceSpecific, // serviceSpecific
	)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error": err.Error(),
				"title": content.Title,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for content.create_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "content.create_failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit content.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to build content metadata: %v", err)
	}
	content.Metadata = meta

	if err := metadatautil.ValidateMetadata(content.Metadata); err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error": err.Error(),
				"title": content.Title,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for content.create_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "content.create_failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit content.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	c, err := s.repo.CreateContent(ctx, content)
	if err != nil {
		s.log.Error("CreateContent failed", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error": err.Error(),
				"title": content.Title,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for content.create_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "content.create_failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit content.create_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to create content: %v", err)
	}
	if s.Cache != nil && c.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "content", c.Id, c.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "content", c.Id, c.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "content", c.Id, c.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "content", c.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	c.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "content.created", c.Id, c.Metadata)
	return &contentpb.ContentResponse{Content: c}, nil
}

func (s *Service) GetContent(ctx context.Context, req *contentpb.GetContentRequest) (*contentpb.ContentResponse, error) {
	c, err := s.repo.GetContent(ctx, req.Id)
	if err != nil {
		s.log.Error("GetContent failed", zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "content not found: %v", err)
	}
	return &contentpb.ContentResponse{Content: c}, nil
}

func (s *Service) UpdateContent(ctx context.Context, req *contentpb.UpdateContentRequest) (*contentpb.ContentResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Content.Metadata); err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":      err.Error(),
				"content_id": req.Content.Id,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for content.update_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "content.update_failed", req.Content.Id, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit content.update_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	c, err := s.repo.UpdateContent(ctx, req.Content)
	if err != nil {
		s.log.Error("UpdateContent failed", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":      err.Error(),
				"content_id": req.Content.Id,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for content.update_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "content.update_failed", req.Content.Id, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit content.update_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to update content: %v", err)
	}
	if s.Cache != nil && c.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "content", c.Id, c.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "content", c.Id, c.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "content", c.Id, c.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "content", c.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	c.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "content.updated", c.Id, c.Metadata)
	return &contentpb.ContentResponse{Content: c}, nil
}

func (s *Service) DeleteContent(ctx context.Context, req *contentpb.DeleteContentRequest) (*contentpb.DeleteContentResponse, error) {
	success, err := s.repo.DeleteContent(ctx, req.Id)
	if err != nil {
		s.log.Error("DeleteContent failed", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":      err.Error(),
				"content_id": req.Id,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for content.delete_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "content.delete_failed", req.Id, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit content.delete_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to delete content: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "content.deleted", req.Id, nil)
		if errEmit != nil {
			s.log.Warn("Failed to emit content.deleted event", zap.Error(errEmit))
		}
	}
	return &contentpb.DeleteContentResponse{Success: success}, nil
}

func (s *Service) ListContent(ctx context.Context, req *contentpb.ListContentRequest) (*contentpb.ListContentResponse, error) {
	results, total, err := s.repo.ListContent(ctx, req.AuthorId, req.Type, req.CampaignId, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.Error("ListContent failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list content: %v", err)
	}
	if total > int(^int32(0)) || total < int(^int32(0))*-1 {
		return nil, status.Errorf(codes.Internal, "total count overflow")
	}
	return &contentpb.ListContentResponse{Contents: results, Total: int32(total)}, nil
}

func (s *Service) AddReaction(ctx context.Context, req *contentpb.AddReactionRequest) (*contentpb.ReactionResponse, error) {
	count, err := s.repo.AddReaction(ctx, req.ContentId, req.UserId, req.Reaction)
	if err != nil {
		s.log.Error("AddReaction failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to add reaction: %v", err)
	}
	if count > int(^int32(0)) || count < int(^int32(0))*-1 {
		return nil, status.Errorf(codes.Internal, "reaction count overflow")
	}
	return &contentpb.ReactionResponse{ContentId: req.ContentId, Reaction: req.Reaction, Count: int32(count)}, nil
}

func (s *Service) ListReactions(ctx context.Context, req *contentpb.ListReactionsRequest) (*contentpb.ListReactionsResponse, error) {
	m, err := s.repo.ListReactions(ctx, req.ContentId)
	if err != nil {
		s.log.Error("ListReactions failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list reactions: %v", err)
	}
	// Preallocate slice for performance
	reactions := make([]*contentpb.ReactionResponse, 0, len(m))
	for reaction, count := range m {
		if count > int(^int32(0)) || count < int(^int32(0))*-1 {
			return nil, status.Errorf(codes.Internal, "reaction count overflow for type %s", reaction)
		}
		reactions = append(reactions, &contentpb.ReactionResponse{
			ContentId: req.ContentId,
			Reaction:  reaction,
			Count:     int32(count),
		})
	}
	return &contentpb.ListReactionsResponse{Reactions: reactions}, nil
}

func (s *Service) AddComment(ctx context.Context, req *contentpb.AddCommentRequest) (*contentpb.CommentResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Metadata); err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":      err.Error(),
				"content_id": req.ContentId,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for content.comment_add_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "content.comment_add_failed", req.ContentId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit content.comment_add_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	comment, err := s.repo.AddComment(ctx, req.ContentId, req.AuthorId, req.Body, req.Metadata)
	if err != nil {
		s.log.Error("AddComment failed", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":      err.Error(),
				"content_id": req.ContentId,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for content.comment_add_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "content.comment_add_failed", req.ContentId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit content.comment_add_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to add comment: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "content.comment_added", req.ContentId, req.Metadata)
		if errEmit != nil {
			s.log.Warn("Failed to emit content.comment_added event", zap.Error(errEmit))
		}
	}
	return &contentpb.CommentResponse{Comment: comment}, nil
}

func (s *Service) ListComments(ctx context.Context, req *contentpb.ListCommentsRequest) (*contentpb.ListCommentsResponse, error) {
	comments, total, err := s.repo.ListComments(ctx, req.ContentId, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.Error("ListComments failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list comments: %v", err)
	}
	if total > int(^int32(0)) || total < int(^int32(0))*-1 {
		return nil, status.Errorf(codes.Internal, "total count overflow")
	}
	return &contentpb.ListCommentsResponse{Comments: comments, Total: int32(total)}, nil
}

func (s *Service) DeleteComment(ctx context.Context, req *contentpb.DeleteCommentRequest) (*contentpb.DeleteCommentResponse, error) {
	success, err := s.repo.DeleteComment(ctx, req.CommentId)
	if err != nil {
		s.log.Error("DeleteComment failed", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":      err.Error(),
				"comment_id": req.CommentId,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for content.comment_delete_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "content.comment_delete_failed", req.CommentId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit content.comment_delete_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to delete comment: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errEmit := s.eventEmitter.EmitEvent(ctx, "content.comment_deleted", req.CommentId, nil)
		if errEmit != nil {
			s.log.Warn("Failed to emit content.comment_deleted event", zap.Error(errEmit))
		}
	}
	return &contentpb.DeleteCommentResponse{Success: success}, nil
}

func (s *Service) SearchContent(ctx context.Context, req *contentpb.SearchContentRequest) (*contentpb.ListContentResponse, error) {
	results, total, err := s.repo.SearchContentFlexible(ctx, req)
	if err != nil {
		s.log.Error("SearchContent failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to search content: %v", err)
	}
	if total > int(^int32(0)) || total < int(^int32(0))*-1 {
		return nil, status.Errorf(codes.Internal, "total count overflow")
	}
	return &contentpb.ListContentResponse{Contents: results, Total: int32(total)}, nil
}

func (s *Service) LogContentEvent(ctx context.Context, req *contentpb.LogContentEventRequest) (*contentpb.LogContentEventResponse, error) {
	err := s.repo.LogContentEvent(ctx, req.Event)
	if err != nil {
		s.log.Error("LogContentEvent failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to log content event: %v", err)
	}
	return &contentpb.LogContentEventResponse{Success: true}, nil
}

func (s *Service) ModerateContent(ctx context.Context, req *contentpb.ModerateContentRequest) (*contentpb.ModerateContentResponse, error) {
	success, statusStr, err := s.repo.ModerateContent(ctx, req.ContentId, req.Action, req.ModeratorId, req.Reason)
	if err != nil {
		s.log.Error("ModerateContent failed", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{
				"error":      err.Error(),
				"content_id": req.ContentId,
			})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for content.moderate_failed event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "content.moderate_failed", req.ContentId, errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit content.moderate_failed event", zap.Error(errEmit))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to moderate content: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		errStruct, err := structpb.NewStruct(map[string]interface{}{
			"action":       req.Action,
			"moderator_id": req.ModeratorId,
			"reason":       req.Reason,
			"status":       statusStr,
		})
		if err != nil {
			s.log.Error("Failed to create structpb.Struct for content.moderated event", zap.Error(err))
			return nil, status.Error(codes.Internal, "internal error")
		}
		errMeta := &commonpb.Metadata{}
		errMeta.ServiceSpecific = errStruct
		errEmit := s.eventEmitter.EmitEvent(ctx, "content.moderated", req.ContentId, errMeta)
		if errEmit != nil {
			s.log.Warn("Failed to emit content.moderated event", zap.Error(errEmit))
		}
	}
	return &contentpb.ModerateContentResponse{Success: success, Status: statusStr}, nil
}
