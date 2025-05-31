package content

import (
	context "context"
	"fmt"
	"time"

	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
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
	translations := map[string]map[string]string{}
	serviceSpecific := map[string]interface{}{"translations": translations}
	meta, err := BuildContentMetadata(
		nil, nil, nil, nil, nil, nil,
		content.Tags,
		serviceSpecific,
	)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.Internal, "failed to build content metadata: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	content.Metadata = meta

	if err := metadata.ValidateMetadata(content.Metadata); err != nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	c, err := s.repo.CreateContent(ctx, content)
	if err != nil {
		s.log.Error("CreateContent failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to create content: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}

	resp := &contentpb.ContentResponse{Content: c}
	success := graceful.WrapSuccess(ctx, codes.OK, "content created", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     c.Id,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     c.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "content.created",
		EventID:      c.Id,
		PatternType:  "content",
		PatternID:    c.Id,
		PatternMeta:  c.Metadata,
	})
	return resp, nil
}

func (s *Service) GetContent(ctx context.Context, req *contentpb.GetContentRequest) (*contentpb.ContentResponse, error) {
	c, err := s.repo.GetContent(ctx, req.Id)
	if err != nil {
		s.log.Error("GetContent failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.NotFound, "content not found: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	return &contentpb.ContentResponse{Content: c}, nil
}

func (s *Service) UpdateContent(ctx context.Context, req *contentpb.UpdateContentRequest) (*contentpb.ContentResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		err := graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "content")
	content, err := s.repo.GetContent(ctx, req.Content.Id)
	if err != nil {
		err := graceful.WrapErr(ctx, codes.NotFound, "content not found", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if !isAdmin && content.AuthorId != authUserID {
		err := graceful.WrapErr(ctx, codes.PermissionDenied, "cannot update content you do not own", nil)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if err := metadata.ValidateMetadata(req.Content.Metadata); err != nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	c, err := s.repo.UpdateContent(ctx, req.Content)
	if err != nil {
		s.log.Error("UpdateContent failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to update content: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &contentpb.ContentResponse{Content: c}
	success := graceful.WrapSuccess(ctx, codes.OK, "content updated", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     c.Id,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     c.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "content.updated",
		EventID:      c.Id,
		PatternType:  "content",
		PatternID:    c.Id,
		PatternMeta:  c.Metadata,
	})
	return resp, nil
}

func (s *Service) DeleteContent(ctx context.Context, req *contentpb.DeleteContentRequest) (*contentpb.DeleteContentResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "content")
	content, err := s.repo.GetContent(ctx, req.Id)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.NotFound, "content not found", nil)
	}
	if !isAdmin && content.AuthorId != authUserID {
		return nil, graceful.WrapErr(ctx, codes.PermissionDenied, "cannot delete content you do not own", nil)
	}
	successVal, err := s.repo.DeleteContent(ctx, req.Id)
	if err != nil {
		s.log.Error("DeleteContent failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to delete content: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}

	resp := &contentpb.DeleteContentResponse{Success: successVal}
	success := graceful.WrapSuccess(ctx, codes.OK, "content deleted", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     req.Id,
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     content.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "content.deleted",
		EventID:      req.Id,
		PatternType:  "content",
		PatternID:    req.Id,
		PatternMeta:  content.Metadata,
	})
	return resp, nil
}

func (s *Service) ListContent(ctx context.Context, req *contentpb.ListContentRequest) (*contentpb.ListContentResponse, error) {
	results, total, err := s.repo.ListContent(ctx, req.AuthorId, req.Type, req.CampaignId, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.Error("ListContent failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list content: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if total > int(^int32(0)) || total < int(^int32(0))*-1 {
		return nil, graceful.WrapErr(ctx, codes.Internal, "total count overflow", nil)
	}
	return &contentpb.ListContentResponse{Contents: results, Total: int32(total)}, nil
}

func (s *Service) AddReaction(ctx context.Context, req *contentpb.AddReactionRequest) (*contentpb.ReactionResponse, error) {
	count, err := s.repo.AddReaction(ctx, req.ContentId, req.UserId, req.Reaction)
	if err != nil {
		s.log.Error("AddReaction failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to add reaction: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if count > int(^int32(0)) || count < int(^int32(0))*-1 {
		return nil, graceful.WrapErr(ctx, codes.Internal, "reaction count overflow", nil)
	}
	resp := &contentpb.ReactionResponse{ContentId: req.ContentId, Reaction: req.Reaction, Count: int32(count)}
	success := graceful.WrapSuccess(ctx, codes.OK, "reaction added", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     fmt.Sprintf("reaction:%s:%s", req.ContentId, req.Reaction),
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "content.reaction_added",
		EventID:      req.ContentId,
		PatternType:  "reaction",
		PatternID:    req.ContentId,
		PatternMeta:  nil, // Optionally fetch content metadata if needed
	})
	return resp, nil
}

func (s *Service) ListReactions(ctx context.Context, req *contentpb.ListReactionsRequest) (*contentpb.ListReactionsResponse, error) {
	m, err := s.repo.ListReactions(ctx, req.ContentId)
	if err != nil {
		s.log.Error("ListReactions failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list reactions: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	// Preallocate slice for performance
	reactions := make([]*contentpb.ReactionResponse, 0, len(m))
	for reaction, count := range m {
		if count > int(^int32(0)) || count < int(^int32(0))*-1 {
			return nil, graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("reaction count overflow for type %s", reaction), nil)
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
	if err := metadata.ValidateMetadata(req.Metadata); err != nil {
		return nil, graceful.WrapErr(ctx, codes.InvalidArgument, "invalid metadata: %v", err)
	}
	comment, err := s.repo.AddComment(ctx, req.ContentId, req.AuthorId, req.Body, req.Metadata)
	if err != nil {
		s.log.Error("AddComment failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to add comment: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &contentpb.CommentResponse{Comment: comment}
	success := graceful.WrapSuccess(ctx, codes.OK, "comment added", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     fmt.Sprintf("comment:%s", comment.Id),
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     comment.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "content.comment_added",
		EventID:      comment.Id,
		PatternType:  "comment",
		PatternID:    comment.Id,
		PatternMeta:  comment.Metadata,
	})
	return resp, nil
}

func (s *Service) ListComments(ctx context.Context, req *contentpb.ListCommentsRequest) (*contentpb.ListCommentsResponse, error) {
	comments, total, err := s.repo.ListComments(ctx, req.ContentId, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.Error("ListComments failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to list comments: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if total > int(^int32(0)) || total < int(^int32(0))*-1 {
		return nil, graceful.WrapErr(ctx, codes.Internal, "total count overflow", nil)
	}
	return &contentpb.ListCommentsResponse{Comments: comments, Total: int32(total)}, nil
}

func (s *Service) DeleteComment(ctx context.Context, req *contentpb.DeleteCommentRequest) (*contentpb.DeleteCommentResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		return nil, graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "content")
	comment, err := s.repo.GetComment(ctx, req.CommentId)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.NotFound, "comment not found", nil)
	}
	if !isAdmin && comment.AuthorId != authUserID {
		return nil, graceful.WrapErr(ctx, codes.PermissionDenied, "cannot delete comment you do not own", nil)
	}
	successVal, err := s.repo.DeleteComment(ctx, req.CommentId)
	if err != nil {
		s.log.Error("DeleteComment failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to delete comment: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &contentpb.DeleteCommentResponse{Success: successVal}
	success := graceful.WrapSuccess(ctx, codes.OK, "comment deleted", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     fmt.Sprintf("comment:%s", req.CommentId),
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		Metadata:     comment.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "content.comment_deleted",
		EventID:      req.CommentId,
		PatternType:  "comment",
		PatternID:    req.CommentId,
		PatternMeta:  comment.Metadata,
	})
	return resp, nil
}

func (s *Service) SearchContent(ctx context.Context, req *contentpb.SearchContentRequest) (*contentpb.ListContentResponse, error) {
	results, total, err := s.repo.SearchContentFlexible(ctx, req)
	if err != nil {
		s.log.Error("SearchContent failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to search content: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	if total > int(^int32(0)) || total < int(^int32(0))*-1 {
		return nil, graceful.WrapErr(ctx, codes.Internal, "total count overflow", nil)
	}
	return &contentpb.ListContentResponse{Contents: results, Total: int32(total)}, nil
}

func (s *Service) LogContentEvent(ctx context.Context, req *contentpb.LogContentEventRequest) (*contentpb.LogContentEventResponse, error) {
	err := s.repo.LogContentEvent(ctx, req.Event)
	if err != nil {
		s.log.Error("LogContentEvent failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to log content event: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	return &contentpb.LogContentEventResponse{Success: true}, nil
}

func (s *Service) ModerateContent(ctx context.Context, req *contentpb.ModerateContentRequest) (*contentpb.ModerateContentResponse, error) {
	successVal, statusStr, err := s.repo.ModerateContent(ctx, req.ContentId, req.Action, req.ModeratorId, req.Reason)
	if err != nil {
		s.log.Error("ModerateContent failed", zap.Error(err))
		err := graceful.WrapErr(ctx, codes.Internal, "failed to moderate content: %v", err)
		err.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		return nil, graceful.ToStatusError(err)
	}
	resp := &contentpb.ModerateContentResponse{Success: successVal, Status: statusStr}
	success := graceful.WrapSuccess(ctx, codes.OK, "content moderated", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     fmt.Sprintf("moderation:%s", req.ContentId),
		CacheValue:   resp,
		CacheTTL:     10 * time.Minute,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "content.moderated",
		EventID:      req.ContentId,
		PatternType:  "moderation",
		PatternID:    req.ContentId,
		PatternMeta:  nil, // Optionally fetch content metadata if needed
	})
	return resp, nil
}
