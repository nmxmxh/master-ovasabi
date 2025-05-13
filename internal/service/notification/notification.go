package notification

import (
	context "context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	notificationrepo "github.com/nmxmxh/master-ovasabi/internal/repository/notification"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	notificationpb.UnimplementedNotificationServiceServer
	log   *zap.Logger
	repo  *notificationrepo.NotificationRepository
	cache *redis.Cache
}

func NewNotificationService(log *zap.Logger, repo *notificationrepo.NotificationRepository, cache *redis.Cache) notificationpb.NotificationServiceServer {
	return &Service{
		log:   log,
		repo:  repo,
		cache: cache,
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
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	notification := &notificationrepo.Notification{
		UserID:   parseInt64(req.UserId),
		Type:     notificationrepo.NotificationType(req.Channel),
		Title:    req.Title,
		Content:  req.Body,
		Status:   notificationrepo.NotificationStatusPending,
		Metadata: req.Metadata,
	}
	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		s.log.Error("Failed to send notification", zap.Error(err))
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
	return &notificationpb.SendNotificationResponse{
		Notification: mapNotificationToProto(created),
		Status:       "created",
	}, nil
}

func (s *Service) SendEmail(ctx context.Context, req *notificationpb.SendEmailRequest) (*notificationpb.SendEmailResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Metadata); err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	notification := &notificationrepo.Notification{
		UserID:   0, // Not mapped directly
		Type:     notificationrepo.NotificationTypeEmail,
		Title:    req.Subject,
		Content:  req.Body,
		Status:   notificationrepo.NotificationStatusPending,
		Metadata: req.Metadata,
	}
	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		s.log.Error("Failed to send email notification", zap.Error(err))
		return nil, grpcstatus.Errorf(codes.Internal, "failed to send email: %v", err)
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
	return &notificationpb.SendEmailResponse{
		MessageId: fmt.Sprint(created.ID),
		Status:    string(created.Status),
		SentAt:    created.SentAt.Unix(),
	}, nil
}

func (s *Service) SendSMS(ctx context.Context, req *notificationpb.SendSMSRequest) (*notificationpb.SendSMSResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Metadata); err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	notification := &notificationrepo.Notification{
		UserID:   0, // Not mapped directly
		Type:     notificationrepo.NotificationTypeSMS,
		Title:    "SMS",
		Content:  req.Message,
		Status:   notificationrepo.NotificationStatusPending,
		Metadata: req.Metadata,
	}
	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		s.log.Error("Failed to send SMS notification", zap.Error(err))
		return nil, grpcstatus.Errorf(codes.Internal, "failed to send SMS: %v", err)
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
	return &notificationpb.SendSMSResponse{
		MessageId: fmt.Sprint(created.ID),
		Status:    string(created.Status),
		SentAt:    created.SentAt.Unix(),
	}, nil
}

func (s *Service) SendPushNotification(ctx context.Context, req *notificationpb.SendPushNotificationRequest) (*notificationpb.SendPushNotificationResponse, error) {
	if err := metadatautil.ValidateMetadata(req.Metadata); err != nil {
		return nil, grpcstatus.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	notification := &notificationrepo.Notification{
		UserID:   parseInt64(req.UserId),
		Type:     notificationrepo.NotificationTypePush,
		Title:    req.Title,
		Content:  req.Message,
		Status:   notificationrepo.NotificationStatusPending,
		Metadata: req.Metadata,
	}
	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		s.log.Error("Failed to send push notification", zap.Error(err))
		return nil, grpcstatus.Errorf(codes.Internal, "failed to send push notification: %v", err)
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
	broadcast := &notificationrepo.Notification{
		UserID:      0, // system/campaign
		Type:        notificationrepo.NotificationTypeInApp,
		Title:       req.Subject,
		Content:     req.Message,
		Status:      notificationrepo.NotificationStatusPending,
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
	notification.Status = notificationrepo.NotificationStatusSent // or a new 'read' status if supported
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

func mapNotificationToProto(n *notificationrepo.Notification) *notificationpb.Notification {
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

func mapStatusToProto(s notificationrepo.NotificationStatus) notificationpb.NotificationStatus {
	switch s {
	case notificationrepo.NotificationStatusPending:
		return notificationpb.NotificationStatus_NOTIFICATION_STATUS_PENDING
	case notificationrepo.NotificationStatusSent:
		return notificationpb.NotificationStatus_NOTIFICATION_STATUS_SENT
	case notificationrepo.NotificationStatusFailed:
		return notificationpb.NotificationStatus_NOTIFICATION_STATUS_FAILED
	case notificationrepo.NotificationStatusCancelled:
		return notificationpb.NotificationStatus_NOTIFICATION_STATUS_UNSPECIFIED
	default:
		return notificationpb.NotificationStatus_NOTIFICATION_STATUS_UNSPECIFIED
	}
}
