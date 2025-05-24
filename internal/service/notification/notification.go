package notification

import (
	context "context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	notificationEvents "github.com/nmxmxh/master-ovasabi/pkg/events"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	structpb "google.golang.org/protobuf/types/known/structpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// Notification Service: Azure-Optimized Provider Integration
//
// This service uses:
//   - AzureEmailProvider for email (from email_provider.go)
//   - AzureSMSProvider for SMS (from sms_provider.go)
//   - AzurePushProvider for push (from push_provider.go)
//
// Providers are initialized from environment variables for easy configuration.

type Service struct {
	notificationpb.UnimplementedNotificationServiceServer
	log          *zap.Logger
	repo         *Repository
	cache        *redis.Cache
	eventEmitter EventEmitter
	eventEnabled bool
}

func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) notificationpb.NotificationServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

func (s *Service) GetNotification(ctx context.Context, req *notificationpb.GetNotificationRequest) (*notificationpb.GetNotificationResponse, error) {
	id := parseInt64(req.NotificationId)
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.log.Error("Failed to get notification", zap.Error(err))
		return nil, grpcstatus.Errorf(codes.NotFound, "notification not found: %v", err)
	}
	return &notificationpb.GetNotificationResponse{
		Notification: mapNotificationToProto(notification),
	}, nil
}

func (s *Service) ListNotifications(ctx context.Context, req *notificationpb.ListNotificationsRequest) (*notificationpb.ListNotificationsResponse, error) {
	userID := parseInt64(req.UserId)
	notifications, err := s.repo.ListByUserID(ctx, userID, int(req.PageSize), int(req.Page))
	if err != nil {
		s.log.Error("Failed to list notifications", zap.Error(err))
		return nil, grpcstatus.Errorf(codes.Internal, "failed to list notifications: %v", err)
	}
	protoNotifications := make([]*notificationpb.Notification, 0, len(notifications))
	for _, n := range notifications {
		protoNotifications = append(protoNotifications, mapNotificationToProto(n))
	}
	var totalCount int32
	if len(protoNotifications) > math.MaxInt32 {
		totalCount = math.MaxInt32
	} else {
		totalCount = int32(math.Min(float64(len(protoNotifications)), float64(math.MaxInt32)))
	}
	return &notificationpb.ListNotificationsResponse{
		Notifications: protoNotifications,
		TotalCount:    totalCount,
		Page:          req.Page,
		TotalPages:    1, // TODO: calculate real total pages
	}, nil
}

func (s *Service) SendNotification(ctx context.Context, req *notificationpb.SendNotificationRequest) (*notificationpb.SendNotificationResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Metadata); err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create error struct for notification.failed event", zap.Error(err))
				return nil, grpcstatus.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "notification.failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit notification.failed event", zap.Error(errEmit))
			}
		}
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	notification := &Notification{
		UserID:     parseInt64(req.UserId),
		CampaignID: req.CampaignId,
		Type:       Type(req.Channel),
		Title:      req.Title,
		Content:    req.Body,
		Status:     StatusPending,
		Metadata:   req.Metadata,
	}
	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		s.log.Error("Failed to send notification", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create error struct for notification.failed event", zap.Error(err))
				return nil, grpcstatus.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "notification.failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit notification.failed event", zap.Error(errEmit))
			}
		}
		return nil, grpcstatus.Errorf(codes.Internal, "failed to send notification: %v", err)
	}
	if s.cache != nil && created.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.cache, "notification", fmt.Sprint(created.ID), created.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "notification", fmt.Sprint(created.ID), created.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "notification", fmt.Sprint(created.ID), created.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "notification", created.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	if s.eventEnabled && s.eventEmitter != nil {
		created.Metadata, _ = notificationEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "notification.sent", fmt.Sprint(created.ID), created.Metadata)
	}
	return &notificationpb.SendNotificationResponse{
		Notification: mapNotificationToProto(created),
		Status:       "created",
	}, nil
}

func (s *Service) SendEmail(ctx context.Context, req *notificationpb.SendEmailRequest) (*notificationpb.SendEmailResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Metadata); err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create error struct for notification.failed event", zap.Error(err))
				return nil, grpcstatus.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "notification.failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit notification.failed event", zap.Error(errEmit))
			}
		}
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	notification := &Notification{
		UserID:     0, // Not mapped directly
		CampaignID: req.CampaignId,
		Type:       TypeEmail,
		Title:      req.Subject,
		Content:    req.Body,
		Status:     StatusPending,
		Metadata:   req.Metadata,
	}
	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		s.log.Error("Failed to save email notification", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create error struct for notification.failed event", zap.Error(err))
				return nil, grpcstatus.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "notification.failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit notification.failed event", zap.Error(errEmit))
			}
		}
		return nil, grpcstatus.Errorf(codes.Internal, "failed to save email notification: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		created.Metadata, _ = notificationEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "notification.sent", fmt.Sprint(created.ID), created.Metadata)
	}
	return &notificationpb.SendEmailResponse{
		MessageId: fmt.Sprint(created.ID),
		Status:    string(created.Status),
		SentAt:    created.SentAt.Unix(),
	}, nil
}

func (s *Service) SendSMS(ctx context.Context, req *notificationpb.SendSMSRequest) (*notificationpb.SendSMSResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Metadata); err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create error struct for notification.failed event", zap.Error(err))
				return nil, grpcstatus.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "notification.failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit notification.failed event", zap.Error(errEmit))
			}
		}
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	notification := &Notification{
		UserID:     0, // Not mapped directly
		CampaignID: req.CampaignId,
		Type:       TypeSMS,
		Title:      "SMS",
		Content:    req.Message,
		Status:     StatusPending,
		Metadata:   req.Metadata,
	}
	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		s.log.Error("Failed to save SMS notification", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create error struct for notification.failed event", zap.Error(err))
				return nil, grpcstatus.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "notification.failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit notification.failed event", zap.Error(errEmit))
			}
		}
		return nil, grpcstatus.Errorf(codes.Internal, "failed to save SMS notification: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		created.Metadata, _ = notificationEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "notification.sent", fmt.Sprint(created.ID), created.Metadata)
	}
	return &notificationpb.SendSMSResponse{
		MessageId: fmt.Sprint(created.ID),
		Status:    string(created.Status),
		SentAt:    created.SentAt.Unix(),
	}, nil
}

func (s *Service) SendPushNotification(ctx context.Context, req *notificationpb.SendPushNotificationRequest) (*notificationpb.SendPushNotificationResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Metadata); err != nil {
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create error struct for notification.failed event", zap.Error(err))
				return nil, grpcstatus.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "notification.failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit notification.failed event", zap.Error(errEmit))
			}
		}
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	notification := &Notification{
		UserID:     parseInt64(req.UserId),
		CampaignID: req.CampaignId,
		Type:       TypePush,
		Title:      req.Title,
		Content:    req.Message,
		Status:     StatusPending,
		Metadata:   req.Metadata,
	}
	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		s.log.Error("Failed to save push notification", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error()})
			if err != nil {
				s.log.Error("Failed to create error struct for notification.failed event", zap.Error(err))
				return nil, grpcstatus.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			errEmit := s.eventEmitter.EmitEvent(ctx, "notification.failed", "", errMeta)
			if errEmit != nil {
				s.log.Warn("Failed to emit notification.failed event", zap.Error(errEmit))
			}
		}
		return nil, grpcstatus.Errorf(codes.Internal, "failed to save push notification: %v", err)
	}
	if s.eventEnabled && s.eventEmitter != nil {
		created.Metadata, _ = notificationEvents.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "notification.sent", fmt.Sprint(created.ID), created.Metadata)
	}
	return &notificationpb.SendPushNotificationResponse{
		NotificationId: fmt.Sprint(created.ID),
		Status:         string(created.Status),
		SentAt:         created.SentAt.Unix(),
	}, nil
}

func (s *Service) BroadcastEvent(ctx context.Context, req *notificationpb.BroadcastEventRequest) (*notificationpb.BroadcastEventResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Payload); err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "invalid payload: %v", err)
	}
	broadcast := &Notification{
		UserID:      0,              // system/campaign
		CampaignID:  req.CampaignId, // now campaign-scoped
		Type:        TypeInApp,
		Title:       req.Subject,
		Content:     req.Message,
		Status:      StatusPending,
		Metadata:    req.Payload,
		ScheduledAt: toTimePtr(req.ScheduledAt),
	}
	created, err := s.repo.CreateBroadcast(ctx, broadcast)
	if err != nil {
		s.log.Error("Failed to create broadcast", zap.Error(err))
		return nil, grpcstatus.Errorf(codes.Internal, "failed to create broadcast: %v", err)
	}
	if s.cache != nil && created.Metadata != nil {
		if err := pattern.CacheMetadata(ctx, s.log, s.cache, "notification", fmt.Sprint(created.ID), created.Metadata, 10*time.Minute); err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	if err := pattern.RegisterSchedule(ctx, s.log, "notification", fmt.Sprint(created.ID), created.Metadata); err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	if err := pattern.EnrichKnowledgeGraph(ctx, s.log, "notification", fmt.Sprint(created.ID), created.Metadata); err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	if err := pattern.RegisterWithNexus(ctx, s.log, "notification", created.Metadata); err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	return &notificationpb.BroadcastEventResponse{
		BroadcastId: fmt.Sprint(created.ID),
		Status:      "created",
		CampaignId:  created.CampaignID,
	}, nil
}

func (s *Service) ListNotificationEvents(ctx context.Context, req *notificationpb.ListNotificationEventsRequest) (*notificationpb.ListNotificationEventsResponse, error) {
	notificationID := parseInt64(req.NotificationId)
	events, err := s.repo.ListNotificationEvents(ctx, notificationID, int(req.PageSize), int(req.Page))
	if err != nil {
		s.log.Error("Failed to list notification events", zap.Error(err))
		return nil, grpcstatus.Errorf(codes.Internal, "failed to list events: %v", err)
	}
	protoEvents := make([]*notificationpb.NotificationEvent, 0, len(events))
	for _, e := range events {
		protoEvents = append(protoEvents, &notificationpb.NotificationEvent{
			EventId:        fmt.Sprint(e.ID),
			NotificationId: fmt.Sprint(e.NotificationID),
			UserId:         fmt.Sprint(e.UserID),
			EventType:      e.EventType,
			// Payload: ... (unmarshal if needed)
			// CreatedAt: ...
		})
	}
	var totalEvents int32
	if len(protoEvents) > math.MaxInt32 {
		totalEvents = math.MaxInt32
	} else {
		totalEvents = int32(math.Min(float64(len(protoEvents)), float64(math.MaxInt32)))
	}
	return &notificationpb.ListNotificationEventsResponse{
		Events: protoEvents,
		Total:  totalEvents,
	}, nil
}

func (s *Service) AcknowledgeNotification(ctx context.Context, req *notificationpb.AcknowledgeNotificationRequest) (*notificationpb.AcknowledgeNotificationResponse, error) {
	id := parseInt64(req.NotificationId)
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.log.Error("Notification not found for acknowledge", zap.Error(err))
		return nil, grpcstatus.Errorf(codes.NotFound, "notification not found: %v", err)
	}
	notification.Status = StatusSent // or a new 'read' status if supported
	if err := s.repo.Update(ctx, notification); err != nil {
		s.log.Error("Failed to update notification status to read", zap.Error(err))
		return nil, grpcstatus.Errorf(codes.Internal, "failed to acknowledge notification: %v", err)
	}
	return &notificationpb.AcknowledgeNotificationResponse{Status: "acknowledged"}, nil
}

func (s *Service) UpdateNotificationPreferences(_ context.Context, _ *notificationpb.UpdateNotificationPreferencesRequest) (*notificationpb.UpdateNotificationPreferencesResponse, error) {
	// TODO: Implement notification preferences update
	return nil, errors.New("not implemented")
}

func (s *Service) SubscribeToEvents(req *notificationpb.SubscribeToEventsRequest, stream notificationpb.NotificationService_SubscribeToEventsServer) error {
	// Example: send a single dummy event, then return
	event := &notificationpb.NotificationEvent{
		EventId:        "1",
		NotificationId: "1",
		UserId:         req.UserId,
		EventType:      "delivered",
		Payload:        nil,
		CreatedAt:      timestamppb.Now(),
	}
	if err := stream.Send(event); err != nil {
		s.log.Error("Failed to send event in stream", zap.Error(err))
		return err
	}
	return nil // End of stream
}

func (s *Service) StreamAssetChunks(req *notificationpb.StreamAssetChunksRequest, stream notificationpb.NotificationService_StreamAssetChunksServer) error {
	// Example: send a single dummy chunk, then return
	chunk := &notificationpb.AssetChunk{
		UploadId: req.AssetId,
		Data:     []byte("example data"),
		Sequence: 1,
	}
	if err := stream.Send(chunk); err != nil {
		s.log.Error("Failed to send asset chunk in stream", zap.Error(err))
		return err
	}
	return nil // End of stream
}

// --- Helpers ---.
func parseInt64(s string) int64 {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		// Log the error and return 0 as a fallback
		fmt.Printf("Failed to parse int64 from '%s': %v\n", s, err)
		return 0
	}
	return id
}

func toTimePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

func mapNotificationToProto(n *Notification) *notificationpb.Notification {
	if n == nil {
		return nil
	}
	return &notificationpb.Notification{
		Id:        fmt.Sprint(n.ID),
		UserId:    fmt.Sprint(n.UserID),
		Channel:   string(n.Type),
		Title:     n.Title,
		Body:      n.Content,
		Status:    mapStatusToProto(n.Status),
		CreatedAt: timestamppb.New(n.CreatedAt),
		UpdatedAt: timestamppb.New(n.UpdatedAt),
		Read:      false, // TODO: map read status
	}
}

func mapStatusToProto(s Status) notificationpb.NotificationStatus {
	switch s {
	case StatusPending:
		return notificationpb.NotificationStatus_NOTIFICATION_STATUS_PENDING
	case StatusSent:
		return notificationpb.NotificationStatus_NOTIFICATION_STATUS_SENT
	case StatusFailed:
		return notificationpb.NotificationStatus_NOTIFICATION_STATUS_FAILED
	case StatusCancelled:
		return notificationpb.NotificationStatus_NOTIFICATION_STATUS_UNSPECIFIED
	default:
		return notificationpb.NotificationStatus_NOTIFICATION_STATUS_UNSPECIFIED
	}
}
