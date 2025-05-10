package notification

import (
	context "context"

	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v1"
	notificationrepo "github.com/nmxmxh/master-ovasabi/internal/repository/notification"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (s *Service) CreateNotification(_ context.Context, _ *notificationpb.CreateNotificationRequest) (*notificationpb.CreateNotificationResponse, error) {
	// TODO: Implement CreateNotification
	return nil, status.Error(codes.Unimplemented, "CreateNotification not yet implemented")
}

func (s *Service) GetNotification(_ context.Context, _ *notificationpb.GetNotificationRequest) (*notificationpb.GetNotificationResponse, error) {
	// TODO: Implement GetNotification
	return nil, status.Error(codes.Unimplemented, "GetNotification not yet implemented")
}

func (s *Service) ListNotifications(_ context.Context, _ *notificationpb.ListNotificationsRequest) (*notificationpb.ListNotificationsResponse, error) {
	// TODO: Implement ListNotifications
	return nil, status.Error(codes.Unimplemented, "ListNotifications not yet implemented")
}

func (s *Service) SendEmail(_ context.Context, _ *notificationpb.SendEmailRequest) (*notificationpb.SendEmailResponse, error) {
	// TODO: Implement SendEmail
	return nil, status.Error(codes.Unimplemented, "SendEmail not yet implemented")
}

func (s *Service) SendSMS(_ context.Context, _ *notificationpb.SendSMSRequest) (*notificationpb.SendSMSResponse, error) {
	// TODO: Implement SendSMS
	return nil, status.Error(codes.Unimplemented, "SendSMS not yet implemented")
}

func (s *Service) SendPushNotification(_ context.Context, _ *notificationpb.SendPushNotificationRequest) (*notificationpb.SendPushNotificationResponse, error) {
	// TODO: Implement SendPushNotification
	return nil, status.Error(codes.Unimplemented, "SendPushNotification not yet implemented")
}

func (s *Service) GetNotificationHistory(_ context.Context, _ *notificationpb.GetNotificationHistoryRequest) (*notificationpb.GetNotificationHistoryResponse, error) {
	// TODO: Implement GetNotificationHistory
	return nil, status.Error(codes.Unimplemented, "GetNotificationHistory not yet implemented")
}

func (s *Service) UpdateNotificationPreferences(_ context.Context, _ *notificationpb.UpdateNotificationPreferencesRequest) (*notificationpb.UpdateNotificationPreferencesResponse, error) {
	// TODO: Implement UpdateNotificationPreferences
	return nil, status.Error(codes.Unimplemented, "UpdateNotificationPreferences not yet implemented")
}
