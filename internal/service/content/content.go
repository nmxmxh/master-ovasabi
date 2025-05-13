package contentservice

import (
	context "context"
	"time"

	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	contentmoderationpb "github.com/nmxmxh/master-ovasabi/api/protos/contentmoderation/v1"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	userpb "github.com/nmxmxh/master-ovasabi/api/protos/user/v1"
	contentrepo "github.com/nmxmxh/master-ovasabi/internal/repository/content"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	contentpb.UnimplementedContentServiceServer
	repo            *contentrepo.Repository
	log             *zap.Logger
	userSvc         userpb.UserServiceServer
	notificationSvc notificationpb.NotificationServiceServer
	searchSvc       searchpb.SearchServiceServer
	moderationSvc   contentmoderationpb.ContentModerationServiceServer
	Cache           *redis.Cache
}

func NewContentService(log *zap.Logger, repo *contentrepo.Repository, userSvc userpb.UserServiceServer, notificationSvc notificationpb.NotificationServiceServer, searchSvc searchpb.SearchServiceServer, moderationSvc contentmoderationpb.ContentModerationServiceServer) contentpb.ContentServiceServer {
	return &Service{
		log:             log,
		repo:            repo,
		userSvc:         userSvc,
		notificationSvc: notificationSvc,
		searchSvc:       searchSvc,
		moderationSvc:   moderationSvc,
	}
}

func (s *Service) CreateContent(ctx context.Context, req *contentpb.CreateContentRequest) (*contentpb.ContentResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Content.Metadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	c, err := s.repo.CreateContent(ctx, req.Content)
	if err != nil {
		s.log.Error("CreateContent failed", zap.Error(err))
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
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	c, err := s.repo.UpdateContent(ctx, req.Content)
	if err != nil {
		s.log.Error("UpdateContent failed", zap.Error(err))
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
	return &contentpb.ContentResponse{Content: c}, nil
}

func (s *Service) DeleteContent(ctx context.Context, req *contentpb.DeleteContentRequest) (*contentpb.DeleteContentResponse, error) {
	success, err := s.repo.DeleteContent(ctx, req.Id)
	if err != nil {
		s.log.Error("DeleteContent failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to delete content: %v", err)
	}
	return &contentpb.DeleteContentResponse{Success: success}, nil
}

func (s *Service) ListContent(ctx context.Context, req *contentpb.ListContentRequest) (*contentpb.ListContentResponse, error) {
	results, total, err := s.repo.ListContent(ctx, req.AuthorId, req.Type, "", int(req.Page), int(req.PageSize))
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
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	comment, err := s.repo.AddComment(ctx, req.ContentId, req.AuthorId, req.Body, req.Metadata)
	if err != nil {
		s.log.Error("AddComment failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to add comment: %v", err)
	}
	if s.Cache != nil && comment.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.Cache, "comment", comment.Id, comment.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "comment", comment.Id, comment.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "comment", comment.Id, comment.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "comment", comment.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
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
		return nil, status.Errorf(codes.Internal, "failed to delete comment: %v", err)
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
	// TODO: Implement moderation logic (call repo.ModerateContent, enrich event, notify, etc.)
	success, statusStr, err := s.repo.ModerateContent(ctx, req.ContentId, req.Action, req.ModeratorId, req.Reason)
	if err != nil {
		s.log.Error("ModerateContent failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to moderate content: %v", err)
	}
	return &contentpb.ModerateContentResponse{Success: success, Status: statusStr}, nil
}
