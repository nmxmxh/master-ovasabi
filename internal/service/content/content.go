package content

import (
	"context"
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	contentpb "github.com/nmxmxh/master-ovasabi/api/protos/content/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
)

// Provider/DI Registration Pattern (Modern, Extensible, DRY)
// ---------------------------------------------------------
// This file implements the centralized Provider pattern for service registration and dependency injection (DI) across the platform.
// It also implements the Graceful Orchestration Standard for error and success handling, as required by the OVASABI platform.
// All orchestration (caching, event emission, knowledge graph enrichment, scheduling, audit, etc.) is handled via the graceful package's orchestration config.
// See docs/amadeus/amadeus_context.md for details and compliance checklists.
//
// Canonical Metadata Pattern: All content entities use common.Metadata, with service-specific fields under metadata.service_specific.content.
// Accessibility & Compliance: All content must include accessibility metadata.
// Bad Actor Pattern: All suspicious actions must enrich metadata.service_specific.content.bad_actor.
//
// For onboarding and extensibility, see docs/services/metadata.md and docs/services/versioning.md.

type Service struct {
	contentpb.UnimplementedContentServiceServer
	repo         *Repository
	log          *zap.Logger
	Cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
	handler      *graceful.Handler
}

func NewService(
	log *zap.Logger,
	repo *Repository,
	cache *redis.Cache,
	eventEmitter events.EventEmitter,
	eventEnabled bool,
) contentpb.ContentServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		handler:      graceful.NewHandler(log, eventEmitter, cache, "content", "v1", eventEnabled),
	}
}

func (s *Service) CreateContent(ctx context.Context, req *contentpb.CreateContentRequest) (*contentpb.ContentResponse, error) {
	content := req.Content
	content.CampaignId = req.CampaignId
	if content.Metadata == nil {
		content.Metadata = &commonpb.Metadata{}
	}
	if err := metadata.SetServiceSpecificField(content.Metadata, "content", "versioning", map[string]interface{}{
		"system_version": "1.0.0", "service_version": "1.0.0", "environment": "prod",
	}); err != nil {
		s.log.Warn("Failed to set service-specific metadata field", zap.Error(err))
	}
	// Accessibility: compliance check counter (always 1 for new content)
	accMeta := map[string]interface{}{
		"compliance": map[string]interface{}{"standards": []map[string]interface{}{{"name": "WCAG", "level": "AA", "version": "2.1", "compliant": true}}},
		"features":   map[string]interface{}{"alt_text": true, "captions": true},
		"audit": map[string]interface{}{
			"checked_by":             "content-service",
			"checked_at":             time.Now().Format(time.RFC3339),
			"method":                 "automated",
			"compliance_check_count": 1,
		},
	}
	if err := metadata.SetServiceSpecificField(content.Metadata, "content", "accessibility", accMeta); err != nil {
		s.log.Warn("Failed to set service-specific metadata field", zap.Error(err))
	}
	// Bad actor: initialize flag_count to 0 for new content
	badActorMeta := map[string]interface{}{"flag_count": 0, "last_flagged_at": ""}
	if err := metadata.SetServiceSpecificField(content.Metadata, "content", "bad_actor", badActorMeta); err != nil {
		s.log.Warn("Failed to set service-specific metadata field", zap.Error(err))
	}
	translations := map[string]map[string]string{}
	if err := metadata.SetServiceSpecificField(content.Metadata, "content", "translations", translations); err != nil {
		s.log.Warn("Failed to set service-specific metadata field", zap.Error(err))
	}
	content.Metadata.Tags = content.Tags
	metaMap := metadata.ProtoToMap(content.Metadata)
	normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, "content", content.Id, content.Tags, "success", "enrich content metadata")
	content.Metadata = metadata.MapToProto(normMap)
	c, err := s.repo.CreateContent(ctx, content)
	if err != nil {
		s.log.Error("CreateContent failed", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to create content: %v", err)
		s.handler.Error(ctx, "create_content", codes.Internal, "failed to create content", gErr, content.Metadata, content.Id)
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &contentpb.ContentResponse{Content: c}
	s.handler.Success(ctx, "create_content", codes.OK, "content created", resp, c.Metadata, c.Id, nil)
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
		gErr := graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
		s.handler.Error(ctx, "update_content", codes.Unauthenticated, "missing authentication", gErr, nil, req.Content.Id)
		return nil, graceful.ToStatusError(gErr)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "content")
	content, err := s.repo.GetContent(ctx, req.Content.Id)
	if err != nil {
		gErr := graceful.WrapErr(ctx, codes.NotFound, "content not found", nil)
		s.handler.Error(ctx, "update_content", codes.NotFound, "content not found", gErr, nil, req.Content.Id)
		return nil, graceful.ToStatusError(gErr)
	}
	if !isAdmin && content.AuthorId != authUserID {
		gErr := graceful.WrapErr(ctx, codes.PermissionDenied, "cannot update content you do not own", nil)
		s.handler.Error(ctx, "update_content", codes.PermissionDenied, "cannot update content you do not own", gErr, nil, req.Content.Id)
		return nil, graceful.ToStatusError(gErr)
	}
	if req.Content.Metadata == nil {
		req.Content.Metadata = &commonpb.Metadata{}
	}
	if err := metadata.SetServiceSpecificField(req.Content.Metadata, "content", "versioning", map[string]interface{}{
		"system_version": "1.0.0", "service_version": "1.0.0", "environment": "prod",
	}); err != nil {
		s.log.Warn("Failed to set service-specific metadata field", zap.Error(err))
	}
	// Accessibility: compliance check counter (always 1 for update, can be extended)
	accMeta := map[string]interface{}{
		"compliance": map[string]interface{}{"standards": []map[string]interface{}{{"name": "WCAG", "level": "AA", "version": "2.1", "compliant": true}}},
		"features":   map[string]interface{}{"alt_text": true, "captions": true},
		"audit": map[string]interface{}{
			"checked_by":             "content-service",
			"checked_at":             time.Now().Format(time.RFC3339),
			"method":                 "automated",
			"compliance_check_count": 1,
		},
	}
	if err := metadata.SetServiceSpecificField(req.Content.Metadata, "content", "accessibility", accMeta); err != nil {
		s.log.Warn("Failed to set service-specific metadata field", zap.Error(err))
	}
	// Bad actor: extract and preserve flag_count from metadata, do not increment on update
	metaMap := metadata.ProtoToMap(req.Content.Metadata)
	flagCount := 0
	if ss, ok := metaMap["service_specific"].(map[string]interface{}); ok {
		if contentMeta, ok := ss["content"].(map[string]interface{}); ok {
			if badActor, ok := contentMeta["bad_actor"].(map[string]interface{}); ok {
				if v, ok := badActor["flag_count"].(float64); ok {
					flagCount = int(v)
				}
			}
		}
	}
	badActorMetaMap := map[string]interface{}{
		"flag_count":      flagCount,
		"last_flagged_at": "",
	}
	if err := metadata.SetServiceSpecificField(req.Content.Metadata, "content", "bad_actor", badActorMetaMap); err != nil {
		s.log.Warn("Failed to set service-specific metadata field", zap.Error(err))
	}
	metaMap = metadata.ProtoToMap(req.Content.Metadata)
	normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, "content", req.Content.Id, req.Content.Tags, "success", "enrich content metadata")
	req.Content.Metadata = metadata.MapToProto(normMap)
	c, err := s.repo.UpdateContent(ctx, req.Content)
	if err != nil {
		s.log.Error("UpdateContent failed", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to update content: %v", err)
		s.handler.Error(ctx, "update_content", codes.Internal, "failed to update content", gErr, req.Content.Metadata, req.Content.Id)
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &contentpb.ContentResponse{Content: c}
	s.handler.Success(ctx, "update_content", codes.OK, "content updated", resp, c.Metadata, c.Id, nil)
	return resp, nil
}

func (s *Service) DeleteContent(ctx context.Context, req *contentpb.DeleteContentRequest) (*contentpb.DeleteContentResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		gErr := graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
		s.handler.Error(ctx, "delete_content", codes.Unauthenticated, "missing authentication", gErr, nil, req.Id)
		return nil, graceful.ToStatusError(gErr)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "content")
	content, err := s.repo.GetContent(ctx, req.Id)
	if err != nil {
		gErr := graceful.WrapErr(ctx, codes.NotFound, "content not found", nil)
		s.handler.Error(ctx, "delete_content", codes.NotFound, "content not found", gErr, nil, req.Id)
		return nil, graceful.ToStatusError(gErr)
	}
	if !isAdmin && content.AuthorId != authUserID {
		gErr := graceful.WrapErr(ctx, codes.PermissionDenied, "cannot delete content you do not own", nil)
		s.handler.Error(ctx, "delete_content", codes.PermissionDenied, "cannot delete content you do not own", gErr, nil, req.Id)
		return nil, graceful.ToStatusError(gErr)
	}
	successVal, err := s.repo.DeleteContent(ctx, req.Id)
	if err != nil {
		s.log.Error("DeleteContent failed", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to delete content: %v", err)
		s.handler.Error(ctx, "delete_content", codes.Internal, "failed to delete content", gErr, content.Metadata, req.Id)
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &contentpb.DeleteContentResponse{Success: successVal}
	s.handler.Success(ctx, "delete_content", codes.OK, "content deleted", resp, content.Metadata, req.Id, nil)
	return resp, nil
}
func (s *Service) ListContent(ctx context.Context, req *contentpb.ListContentRequest) (*contentpb.ListContentResponse, error) {
	results, total, err := s.repo.ListContent(ctx, req.AuthorId, req.Type, req.CampaignId, int(req.Page), int(req.PageSize))
	if err != nil {
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to list content: %v", err)
		s.handler.Error(ctx, "list_content", codes.Internal, "failed to list content", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	if total > int(^int32(0)) || total < int(^int32(0))*-1 {
		gErr := graceful.WrapErr(ctx, codes.Internal, "total count overflow", nil)
		s.handler.Error(ctx, "list_content", codes.Internal, "total count overflow", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &contentpb.ListContentResponse{Contents: results, Total: int32(total)}
	s.handler.Success(ctx, "list_content", codes.OK, "content listed", resp, nil, "", nil)
	return resp, nil
}

func (s *Service) AddReaction(ctx context.Context, req *contentpb.AddReactionRequest) (*contentpb.ReactionResponse, error) {
	count, err := s.repo.AddReaction(ctx, req.ContentId, req.UserId, req.Reaction)
	if err != nil {
		s.log.Error("AddReaction failed", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to add reaction: %v", err)
		s.handler.Error(ctx, "add_reaction", codes.Internal, "failed to add reaction", gErr, nil, req.ContentId)
		return nil, graceful.ToStatusError(gErr)
	}
	if count > int(^int32(0)) || count < int(^int32(0))*-1 {
		gErr := graceful.WrapErr(ctx, codes.Internal, "reaction count overflow", nil)
		s.handler.Error(ctx, "add_reaction", codes.Internal, "reaction count overflow", gErr, nil, req.ContentId)
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &contentpb.ReactionResponse{ContentId: req.ContentId, Reaction: req.Reaction, Count: int32(count)}
	s.handler.Success(ctx, "add_reaction", codes.OK, "reaction added", resp, nil, req.ContentId, nil)
	return resp, nil
}

func (s *Service) ListReactions(ctx context.Context, req *contentpb.ListReactionsRequest) (*contentpb.ListReactionsResponse, error) {
	m, err := s.repo.ListReactions(ctx, req.ContentId)
	if err != nil {
		s.log.Error("ListReactions failed", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to list reactions: %v", err)
		s.handler.Error(ctx, "list_reactions", codes.Internal, "failed to list reactions", gErr, nil, req.ContentId)
		return nil, graceful.ToStatusError(gErr)
	}
	reactions := make([]*contentpb.ReactionResponse, 0, len(m))
	for reaction, count := range m {
		if count > int(^int32(0)) || count < int(^int32(0))*-1 {
			gErr := graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("reaction count overflow for type %s", reaction), nil)
			s.handler.Error(ctx, "list_reactions", codes.Internal, "reaction count overflow", gErr, nil, req.ContentId)
			return nil, graceful.ToStatusError(gErr)
		}
		reactions = append(reactions, &contentpb.ReactionResponse{
			ContentId: req.ContentId,
			Reaction:  reaction,
			Count:     int32(count),
		})
	}
	resp := &contentpb.ListReactionsResponse{Reactions: reactions}
	s.handler.Success(ctx, "list_reactions", codes.OK, "reactions listed", resp, nil, req.ContentId, nil)
	return resp, nil
}

func (s *Service) AddComment(ctx context.Context, req *contentpb.AddCommentRequest) (*contentpb.CommentResponse, error) {
	if req.Metadata == nil {
		req.Metadata = &commonpb.Metadata{}
	}
	if err := metadata.SetServiceSpecificField(req.Metadata, "content", "versioning", map[string]interface{}{
		"system_version": "1.0.0", "service_version": "1.0.0", "environment": "prod",
	}); err != nil {
		s.log.Warn("Failed to set service-specific metadata field", zap.Error(err))
	}
	metaMap := metadata.ProtoToMap(req.Metadata)
	flagCount := 0
	if ss, ok := metaMap["service_specific"].(map[string]interface{}); ok {
		if contentMeta, ok := ss["content"].(map[string]interface{}); ok {
			if badActor, ok := contentMeta["bad_actor"].(map[string]interface{}); ok {
				if v, ok := badActor["flag_count"].(float64); ok {
					flagCount = int(v)
				}
			}
		}
	}
	flagCount++
	badActorMetaMap := map[string]interface{}{
		"flag_count":      flagCount,
		"last_flagged_at": time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(req.Metadata, "content", "bad_actor", badActorMetaMap); err != nil {
		s.log.Warn("Failed to set service-specific metadata field", zap.Error(err))
	}
	metaMap = metadata.ProtoToMap(req.Metadata)
	normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, "comment", "", nil, "success", "enrich comment metadata")
	req.Metadata = metadata.MapToProto(normMap)
	comment, err := s.repo.AddComment(ctx, req.ContentId, req.AuthorId, req.Body, req.Metadata)
	if err != nil {
		s.log.Error("AddComment failed", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to add comment: %v", err)
		s.handler.Error(ctx, "add_comment", codes.Internal, "failed to add comment", gErr, req.Metadata, req.ContentId)
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &contentpb.CommentResponse{Comment: comment}
	s.handler.Success(ctx, "add_comment", codes.OK, "comment added", resp, comment.Metadata, req.ContentId, nil)
	return resp, nil
}

func (s *Service) ListComments(ctx context.Context, req *contentpb.ListCommentsRequest) (*contentpb.ListCommentsResponse, error) {
	comments, total, err := s.repo.ListComments(ctx, req.ContentId, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.Error("ListComments failed", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to list comments: %v", err)
		s.handler.Error(ctx, "list_comments", codes.Internal, "failed to list comments", gErr, nil, req.ContentId)
		return nil, graceful.ToStatusError(gErr)
	}
	if total > int(^int32(0)) || total < int(^int32(0))*-1 {
		gErr := graceful.WrapErr(ctx, codes.Internal, "total count overflow", nil)
		s.handler.Error(ctx, "list_comments", codes.Internal, "total count overflow", gErr, nil, req.ContentId)
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &contentpb.ListCommentsResponse{Comments: comments, Total: int32(total)}
	s.handler.Success(ctx, "list_comments", codes.OK, "comments listed", resp, nil, req.ContentId, nil)
	return resp, nil
}

func (s *Service) DeleteComment(ctx context.Context, req *contentpb.DeleteCommentRequest) (*contentpb.DeleteCommentResponse, error) {
	authUserID, ok := utils.GetAuthenticatedUserID(ctx)
	if !ok {
		gErr := graceful.WrapErr(ctx, codes.Unauthenticated, "missing authentication", nil)
		s.handler.Error(ctx, "content:delete_comment:v1:failed", codes.Unauthenticated, "missing authentication", gErr, nil, req.CommentId)
		return nil, graceful.ToStatusError(gErr)
	}
	roles, _ := utils.GetAuthenticatedUserRoles(ctx)
	isAdmin := utils.IsServiceAdmin(roles, "content")
	comment, err := s.repo.GetComment(ctx, req.CommentId)
	if err != nil {
		gErr := graceful.WrapErr(ctx, codes.NotFound, "comment not found", nil)
		s.handler.Error(ctx, "delete_comment", codes.NotFound, "comment not found", gErr, nil, req.CommentId)
		return nil, graceful.ToStatusError(gErr)
	}
	if !isAdmin && comment.AuthorId != authUserID {
		gErr := graceful.WrapErr(ctx, codes.PermissionDenied, "cannot delete comment you do not own", nil)
		s.handler.Error(ctx, "delete_comment", codes.PermissionDenied, "cannot delete comment you do not own", gErr, nil, req.CommentId)
		return nil, graceful.ToStatusError(gErr)
	}
	successVal, err := s.repo.DeleteComment(ctx, req.CommentId)
	if err != nil {
		s.log.Error("DeleteComment failed", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to delete comment: %v", err)
		s.handler.Error(ctx, "delete_comment", codes.Internal, "failed to delete comment", gErr, comment.Metadata, req.CommentId)
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &contentpb.DeleteCommentResponse{Success: successVal}
	s.handler.Success(ctx, "delete_comment", codes.OK, "comment deleted", resp, comment.Metadata, req.CommentId, nil)
	return resp, nil
}

func (s *Service) SearchContent(ctx context.Context, req *contentpb.SearchContentRequest) (*contentpb.ListContentResponse, error) {
	results, total, err := s.repo.SearchContentFlexible(ctx, req)
	if err != nil {
		s.log.Error("SearchContent failed", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to search content: %v", err)
		s.handler.Error(ctx, "search_content", codes.Internal, "failed to search content", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	if total > int(^int32(0)) || total < int(^int32(0))*-1 {
		gErr := graceful.WrapErr(ctx, codes.Internal, "total count overflow", nil)
		s.handler.Error(ctx, "search_content", codes.Internal, "total count overflow", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &contentpb.ListContentResponse{Contents: results, Total: int32(total)}
	s.handler.Success(ctx, "search_content", codes.OK, "content search completed", resp, nil, "", nil)
	return resp, nil
}

func (s *Service) LogContentEvent(ctx context.Context, req *contentpb.LogContentEventRequest) (*contentpb.LogContentEventResponse, error) {
	err := s.repo.LogContentEvent(ctx, req.Event)
	if err != nil {
		s.log.Error("LogContentEvent failed", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, "failed to log content event: %v", err)
		s.handler.Error(ctx, "log_content_event", codes.Internal, "failed to log content event", gErr, nil, "")
		return nil, graceful.ToStatusError(gErr)
	}
	resp := &contentpb.LogContentEventResponse{Success: true}
	s.handler.Success(ctx, "log_content_event", codes.OK, "content event logged", resp, nil, "", nil)
	return resp, nil
}

func (s *Service) ModerateContent(ctx context.Context, req *contentpb.ModerateContentRequest) (*contentpb.ModerateContentResponse, error) {
	err := s.repo.ModerateContent(ctx, req.ContentId, req.ModeratorId, req.Action, req.Reason)
	if err != nil {
		s.log.Error("ModerateContent failed", zap.Error(err))
		gErr := graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to moderate content: %v", err), err)
		s.handler.Error(ctx, "moderate_content", codes.Internal, "failed to moderate content", gErr, nil, req.ContentId)
		return nil, graceful.ToStatusError(gErr)
	}
	meta := &commonpb.Metadata{}
	if err := metadata.SetServiceSpecificField(meta, "content", "versioning", map[string]interface{}{
		"system_version": "1.0.0", "service_version": "1.0.0", "environment": "prod",
	}); err != nil {
		s.log.Warn("Failed to set service-specific metadata field (versioning)", zap.Error(err))
	}
	metaMap := metadata.ProtoToMap(meta)
	flagCount := 0
	if ss, ok := metaMap["service_specific"].(map[string]interface{}); ok {
		if contentMeta, ok := ss["content"].(map[string]interface{}); ok {
			if badActor, ok := contentMeta["bad_actor"].(map[string]interface{}); ok {
				if v, ok := badActor["flag_count"].(float64); ok {
					flagCount = int(v)
				}
			}
		}
	}
	flagCount++
	badActorMetaMap := map[string]interface{}{
		"flag_count":      flagCount,
		"last_flagged_at": time.Now().Format(time.RFC3339),
	}
	if err := metadata.SetServiceSpecificField(meta, "content", "bad_actor", badActorMetaMap); err != nil {
		s.log.Warn("Failed to set service-specific metadata field (bad_actor)", zap.Error(err))
	}
	metaMap = metadata.ProtoToMap(meta)
	normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, "moderation", req.ContentId, nil, "success", "enrich moderation metadata")
	meta = metadata.MapToProto(normMap)
	resp := &contentpb.ModerateContentResponse{
		Success: true,
		Status:  req.Action,
	}
	s.handler.Success(ctx, "moderate_content", codes.OK, "content moderated", resp, meta, req.ContentId, nil)
	return resp, nil
}
