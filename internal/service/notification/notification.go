package notification

import (
	"context"

	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	notificationrepo "github.com/nmxmxh/master-ovasabi/internal/repository/notification"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	EmailType NotificationType = "email"
	SMSType   NotificationType = "sms"
	PushType  NotificationType = "push"
)

// QueuedNotification represents a notification in the queue
type QueuedNotification struct {
	Type     NotificationType  `json:"type"`
	To       string            `json:"to"`
	Subject  string            `json:"subject,omitempty"`
	Message  string            `json:"message"`
	Title    string            `json:"title,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	UserID   string            `json:"user_id,omitempty"`
}

// Service implements the NotificationService gRPC interface.
type Service struct {
	notificationpb.UnimplementedNotificationServiceServer
	log   *zap.Logger
	cache *redis.Cache
	repo  *notificationrepo.NotificationRepository
}

// NewNotificationService creates a new instance of NotificationService.
func NewNotificationService(log *zap.Logger, repo *notificationrepo.NotificationRepository, cache *redis.Cache) notificationpb.NotificationServiceServer {
	return &Service{
		log:   log,
		repo:  repo,
		cache: cache,
	}
}

// SendEmail implements the SendEmail RPC method.
func (s *Service) SendEmail(ctx context.Context, req *notificationpb.SendEmailRequest) (*notificationpb.SendEmailResponse, error) {
	// TODO: Implement CreateMasterRecord, CreateServiceNotification, LogEvent in NotificationRepository
	return nil, status.Error(codes.Unimplemented, "SendEmail repository integration not yet implemented")
}

// SendSMS implements the SendSMS RPC method.
func (s *Service) SendSMS(ctx context.Context, req *notificationpb.SendSMSRequest) (*notificationpb.SendSMSResponse, error) {
	// TODO: Implement CreateMasterRecord, CreateServiceNotification, LogEvent in NotificationRepository
	return nil, status.Error(codes.Unimplemented, "SendSMS repository integration not yet implemented")
}

// SendPushNotification implements the SendPushNotification RPC method.
func (s *Service) SendPushNotification(ctx context.Context, req *notificationpb.SendPushNotificationRequest) (*notificationpb.SendPushNotificationResponse, error) {
	// TODO: Implement CreateMasterRecord, CreateServiceNotification, LogEvent in NotificationRepository
	return nil, status.Error(codes.Unimplemented, "SendPushNotification repository integration not yet implemented")
}

// GetNotificationHistory implements the GetNotificationHistory RPC method.
func (s *Service) GetNotificationHistory(ctx context.Context, req *notificationpb.GetNotificationHistoryRequest) (*notificationpb.GetNotificationHistoryResponse, error) {
	// TODO: Implement QueryNotificationHistory in NotificationRepository
	return nil, status.Error(codes.Unimplemented, "GetNotificationHistory repository integration not yet implemented")
}

// UpdateNotificationPreferences implements the UpdateNotificationPreferences RPC method.
func (s *Service) UpdateNotificationPreferences(ctx context.Context, req *notificationpb.UpdateNotificationPreferencesRequest) (*notificationpb.UpdateNotificationPreferencesResponse, error) {
	// TODO: Implement UpdateNotificationPreferences and LogEvent in NotificationRepository
	return nil, status.Error(codes.Unimplemented, "UpdateNotificationPreferences repository integration not yet implemented")
}
