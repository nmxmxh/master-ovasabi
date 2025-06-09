// Messaging Service Implementation
// -------------------------------
//
// This file implements the MessagingService gRPC interface, following the robust service pattern.
// It uses dependency injection for logger, repository, and cache, and is ready for extensibility.
//
// See docs/amadeus/amadeus_context.md for service standards.

package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	messagingpb "github.com/nmxmxh/master-ovasabi/api/protos/messaging/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the MessagingService gRPC interface.
type Service struct {
	messagingpb.UnimplementedMessagingServiceServer
	log          *zap.Logger
	repo         *Repository
	cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
}

// NewService creates a new MessagingService instance with event bus support.
func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool) messagingpb.MessagingServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

// Helper: map repository.Message to messagingpb.Message.
func mapRepoMessageToProto(msg *Message) *messagingpb.Message {
	if msg == nil {
		return nil
	}
	return &messagingpb.Message{
		Id:             msg.ID,
		ThreadId:       msg.ThreadID,
		ConversationId: msg.ConversationID,
		ChatGroupId:    msg.ChatGroupID,
		SenderId:       msg.SenderID,
		RecipientIds:   msg.RecipientIDs,
		Content:        msg.Content,
		Type:           messagingpb.MessageType(messagingpb.MessageType_value[msg.Type]),
		// Attachments, Reactions: omitted for brevity
		Status:    messagingpb.MessageStatus(messagingpb.MessageStatus_value[msg.Status]),
		CreatedAt: timestamppb.New(msg.CreatedAt),
		UpdatedAt: timestamppb.New(msg.UpdatedAt),
		Edited:    msg.Edited,
		Deleted:   msg.Deleted,
		Metadata:  msg.Metadata,
	}
}

// Helper: map repository.Thread to messagingpb.Thread.
func mapRepoThreadToProto(thread *Thread) *messagingpb.Thread {
	if thread == nil {
		return nil
	}
	return &messagingpb.Thread{
		Id:             thread.ID,
		ParticipantIds: thread.ParticipantIDs,
		Subject:        thread.Subject,
		MessageIds:     thread.MessageIDs,
		Metadata:       thread.Metadata,
		CreatedAt:      timestamppb.New(thread.CreatedAt),
		UpdatedAt:      timestamppb.New(thread.UpdatedAt),
	}
}

// Helper: map repository.Conversation to messagingpb.Conversation.
func mapRepoConversationToProto(conv *Conversation) *messagingpb.Conversation {
	if conv == nil {
		return nil
	}
	return &messagingpb.Conversation{
		Id:             conv.ID,
		ParticipantIds: conv.ParticipantIDs,
		ChatGroupId:    conv.ChatGroupID,
		ThreadIds:      conv.ThreadIDs,
		Metadata:       conv.Metadata,
		CreatedAt:      timestamppb.New(conv.CreatedAt),
		UpdatedAt:      timestamppb.New(conv.UpdatedAt),
	}
}

// Helper: map repository.ChatGroup to messagingpb.ChatGroup.
func mapRepoChatGroupToProto(group *ChatGroup) *messagingpb.ChatGroup {
	if group == nil {
		return nil
	}
	return &messagingpb.ChatGroup{
		Id:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		MemberIds:   group.MemberIDs,
		Roles:       group.Roles,
		Metadata:    group.Metadata,
		CreatedAt:   timestamppb.New(group.CreatedAt),
		UpdatedAt:   timestamppb.New(group.UpdatedAt),
	}
}

// Helper: map repository.MessageEvent to messagingpb.MessageEvent.
func mapRepoMessageEventToProto(event *MessageEvent) *messagingpb.MessageEvent {
	if event == nil {
		return nil
	}
	return &messagingpb.MessageEvent{
		EventId:        event.ID,
		MessageId:      event.MessageID,
		ThreadId:       "", // Not available in event, can be enriched if needed
		ConversationId: "",
		ChatGroupId:    "",
		EventType:      event.EventType,
		Payload:        nil, // Can unmarshal event.Payload to structpb.Struct if needed
		CreatedAt:      timestamppb.New(event.CreatedAt),
	}
}

// --- MessagingService RPCs ---

// [CANONICAL] All service methods must hydrate and return canonical metadata.
// All orchestration (success/error) must use the graceful pattern.
func (s *Service) SendMessage(ctx context.Context, req *messagingpb.SendMessageRequest) (*messagingpb.SendMessageResponse, error) {
	if req.Metadata == nil {
		req.Metadata = &commonpb.Metadata{}
	}
	// Canonical: Ensure versioning and business fields are set in service_specific.messaging
	if err := metadata.SetServiceSpecificField(req.Metadata, "messaging", "versioning", map[string]interface{}{
		"system_version":  "1.0.0",
		"service_version": "1.0.0",
		"environment":     "prod",
	}); err != nil {
		s.log.Warn("Failed to set service-specific metadata field (versioning)", zap.Error(err))
	}
	// Normalize metadata before sending to repo
	metaMap := metadata.ProtoToMap(req.Metadata)
	normMap := metadata.Handler{}.NormalizeAndCalculate(metaMap, "messaging", req.Content, nil, "success", "enrich messaging metadata")
	req.Metadata = metadata.MapToProto(normMap)
	msg, err := s.repo.SendMessage(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to send message", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := msg.Metadata
	resp := &messagingpb.SendMessageResponse{Message: mapRepoMessageToProto(msg)}
	success := graceful.WrapSuccess(ctx, codes.OK, "message sent", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) SendGroupMessage(ctx context.Context, req *messagingpb.SendGroupMessageRequest) (*messagingpb.SendGroupMessageResponse, error) {
	if req.Metadata == nil {
		req.Metadata = &commonpb.Metadata{}
	}
	msg, err := s.repo.SendGroupMessage(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to send group message", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"message_id": msg.ID,
		}, nil),
	}
	resp := &messagingpb.SendGroupMessageResponse{Message: mapRepoMessageToProto(msg)}
	success := graceful.WrapSuccess(ctx, codes.OK, "group message sent", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) EditMessage(ctx context.Context, req *messagingpb.EditMessageRequest) (*messagingpb.EditMessageResponse, error) {
	msg, err := s.repo.EditMessage(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to edit message", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"message_id": msg.ID,
		}, nil),
	}
	resp := &messagingpb.EditMessageResponse{Message: mapRepoMessageToProto(msg)}
	success := graceful.WrapSuccess(ctx, codes.OK, "message edited", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) DeleteMessage(ctx context.Context, req *messagingpb.DeleteMessageRequest) (*messagingpb.DeleteMessageResponse, error) {
	successVal, err := s.repo.DeleteMessageByRequest(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to delete message", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"message_id": req.MessageId,
		}, nil),
	}
	resp := &messagingpb.DeleteMessageResponse{Success: successVal}
	success := graceful.WrapSuccess(ctx, codes.OK, "message deleted", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) ReactToMessage(ctx context.Context, req *messagingpb.ReactToMessageRequest) (*messagingpb.ReactToMessageResponse, error) {
	msg, err := s.repo.ReactToMessage(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to react to message", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	// Minimal metadata for reaction
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"reaction": req.Emoji,
			"user_id":  req.UserId,
		}, nil),
	}
	resp := &messagingpb.ReactToMessageResponse{Message: mapRepoMessageToProto(msg)}
	success := graceful.WrapSuccess(ctx, codes.OK, "reaction added", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) GetMessage(ctx context.Context, req *messagingpb.GetMessageRequest) (*messagingpb.GetMessageResponse, error) {
	msg, err := s.repo.GetMessageByID(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.NotFound, "message not found", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"message_id": msg.ID,
		}, nil),
	}
	resp := &messagingpb.GetMessageResponse{Message: mapRepoMessageToProto(msg)}
	success := graceful.WrapSuccess(ctx, codes.OK, "message fetched", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) ListMessages(ctx context.Context, req *messagingpb.ListMessagesRequest) (*messagingpb.ListMessagesResponse, error) {
	msgs, total, err := s.repo.ListMessagesByFilter(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to list messages", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	protoMsgs := make([]*messagingpb.Message, 0, len(msgs))
	for _, m := range msgs {
		pm := mapRepoMessageToProto(m)
		// Canonical: hydrate messaging.Metadata from ServiceSpecific
		if pm.Metadata != nil && pm.Metadata.ServiceSpecific != nil {
			metaMap := metadata.StructToMap(pm.Metadata.ServiceSpecific)
			var msgMeta Metadata
			if raw, ok := metaMap["messaging"]; ok {
				metaBytes, err := json.Marshal(raw)
				if err != nil {
					s.log.Error("failed to marshal msgMeta", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to marshal msgMeta", err))
				}
				err = json.Unmarshal(metaBytes, &msgMeta)
				if err != nil {
					s.log.Error("failed to unmarshal metaBytes", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to unmarshal metaBytes", err))
				}
				if msgMeta.Versioning == nil {
					msgMeta.Versioning = &VersioningMetadata{SystemVersion: "1.0.0"}
				}
				metaMapOut := make(map[string]interface{})
				metaBytesOut, err := json.Marshal(msgMeta)
				if err != nil {
					s.log.Error("failed to marshal msgMeta", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to marshal msgMeta", err))
				}
				err = json.Unmarshal(metaBytesOut, &metaMapOut)
				if err != nil {
					s.log.Error("failed to unmarshal metaBytes", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to unmarshal metaBytes", err))
				}
				structMeta := metadata.MapToStruct(map[string]interface{}{"messaging": metaMapOut})
				pm.Metadata.ServiceSpecific = structMeta
			}
		}
		protoMsgs = append(protoMsgs, pm)
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	totalPages := 1
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"page":      req.Page,
			"page_size": pageSize,
			"total":     total,
		}, nil),
	}
	resp := &messagingpb.ListMessagesResponse{
		Messages:   protoMsgs,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32(totalPages),
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "messages listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) ListThreads(ctx context.Context, req *messagingpb.ListThreadsRequest) (*messagingpb.ListThreadsResponse, error) {
	threads, total, err := s.repo.ListThreadsByUser(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to list threads", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	protoThreads := make([]*messagingpb.Thread, 0, len(threads))
	for _, t := range threads {
		pt := mapRepoThreadToProto(t)
		if pt.Metadata != nil && pt.Metadata.ServiceSpecific != nil {
			metaMap := metadata.StructToMap(pt.Metadata.ServiceSpecific)
			var threadMeta Metadata
			if raw, ok := metaMap["messaging"]; ok {
				metaBytes, err := json.Marshal(raw)
				if err != nil {
					s.log.Error("failed to marshal msgMeta", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to marshal msgMeta", err))
				}
				err = json.Unmarshal(metaBytes, &threadMeta)
				if err != nil {
					s.log.Error("failed to unmarshal metaBytes", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to unmarshal metaBytes", err))
				}
				if threadMeta.Versioning == nil {
					threadMeta.Versioning = &VersioningMetadata{SystemVersion: "1.0.0"}
				}
				metaMapOut := make(map[string]interface{})
				metaBytesOut, err := json.Marshal(threadMeta)
				if err != nil {
					s.log.Error("failed to marshal msgMeta", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to marshal msgMeta", err))
				}
				err = json.Unmarshal(metaBytesOut, &metaMapOut)
				if err != nil {
					s.log.Error("failed to unmarshal metaBytes", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to unmarshal metaBytes", err))
				}
				structMeta := metadata.MapToStruct(map[string]interface{}{"messaging": metaMapOut})
				pt.Metadata.ServiceSpecific = structMeta
			}
		}
		protoThreads = append(protoThreads, pt)
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	totalPages := 1
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"page":      req.Page,
			"page_size": pageSize,
			"total":     total,
		}, nil),
	}
	resp := &messagingpb.ListThreadsResponse{
		Threads:    protoThreads,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32(totalPages),
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "threads listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) ListConversations(ctx context.Context, req *messagingpb.ListConversationsRequest) (*messagingpb.ListConversationsResponse, error) {
	convs, total, err := s.repo.ListConversationsByUser(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to list conversations", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	protoConvs := make([]*messagingpb.Conversation, 0, len(convs))
	for _, c := range convs {
		pc := mapRepoConversationToProto(c)
		if pc.Metadata != nil && pc.Metadata.ServiceSpecific != nil {
			metaMap := metadata.StructToMap(pc.Metadata.ServiceSpecific)
			var convMeta Metadata
			if raw, ok := metaMap["messaging"]; ok {
				metaBytes, err := json.Marshal(raw)
				if err != nil {
					s.log.Error("failed to marshal msgMeta", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to marshal msgMeta", err))
				}
				err = json.Unmarshal(metaBytes, &convMeta)
				if err != nil {
					s.log.Error("failed to unmarshal metaBytes", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to unmarshal metaBytes", err))
				}
				if convMeta.Versioning == nil {
					convMeta.Versioning = &VersioningMetadata{SystemVersion: "1.0.0"}
				}
				metaMapOut := make(map[string]interface{})
				metaBytesOut, err := json.Marshal(convMeta)
				if err != nil {
					s.log.Error("failed to marshal msgMeta", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to marshal msgMeta", err))
				}
				err = json.Unmarshal(metaBytesOut, &metaMapOut)
				if err != nil {
					s.log.Error("failed to unmarshal metaBytes", zap.Error(err))
					return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Internal, "failed to unmarshal metaBytes", err))
				}
				structMeta := metadata.MapToStruct(map[string]interface{}{"messaging": metaMapOut})
				pc.Metadata.ServiceSpecific = structMeta
			}
		}
		protoConvs = append(protoConvs, pc)
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	totalPages := 1
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"page":      req.Page,
			"page_size": pageSize,
			"total":     total,
		}, nil),
	}
	resp := &messagingpb.ListConversationsResponse{
		Conversations: protoConvs,
		TotalCount:    utils.ToInt32(total),
		Page:          req.Page,
		TotalPages:    utils.ToInt32(totalPages),
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "conversations listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) StreamMessages(req *messagingpb.StreamMessagesRequest, srv messagingpb.MessagingService_StreamMessagesServer) error {
	ctx := srv.Context()
	redisClient := s.cache.GetClient()
	if redisClient == nil {
		err := graceful.WrapErr(ctx, codes.Unavailable, "Redis client unavailable", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return graceful.ToStatusError(err)
	}

	// Determine channels to subscribe to
	channels := []string{}
	if req.UserId != "" {
		channels = append(channels, "messaging:events:user:"+req.UserId)
	}
	for _, convID := range req.ConversationIds {
		if convID != "" {
			channels = append(channels, "messaging:events:conversation:"+convID)
		}
	}
	for _, groupID := range req.ChatGroupIds {
		if groupID != "" {
			channels = append(channels, "messaging:events:chat_group:"+groupID)
		}
	}
	if len(channels) == 0 {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "No channels to subscribe to (user_id, conversation_ids, or chat_group_ids required)", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return graceful.ToStatusError(err)
	}

	// Optionally: send recent message history (last 20 messages per channel)
	// (For brevity, only for the first conversation or group)
	if len(req.ConversationIds) > 0 && req.ConversationIds[0] != "" {
		msgs, _, err := s.repo.ListMessagesByFilter(ctx, &messagingpb.ListMessagesRequest{
			ConversationId: req.ConversationIds[0],
			Page:           1,
			PageSize:       20,
		})
		if err == nil {
			for i := len(msgs) - 1; i >= 0; i-- { // send oldest first
				msg := msgs[i]
				// Fix: convert int64 CampaignID to string for proto
				campaignIDStr := ""
				if msg.CampaignID != 0 {
					campaignIDStr = strconv.FormatInt(msg.CampaignID, 10)
				}
				event := &messagingpb.MessageEvent{
					EventId:        "history-" + msg.ID,
					MessageId:      msg.ID,
					ThreadId:       msg.ThreadID,
					ConversationId: msg.ConversationID,
					ChatGroupId:    msg.ChatGroupID,
					EventType:      "history",
					Payload:        nil, // Optionally marshal msg.Content/metadata
					CreatedAt:      timestamppb.New(msg.CreatedAt),
					CampaignId:     campaignIDStr,
				}
				if err := srv.Send(event); err != nil {
					err = graceful.WrapErr(ctx, codes.Canceled, "client disconnected during history send", err)
					var ce *graceful.ContextError
					if errors.As(err, &ce) {
						ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
					}
					return graceful.ToStatusError(err)
				}
			}
		}
	}

	pubsub := redisClient.Subscribe(ctx, channels...)
	defer pubsub.Close()
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			meta := &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"stream": "StreamMessages closed by client"}, nil)}
			success := graceful.WrapSuccess(ctx, codes.OK, "StreamMessages closed", nil, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
			return nil
		case msg, ok := <-ch:
			if !ok {
				err := graceful.WrapErr(ctx, codes.Unavailable, "Redis pubsub channel closed", nil)
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				}
				return graceful.ToStatusError(err)
			}
			var event messagingpb.MessageEvent
			if err := metadata.UnmarshalCanonical([]byte(msg.Payload), &event); err != nil {
				s.log.Warn("Failed to unmarshal MessageEvent", zap.Error(err))
				continue
			}
			if err := srv.Send(&event); err != nil {
				err = graceful.WrapErr(ctx, codes.Canceled, "client disconnected during event send", err)
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				}
				return graceful.ToStatusError(err)
			}
		}
	}
}

func (s *Service) StreamTyping(req *messagingpb.StreamTypingRequest, srv messagingpb.MessagingService_StreamTypingServer) error {
	ctx := srv.Context()
	redisClient := s.cache.GetClient()
	if redisClient == nil {
		err := graceful.WrapErr(ctx, codes.Unavailable, "Redis client unavailable", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return graceful.ToStatusError(err)
	}
	// Determine channel
	var channel string
	switch {
	case req.ConversationId != "":
		channel = "messaging:events:typing:conversation:" + req.ConversationId
	case req.ChatGroupId != "":
		channel = "messaging:events:typing:chat_group:" + req.ChatGroupId
	default:
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "conversation_id or chat_group_id required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return graceful.ToStatusError(err)
	}
	pubsub := redisClient.Subscribe(ctx, channel)
	defer pubsub.Close()
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			meta := &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"stream": "StreamTyping closed by client"}, nil)}
			success := graceful.WrapSuccess(ctx, codes.OK, "StreamTyping closed", nil, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
			return nil
		case msg, ok := <-ch:
			if !ok {
				err := graceful.WrapErr(ctx, codes.Unavailable, "Redis pubsub channel closed", nil)
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				}
				return graceful.ToStatusError(err)
			}
			var event messagingpb.TypingEvent
			if err := metadata.UnmarshalCanonical([]byte(msg.Payload), &event); err != nil {
				s.log.Warn("Failed to unmarshal TypingEvent", zap.Error(err))
				continue
			}
			if err := srv.Send(&event); err != nil {
				err = graceful.WrapErr(ctx, codes.Canceled, "client disconnected during event send", err)
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				}
				return graceful.ToStatusError(err)
			}
		}
	}
}

func (s *Service) StreamPresence(req *messagingpb.StreamPresenceRequest, srv messagingpb.MessagingService_StreamPresenceServer) error {
	ctx := srv.Context()
	redisClient := s.cache.GetClient()
	if redisClient == nil {
		err := graceful.WrapErr(ctx, codes.Unavailable, "Redis client unavailable", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return graceful.ToStatusError(err)
	}
	// Determine channel
	var channel string
	switch {
	case req.UserId != "":
		channel = "messaging:events:presence:user:" + req.UserId
	case req.CampaignId != "":
		campaignIDInt, err := strconv.ParseInt(req.CampaignId, 10, 64)
		if err != nil {
			err := graceful.WrapErr(ctx, codes.InvalidArgument, "invalid campaign_id format", err)
			var ce *graceful.ContextError
			if errors.As(err, &ce) {
				ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
			}
			return graceful.ToStatusError(err)
		}
		channel = "messaging:events:presence:campaign:" + strconv.FormatInt(campaignIDInt, 10)
	default:
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "user_id or campaign_id required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return graceful.ToStatusError(err)
	}
	pubsub := redisClient.Subscribe(ctx, channel)
	defer pubsub.Close()
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			meta := &commonpb.Metadata{ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"stream": "StreamPresence closed by client"}, nil)}
			success := graceful.WrapSuccess(ctx, codes.OK, "StreamPresence closed", nil, nil)
			success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
			return nil
		case msg, ok := <-ch:
			if !ok {
				err := graceful.WrapErr(ctx, codes.Unavailable, "Redis pubsub channel closed", nil)
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				}
				return graceful.ToStatusError(err)
			}
			var event messagingpb.PresenceEvent
			if err := metadata.UnmarshalCanonical([]byte(msg.Payload), &event); err != nil {
				s.log.Warn("Failed to unmarshal PresenceEvent", zap.Error(err))
				continue
			}
			if err := srv.Send(&event); err != nil {
				err = graceful.WrapErr(ctx, codes.Canceled, "client disconnected during event send", err)
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				}
				return graceful.ToStatusError(err)
			}
		}
	}
}

func (s *Service) MarkAsRead(ctx context.Context, req *messagingpb.MarkAsReadRequest) (*messagingpb.MarkAsReadResponse, error) {
	successVal, err := s.repo.MarkAsRead(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to mark as read", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"message_id": req.MessageId,
			"user_id":    req.UserId,
		}, nil),
	}
	resp := &messagingpb.MarkAsReadResponse{Success: successVal}
	success := graceful.WrapSuccess(ctx, codes.OK, "message marked as read", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) MarkAsDelivered(ctx context.Context, req *messagingpb.MarkAsDeliveredRequest) (*messagingpb.MarkAsDeliveredResponse, error) {
	successVal, err := s.repo.MarkAsDelivered(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to mark as delivered", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"message_id": req.MessageId,
			"user_id":    req.UserId,
		}, nil),
	}
	resp := &messagingpb.MarkAsDeliveredResponse{Success: successVal}
	success := graceful.WrapSuccess(ctx, codes.OK, "message marked as delivered", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) AcknowledgeMessage(ctx context.Context, req *messagingpb.AcknowledgeMessageRequest) (*messagingpb.AcknowledgeMessageResponse, error) {
	successVal, err := s.repo.AcknowledgeMessage(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to acknowledge message", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"message_id": req.MessageId,
			"user_id":    req.UserId,
		}, nil),
	}
	resp := &messagingpb.AcknowledgeMessageResponse{Success: successVal}
	success := graceful.WrapSuccess(ctx, codes.OK, "message acknowledged", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) CreateChatGroup(ctx context.Context, req *messagingpb.CreateChatGroupRequest) (*messagingpb.CreateChatGroupResponse, error) {
	group, err := s.repo.CreateChatGroupWithRequest(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to create chat group", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"group_id": group.ID,
		}, nil),
	}
	resp := &messagingpb.CreateChatGroupResponse{ChatGroup: mapRepoChatGroupToProto(group)}
	success := graceful.WrapSuccess(ctx, codes.OK, "chat group created", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) AddChatGroupMember(ctx context.Context, req *messagingpb.AddChatGroupMemberRequest) (*messagingpb.AddChatGroupMemberResponse, error) {
	group, err := s.repo.AddChatGroupMember(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to add chat group member", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"group_id": group.ID,
			"user_id":  req.UserId,
		}, nil),
	}
	resp := &messagingpb.AddChatGroupMemberResponse{ChatGroup: mapRepoChatGroupToProto(group)}
	success := graceful.WrapSuccess(ctx, codes.OK, "chat group member added", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) RemoveChatGroupMember(ctx context.Context, req *messagingpb.RemoveChatGroupMemberRequest) (*messagingpb.RemoveChatGroupMemberResponse, error) {
	group, err := s.repo.RemoveChatGroupMember(ctx, req)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to remove chat group member", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"group_id": group.ID,
			"user_id":  req.UserId,
		}, nil),
	}
	resp := &messagingpb.RemoveChatGroupMemberResponse{ChatGroup: mapRepoChatGroupToProto(group)}
	success := graceful.WrapSuccess(ctx, codes.OK, "chat group member removed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) ListChatGroupMembers(ctx context.Context, req *messagingpb.ListChatGroupMembersRequest) (*messagingpb.ListChatGroupMembersResponse, error) {
	group, err := s.repo.GetChatGroupByID(ctx, req.ChatGroupId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.NotFound, "chat group not found", err)
		s.log.Warn("failed to fetch chat group", zap.Error(err))
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	members := group.MemberIDs
	total := len(members)
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	pagedMembers := members[start:end]
	totalPages := 1
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"group_id":  req.ChatGroupId,
			"page":      page,
			"page_size": pageSize,
			"total":     total,
		}, nil),
	}
	resp := &messagingpb.ListChatGroupMembersResponse{
		MemberIds:  pagedMembers,
		TotalCount: utils.ToInt32(total),
		Page:       utils.ToInt32(page),
		TotalPages: utils.ToInt32(totalPages),
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "chat group members listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) UpdateMessagingPreferences(ctx context.Context, req *messagingpb.UpdateMessagingPreferencesRequest) (*messagingpb.UpdateMessagingPreferencesResponse, error) {
	if req.UserId == "" || req.Preferences == nil {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "user_id and preferences required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	err := s.repo.UpdateMessagingPreferences(ctx, req.UserId, req.Preferences)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to update preferences", err)
		s.log.Warn("failed to update messaging preferences", zap.Error(err))
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	prefs, updatedAt, err := s.repo.GetMessagingPreferences(ctx, req.UserId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to fetch updated preferences", err)
		s.log.Warn("failed to fetch updated preferences", zap.Error(err))
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"user_id": req.UserId,
		}, nil),
	}
	resp := &messagingpb.UpdateMessagingPreferencesResponse{
		Preferences: prefs,
		UpdatedAt:   updatedAt,
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "messaging preferences updated", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}

func (s *Service) ListMessageEvents(ctx context.Context, req *messagingpb.ListMessageEventsRequest) (*messagingpb.ListMessageEventsResponse, error) {
	if req.UserId == "" {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "user_id required", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize
	msgEvents, total, err := s.repo.ListMessageEventsByUser(ctx, req.UserId, pageSize, offset)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to list message events", err)
		s.log.Warn("failed to list message events", zap.Error(err))
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	protoEvents := make([]*messagingpb.MessageEvent, 0, len(msgEvents))
	for _, e := range msgEvents {
		protoEvents = append(protoEvents, mapRepoMessageEventToProto(e))
	}
	totalPages := 1
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	meta := &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{
			"user_id":   req.UserId,
			"page":      page,
			"page_size": pageSize,
			"total":     total,
		}, nil),
	}
	resp := &messagingpb.ListMessageEventsResponse{
		Events:     protoEvents,
		TotalCount: utils.ToInt32(total),
		Page:       utils.ToInt32(page),
		TotalPages: utils.ToInt32(totalPages),
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "message events listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Metadata: meta})
	return resp, nil
}
