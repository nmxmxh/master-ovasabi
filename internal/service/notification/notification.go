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
	"github.com/nmxmxh/master-ovasabi/pkg/contextx"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
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
	eventEmitter events.EventEmitter
	eventEnabled bool
}

func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool) notificationpb.NotificationServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

func (s *Service) GetNotification(ctx context.Context, req *notificationpb.GetNotificationRequest) (*notificationpb.GetNotificationResponse, error) {
	id := s.parseInt64(req.NotificationId)
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.NotFound, "notification not found", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	return &notificationpb.GetNotificationResponse{
		Notification: s.mapNotificationToProto(notification),
	}, nil
}

func (s *Service) ListNotifications(ctx context.Context, req *notificationpb.ListNotificationsRequest) (*notificationpb.ListNotificationsResponse, error) {
	userID := s.parseInt64(req.UserId)
	notifications, err := s.repo.ListByUserID(ctx, userID, int(req.PageSize), int(req.Page))
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to list notifications", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	protoNotifications := make([]*notificationpb.Notification, 0, len(notifications))
	for _, n := range notifications {
		protoNotifications = append(protoNotifications, s.mapNotificationToProto(n))
	}
	var totalCount int32
	if len(protoNotifications) > math.MaxInt32 {
		totalCount = math.MaxInt32
	} else {
		totalCount = int32(math.Min(float64(len(protoNotifications)), float64(math.MaxInt32)))
	}
	// Calculate real total pages
	totalPages := int32(1)
	if req.PageSize > 0 {
		totalPages = int32(math.Ceil(float64(totalCount) / float64(req.PageSize)))
	}
	return &notificationpb.ListNotificationsResponse{
		Notifications: protoNotifications,
		TotalCount:    totalCount,
		Page:          req.Page,
		TotalPages:    totalPages,
	}, nil
}

func (s *Service) SendNotification(ctx context.Context, req *notificationpb.SendNotificationRequest) (*notificationpb.SendNotificationResponse, error) {
	userID := s.extractAuthContext(ctx, req.Metadata)
	isGuest := userID == ""
	if !isGuest && userID == "" {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Unauthenticated, "unauthenticated: user_id required", nil))
	}
	if isGuest {
		if req.UserId != "" && req.UserId != "guest" {
			return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.PermissionDenied, "guests cannot send direct user notifications", nil))
		}
	}
	notification := &Notification{
		UserID:     s.parseInt64(req.UserId),
		CampaignID: req.CampaignId,
		Type:       Type(req.Channel),
		Title:      req.Title,
		Content:    req.Body,
		Status:     StatusPending,
		Metadata:   req.Metadata,
	}
	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to send notification", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "notification sent", created, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.cache,
		CacheKey:     fmt.Sprint(created.ID),
		CacheValue:   created,
		CacheTTL:     10 * time.Minute,
		Metadata:     created.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "notification.sent",
		EventID:      fmt.Sprint(created.ID),
		PatternType:  "notification",
		PatternID:    fmt.Sprint(created.ID),
		PatternMeta:  created.Metadata,
	})
	return &notificationpb.SendNotificationResponse{
		Notification: s.mapNotificationToProto(created),
		Status:       "created",
	}, nil
}

func (s *Service) SendEmail(ctx context.Context, req *notificationpb.SendEmailRequest) (*notificationpb.SendEmailResponse, error) {
	userID := s.extractAuthContext(ctx, req.Metadata)
	if userID == "" {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Unauthenticated, "unauthenticated: user_id required for email", nil))
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
		err = graceful.WrapErr(ctx, codes.Internal, "failed to save email notification", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "email notification sent", created, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.cache,
		CacheKey:     fmt.Sprint(created.ID),
		CacheValue:   created,
		CacheTTL:     10 * time.Minute,
		Metadata:     created.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "notification.email_sent",
		EventID:      fmt.Sprint(created.ID),
		PatternType:  "notification",
		PatternID:    fmt.Sprint(created.ID),
		PatternMeta:  created.Metadata,
	})
	return &notificationpb.SendEmailResponse{
		MessageId: fmt.Sprint(created.ID),
		Status:    string(created.Status),
		SentAt:    created.SentAt.Unix(),
	}, nil
}

func (s *Service) SendSMS(ctx context.Context, req *notificationpb.SendSMSRequest) (*notificationpb.SendSMSResponse, error) {
	userID := s.extractAuthContext(ctx, req.Metadata)
	if userID == "" {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Unauthenticated, "unauthenticated: user_id required for SMS", nil))
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
		err = graceful.WrapErr(ctx, codes.Internal, "failed to save SMS notification", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "sms notification sent", created, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.cache,
		CacheKey:     fmt.Sprint(created.ID),
		CacheValue:   created,
		CacheTTL:     10 * time.Minute,
		Metadata:     created.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "notification.sms_sent",
		EventID:      fmt.Sprint(created.ID),
		PatternType:  "notification",
		PatternID:    fmt.Sprint(created.ID),
		PatternMeta:  created.Metadata,
	})
	return &notificationpb.SendSMSResponse{
		MessageId: fmt.Sprint(created.ID),
		Status:    string(created.Status),
		SentAt:    created.SentAt.Unix(),
	}, nil
}

func (s *Service) SendPushNotification(ctx context.Context, req *notificationpb.SendPushNotificationRequest) (*notificationpb.SendPushNotificationResponse, error) {
	userID := s.extractAuthContext(ctx, req.Metadata)
	if userID == "" {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Unauthenticated, "unauthenticated: user_id required for push notification", nil))
	}
	notification := &Notification{
		UserID:     s.parseInt64(req.UserId),
		CampaignID: req.CampaignId,
		Type:       TypePush,
		Title:      req.Title,
		Content:    req.Message,
		Status:     StatusPending,
		Metadata:   req.Metadata,
	}
	created, err := s.repo.Create(ctx, notification)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to save push notification", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "push notification sent", created, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.cache,
		CacheKey:     fmt.Sprint(created.ID),
		CacheValue:   created,
		CacheTTL:     10 * time.Minute,
		Metadata:     created.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "notification.push_sent",
		EventID:      fmt.Sprint(created.ID),
		PatternType:  "notification",
		PatternID:    fmt.Sprint(created.ID),
		PatternMeta:  created.Metadata,
	})
	return &notificationpb.SendPushNotificationResponse{
		NotificationId: fmt.Sprint(created.ID),
		Status:         string(created.Status),
		SentAt:         created.SentAt.Unix(),
	}, nil
}

func (s *Service) BroadcastEvent(ctx context.Context, req *notificationpb.BroadcastEventRequest) (*notificationpb.BroadcastEventResponse, error) {
	userID := s.extractAuthContext(ctx, req.Payload)
	isGuest := userID == ""
	if !isGuest && userID == "" {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.Unauthenticated, "unauthenticated: user_id or guest_nickname/device_id required", nil))
	}
	if isGuest && req.CampaignId == 0 {
		return nil, graceful.ToStatusError(graceful.WrapErr(ctx, codes.PermissionDenied, "guests can only broadcast to campaigns", nil))
	}
	broadcast := &Notification{
		UserID:      0,              // system/campaign
		CampaignID:  req.CampaignId, // now campaign-scoped
		Type:        TypeInApp,
		Title:       req.Subject,
		Content:     req.Message,
		Status:      StatusPending,
		Metadata:    req.Payload,
		ScheduledAt: s.toTimePtr(req.ScheduledAt),
	}
	created, err := s.repo.CreateBroadcast(ctx, broadcast)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to create broadcast", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "broadcast sent", created, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.cache,
		CacheKey:     fmt.Sprint(created.ID),
		CacheValue:   created,
		CacheTTL:     10 * time.Minute,
		Metadata:     created.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "notification.broadcast_sent",
		EventID:      fmt.Sprint(created.ID),
		PatternType:  "notification",
		PatternID:    fmt.Sprint(created.ID),
		PatternMeta:  created.Metadata,
	})
	return &notificationpb.BroadcastEventResponse{
		BroadcastId: fmt.Sprint(created.ID),
		Status:      "created",
		CampaignId:  created.CampaignID,
	}, nil
}

func (s *Service) ListNotificationEvents(ctx context.Context, req *notificationpb.ListNotificationEventsRequest) (*notificationpb.ListNotificationEventsResponse, error) {
	notificationID := s.parseInt64(req.NotificationId)
	eventList, err := s.repo.ListNotificationEvents(ctx, notificationID, int(req.PageSize), int(req.Page))
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to list events", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	protoEvents := make([]*notificationpb.NotificationEvent, 0, len(eventList))
	for _, e := range eventList {
		protoEvents = append(protoEvents, &notificationpb.NotificationEvent{
			EventId:        e.EventID,
			NotificationId: e.NotificationID,
			UserId:         e.UserID,
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
	id := s.parseInt64(req.NotificationId)
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.NotFound, "notification not found", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	notification.Status = StatusSent // or a new 'read' status if supported
	if err := s.repo.Update(ctx, notification); err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to acknowledge notification", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "notification acknowledged", notification, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.cache,
		CacheKey:     fmt.Sprint(notification.ID),
		CacheValue:   notification,
		CacheTTL:     10 * time.Minute,
		Metadata:     notification.Metadata,
		EventEmitter: s.eventEmitter,
		EventEnabled: s.eventEnabled,
		EventType:    "notification.acknowledged",
		EventID:      fmt.Sprint(notification.ID),
		PatternType:  "notification",
		PatternID:    fmt.Sprint(notification.ID),
		PatternMeta:  notification.Metadata,
	})
	return &notificationpb.AcknowledgeNotificationResponse{Status: "acknowledged"}, nil
}

func (s *Service) UpdateNotificationPreferences(ctx context.Context, _ *notificationpb.UpdateNotificationPreferencesRequest) (*notificationpb.UpdateNotificationPreferencesResponse, error) {
	err := graceful.WrapErr(ctx, codes.Unimplemented, "notification preferences update not implemented", errors.New("not implemented"))
	var ce *graceful.ContextError
	if errors.As(err, &ce) {
		ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
	}
	return nil, graceful.ToStatusError(err)
}

func (s *Service) SubscribeToEvents(req *notificationpb.SubscribeToEventsRequest, stream notificationpb.NotificationService_SubscribeToEventsServer) error {
	ctx := stream.Context()
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
		errWrapped := graceful.WrapErr(ctx, codes.Internal, "Failed to send event in stream", err)
		var ce *graceful.ContextError
		if errors.As(errWrapped, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return graceful.ToStatusError(errWrapped)
	}
	return nil // End of stream
}

func (s *Service) StreamAssetChunks(req *notificationpb.StreamAssetChunksRequest, stream notificationpb.NotificationService_StreamAssetChunksServer) error {
	ctx := stream.Context()
	// Example: send a single dummy chunk, then return
	chunk := &notificationpb.AssetChunk{
		UploadId: req.AssetId,
		Data:     []byte("example data"),
		Sequence: 1,
	}
	if err := stream.Send(chunk); err != nil {
		errWrapped := graceful.WrapErr(ctx, codes.Internal, "Failed to send asset chunk in stream", err)
		var ce *graceful.ContextError
		if errors.As(errWrapped, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return graceful.ToStatusError(errWrapped)
	}
	return nil // End of stream
}

// --- Helpers ---.
func (s *Service) parseInt64(str string) int64 {
	id, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		s.log.Warn("Failed to parse int64", zap.String("input", str), zap.Error(err))
		return 0
	}
	return id
}

func (s *Service) toTimePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

func (s *Service) mapNotificationToProto(n *Notification) *notificationpb.Notification {
	if n == nil {
		return nil
	}
	read := n.Status == StatusSent

	return &notificationpb.Notification{
		Id:        fmt.Sprint(n.ID),
		UserId:    fmt.Sprint(n.UserID),
		Channel:   string(n.Type),
		Title:     n.Title,
		Body:      n.Content,
		Status:    s.mapStatusToProto(n.Status),
		CreatedAt: timestamppb.New(n.CreatedAt),
		UpdatedAt: timestamppb.New(n.UpdatedAt),
		Read:      read, // Now mapped from status
	}
}

func (s *Service) mapStatusToProto(status Status) notificationpb.NotificationStatus {
	switch status {
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

// extractAuthContext extracts user_id from context or metadata.
func (s *Service) extractAuthContext(ctx context.Context, meta *commonpb.Metadata) (userID string) {
	// Try contextx.Auth first
	authCtx := contextx.Auth(ctx)
	if authCtx != nil && authCtx.UserID != "" {
		userID = authCtx.UserID
	}
	// Fallback: try metadata
	if userID == "" && meta != nil && meta.ServiceSpecific != nil {
		m := meta.ServiceSpecific.AsMap()
		if a, ok := m["actor"].(map[string]interface{}); ok {
			if v, ok := a["user_id"].(string); ok && userID == "" {
				userID = v
			}
		}
	}
	return userID
}
