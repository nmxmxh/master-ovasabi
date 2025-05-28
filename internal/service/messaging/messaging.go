// Messaging Service Implementation
// -------------------------------
//
// This file implements the MessagingService gRPC interface, following the robust service pattern.
// It uses dependency injection for logger, repository, and cache, and is ready for extensibility.
//
// See docs/amadeus/amadeus_context.md for service standards.

package messaging

import (
	context "context"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	messagingpb "github.com/nmxmxh/master-ovasabi/api/protos/messaging/v1"
	messageEvents "github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service implements the MessagingService gRPC interface.
type Service struct {
	messagingpb.UnimplementedMessagingServiceServer
	log          *zap.Logger
	repo         *Repository
	cache        *redis.Cache
	eventEmitter EventEmitter
	eventEnabled bool
}

// NewService creates a new MessagingService instance with event bus support.
func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) messagingpb.MessagingServiceServer {
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

func (s *Service) SendMessage(ctx context.Context, req *messagingpb.SendMessageRequest) (*messagingpb.SendMessageResponse, error) {
	if req.Metadata == nil {
		req.Metadata = &commonpb.Metadata{}
	}
	msg, err := s.repo.SendMessage(ctx, req)
	if err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "sender_id": req.SenderId})
			errMeta := &commonpb.Metadata{ServiceSpecific: errStruct}
			_, ok := messageEvents.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.cache, "message", "messaging.send_failed", req.SenderId, errMeta, zap.Error(err))
			if !ok {
				s.log.Error("Failed to emit messaging.send_failed event", zap.Error(err))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to send message: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		successStruct := metadata.NewStructFromMap(map[string]interface{}{"message_id": msg.ID})
		successMeta := &commonpb.Metadata{ServiceSpecific: successStruct}
		_, ok := messageEvents.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.cache, "message", "messaging.sent", msg.ID, successMeta, zap.String("message_id", msg.ID))
		if !ok {
			s.log.Error("Failed to emit messaging.sent event", zap.String("message_id", msg.ID))
		}
	}
	return &messagingpb.SendMessageResponse{Message: mapRepoMessageToProto(msg)}, nil
}

func (s *Service) SendGroupMessage(ctx context.Context, req *messagingpb.SendGroupMessageRequest) (*messagingpb.SendGroupMessageResponse, error) {
	if req.Metadata == nil {
		req.Metadata = &commonpb.Metadata{}
	}
	msg, err := s.repo.SendGroupMessage(ctx, req)
	if err != nil {
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{
				"error":     err.Error(),
				"sender_id": req.SenderId,
				"content":   req.Content,
			})
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			_, ok := messageEvents.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.cache, "message", "messaging.group_send_failed", "", errMeta)
			if !ok {
				s.log.Warn("Failed to emit messaging.group_send_failed event", zap.Error(err))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to send group message: %v", err)
	}
	// Emit messaging.group_sent event after successful send
	if s.eventEnabled && s.eventEmitter != nil {
		msg.Metadata, _ = messageEvents.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.cache, "message", "messaging.group_sent", msg.ID, msg.Metadata)
	}
	return &messagingpb.SendGroupMessageResponse{Message: mapRepoMessageToProto(msg)}, nil
}

func (s *Service) EditMessage(ctx context.Context, req *messagingpb.EditMessageRequest) (*messagingpb.EditMessageResponse, error) {
	msg, err := s.repo.EditMessage(ctx, req)
	if err != nil {
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{
				"error":      err.Error(),
				"message_id": req.MessageId,
			})
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			_, ok := messageEvents.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.cache, "message", "messaging.edit_failed", req.MessageId, errMeta)
			if !ok {
				s.log.Warn("Failed to emit messaging.edit_failed event", zap.Error(err))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to edit message: %v", err)
	}
	// Emit messaging.edited event after successful edit
	if s.eventEnabled && s.eventEmitter != nil {
		msg.Metadata, _ = messageEvents.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.cache, "message", "messaging.edited", msg.ID, msg.Metadata)
	}
	return &messagingpb.EditMessageResponse{Message: mapRepoMessageToProto(msg)}, nil
}

func (s *Service) DeleteMessage(ctx context.Context, req *messagingpb.DeleteMessageRequest) (*messagingpb.DeleteMessageResponse, error) {
	success, err := s.repo.DeleteMessageByRequest(ctx, req)
	if err != nil {
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{
				"error":      err.Error(),
				"message_id": req.MessageId,
			})
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			_, ok := messageEvents.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.cache, "message", "messaging.delete_failed", req.MessageId, errMeta)
			if !ok {
				s.log.Warn("Failed to emit messaging.delete_failed event", zap.Error(err))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to delete message: %v", err)
	}
	// Emit messaging.deleted event after successful deletion
	if s.eventEnabled && s.eventEmitter != nil {
		_, ok := messageEvents.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.cache, "message", "messaging.deleted", req.MessageId, nil)
		if !ok {
			s.log.Warn("Failed to emit messaging.deleted event", zap.Error(err))
		}
	}
	return &messagingpb.DeleteMessageResponse{Success: success}, nil
}

func (s *Service) ReactToMessage(ctx context.Context, req *messagingpb.ReactToMessageRequest) (*messagingpb.ReactToMessageResponse, error) {
	msg, err := s.repo.ReactToMessage(ctx, req)
	if err != nil {
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct := metadata.NewStructFromMap(map[string]interface{}{
				"error":      err.Error(),
				"message_id": req.MessageId,
			})
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			_, ok := messageEvents.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.cache, "message", "messaging.react_failed", req.MessageId, errMeta)
			if !ok {
				s.log.Warn("Failed to emit messaging.react_failed event", zap.Error(err))
			}
		}
		return nil, status.Errorf(codes.Internal, "failed to react to message: %v", err)
	}
	// Emit messaging.reacted event after successful reaction
	if s.eventEnabled && s.eventEmitter != nil {
		msg.Metadata, _ = messageEvents.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.cache, "message", "messaging.reacted", msg.ID, msg.Metadata)
	}
	return &messagingpb.ReactToMessageResponse{Message: mapRepoMessageToProto(msg)}, nil
}

func (s *Service) GetMessage(ctx context.Context, req *messagingpb.GetMessageRequest) (*messagingpb.GetMessageResponse, error) {
	msg, err := s.repo.GetMessageByID(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "message not found: %v", err)
	}
	return &messagingpb.GetMessageResponse{Message: mapRepoMessageToProto(msg)}, nil
}

func (s *Service) ListMessages(ctx context.Context, req *messagingpb.ListMessagesRequest) (*messagingpb.ListMessagesResponse, error) {
	msgs, total, err := s.repo.ListMessagesByFilter(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list messages: %v", err)
	}
	protoMsgs := make([]*messagingpb.Message, 0, len(msgs))
	for _, m := range msgs {
		pm := mapRepoMessageToProto(m)
		// Enrich metadata using canonical helpers
		if pm.Metadata != nil {
			meta := ExtractMessagingMetadata(pm.Metadata)
			// Example enrichment: ensure versioning is set
			if meta.Versioning == nil {
				meta.Versioning = &VersioningMetadata{SystemVersion: "1.0.0"}
			}
			structMeta, err := meta.ToStruct()
			if err != nil {
				s.log.Warn("failed to convert metadata to struct", zap.Error(err))
			} else if pm.Metadata.ServiceSpecific == nil {
				pm.Metadata.ServiceSpecific = structMeta
			}
			if pm.Metadata.ServiceSpecific == nil {
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
	return &messagingpb.ListMessagesResponse{
		Messages:   protoMsgs,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32(totalPages),
	}, nil
}

func (s *Service) ListThreads(ctx context.Context, req *messagingpb.ListThreadsRequest) (*messagingpb.ListThreadsResponse, error) {
	threads, total, err := s.repo.ListThreadsByUser(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list threads: %v", err)
	}
	protoThreads := make([]*messagingpb.Thread, 0, len(threads))
	for _, t := range threads {
		pt := mapRepoThreadToProto(t)
		// Enrich metadata
		if pt.Metadata != nil {
			meta := ExtractMessagingMetadata(pt.Metadata)
			if meta.Versioning == nil {
				meta.Versioning = &VersioningMetadata{SystemVersion: "1.0.0"}
			}
			structMeta, err := meta.ToStruct()
			if err != nil {
				s.log.Warn("failed to convert metadata to struct", zap.Error(err))
			} else if pt.Metadata.ServiceSpecific == nil {
				pt.Metadata.ServiceSpecific = structMeta
			}
			if pt.Metadata.ServiceSpecific == nil {
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
	return &messagingpb.ListThreadsResponse{
		Threads:    protoThreads,
		TotalCount: utils.ToInt32(total),
		Page:       req.Page,
		TotalPages: utils.ToInt32(totalPages),
	}, nil
}

func (s *Service) ListConversations(ctx context.Context, req *messagingpb.ListConversationsRequest) (*messagingpb.ListConversationsResponse, error) {
	convs, total, err := s.repo.ListConversationsByUser(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list conversations: %v", err)
	}
	protoConvs := make([]*messagingpb.Conversation, 0, len(convs))
	for _, c := range convs {
		pc := mapRepoConversationToProto(c)
		// Enrich metadata
		if pc.Metadata != nil {
			meta := ExtractMessagingMetadata(pc.Metadata)
			if meta.Versioning == nil {
				meta.Versioning = &VersioningMetadata{SystemVersion: "1.0.0"}
			}
			structMeta, err := meta.ToStruct()
			if err != nil {
				s.log.Warn("failed to convert metadata to struct", zap.Error(err))
			} else if pc.Metadata.ServiceSpecific == nil {
				pc.Metadata.ServiceSpecific = structMeta
			}
			if pc.Metadata.ServiceSpecific == nil {
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
	return &messagingpb.ListConversationsResponse{
		Conversations: protoConvs,
		TotalCount:    utils.ToInt32(total),
		Page:          req.Page,
		TotalPages:    utils.ToInt32(totalPages),
	}, nil
}

func (s *Service) StreamMessages(_ *messagingpb.StreamMessagesRequest, _ messagingpb.MessagingService_StreamMessagesServer) error {
	// TODO: Implement stream messages logic
	return nil
}

func (s *Service) StreamTyping(_ *messagingpb.StreamTypingRequest, _ messagingpb.MessagingService_StreamTypingServer) error {
	// TODO: Implement stream typing logic
	return nil
}

func (s *Service) StreamPresence(_ *messagingpb.StreamPresenceRequest, _ messagingpb.MessagingService_StreamPresenceServer) error {
	// TODO: Implement stream presence logic
	return nil
}

func (s *Service) MarkAsRead(ctx context.Context, req *messagingpb.MarkAsReadRequest) (*messagingpb.MarkAsReadResponse, error) {
	success, err := s.repo.MarkAsRead(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mark as read: %v", err)
	}
	return &messagingpb.MarkAsReadResponse{Success: success}, nil
}

func (s *Service) MarkAsDelivered(ctx context.Context, req *messagingpb.MarkAsDeliveredRequest) (*messagingpb.MarkAsDeliveredResponse, error) {
	success, err := s.repo.MarkAsDelivered(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to mark as delivered: %v", err)
	}
	return &messagingpb.MarkAsDeliveredResponse{Success: success}, nil
}

func (s *Service) AcknowledgeMessage(ctx context.Context, req *messagingpb.AcknowledgeMessageRequest) (*messagingpb.AcknowledgeMessageResponse, error) {
	success, err := s.repo.AcknowledgeMessage(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to acknowledge message: %v", err)
	}
	return &messagingpb.AcknowledgeMessageResponse{Success: success}, nil
}

func (s *Service) CreateChatGroup(ctx context.Context, req *messagingpb.CreateChatGroupRequest) (*messagingpb.CreateChatGroupResponse, error) {
	group, err := s.repo.CreateChatGroupWithRequest(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create chat group: %v", err)
	}
	return &messagingpb.CreateChatGroupResponse{ChatGroup: mapRepoChatGroupToProto(group)}, nil
}

func (s *Service) AddChatGroupMember(ctx context.Context, req *messagingpb.AddChatGroupMemberRequest) (*messagingpb.AddChatGroupMemberResponse, error) {
	group, err := s.repo.AddChatGroupMember(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add chat group member: %v", err)
	}
	return &messagingpb.AddChatGroupMemberResponse{ChatGroup: mapRepoChatGroupToProto(group)}, nil
}

func (s *Service) RemoveChatGroupMember(ctx context.Context, req *messagingpb.RemoveChatGroupMemberRequest) (*messagingpb.RemoveChatGroupMemberResponse, error) {
	group, err := s.repo.RemoveChatGroupMember(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to remove chat group member: %v", err)
	}
	return &messagingpb.RemoveChatGroupMemberResponse{ChatGroup: mapRepoChatGroupToProto(group)}, nil
}

func (s *Service) ListChatGroupMembers(ctx context.Context, req *messagingpb.ListChatGroupMembersRequest) (*messagingpb.ListChatGroupMembersResponse, error) {
	// Fetch the chat group by ID
	group, err := s.repo.GetChatGroupByID(ctx, req.ChatGroupId)
	if err != nil {
		s.log.Warn("failed to fetch chat group", zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "chat group not found: %v", err)
	}
	// Paginate member IDs
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
	return &messagingpb.ListChatGroupMembersResponse{
		MemberIds:  pagedMembers,
		TotalCount: utils.ToInt32(total),
		Page:       utils.ToInt32(page),
		TotalPages: utils.ToInt32(totalPages),
	}, nil
}

func (s *Service) UpdateMessagingPreferences(ctx context.Context, req *messagingpb.UpdateMessagingPreferencesRequest) (*messagingpb.UpdateMessagingPreferencesResponse, error) {
	if req.UserId == "" || req.Preferences == nil {
		return nil, status.Error(codes.InvalidArgument, "user_id and preferences required")
	}
	err := s.repo.UpdateMessagingPreferences(ctx, req.UserId, req.Preferences)
	if err != nil {
		s.log.Warn("failed to update messaging preferences", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update preferences: %v", err)
	}
	prefs, updatedAt, err := s.repo.GetMessagingPreferences(ctx, req.UserId)
	if err != nil {
		s.log.Warn("failed to fetch updated preferences", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to fetch updated preferences: %v", err)
	}
	return &messagingpb.UpdateMessagingPreferencesResponse{
		Preferences: prefs,
		UpdatedAt:   updatedAt,
	}, nil
}

func (s *Service) ListMessageEvents(ctx context.Context, req *messagingpb.ListMessageEventsRequest) (*messagingpb.ListMessageEventsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id required")
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
	events, total, err := s.repo.ListMessageEventsByUser(ctx, req.UserId, pageSize, offset)
	if err != nil {
		s.log.Warn("failed to list message events", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to list message events: %v", err)
	}
	protoEvents := make([]*messagingpb.MessageEvent, 0, len(events))
	for _, e := range events {
		protoEvents = append(protoEvents, mapRepoMessageEventToProto(e))
	}
	totalPages := 1
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return &messagingpb.ListMessageEventsResponse{
		Events:     protoEvents,
		TotalCount: utils.ToInt32(total),
		Page:       utils.ToInt32(page),
		TotalPages: utils.ToInt32(totalPages),
	}, nil
}
