package notification

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the NotificationService gRPC interface
type Service struct {
	notificationpb.UnimplementedNotificationServiceServer
	log         *zap.Logger
	mu          sync.RWMutex
	history     map[string][]*notificationpb.NotificationHistory
	preferences map[string]*notificationpb.NotificationPreferences
}

// NewNotificationService creates a new instance of NotificationService
func NewNotificationService(log *zap.Logger) notificationpb.NotificationServiceServer {
	return &Service{
		log:         log,
		history:     make(map[string][]*notificationpb.NotificationHistory),
		preferences: make(map[string]*notificationpb.NotificationPreferences),
	}
}

// SendEmail implements the SendEmail RPC method
func (s *Service) SendEmail(ctx context.Context, req *notificationpb.SendEmailRequest) (*notificationpb.SendEmailResponse, error) {
	// In a real implementation, you would:
	// 1. Validate email format
	// 2. Check user preferences
	// 3. Use an email service provider
	// 4. Handle rate limiting
	// For this example, we'll just record the notification

	notification := &notificationpb.NotificationHistory{
		Id:        uuid.New().String(),
		Type:      "email",
		Status:    "sent",
		Content:   req.Body,
		Metadata:  req.Metadata,
		CreatedAt: time.Now().Unix(),
	}

	s.mu.Lock()
	if _, ok := s.history[req.To]; !ok {
		s.history[req.To] = make([]*notificationpb.NotificationHistory, 0)
	}
	s.history[req.To] = append(s.history[req.To], notification)
	s.mu.Unlock()

	return &notificationpb.SendEmailResponse{
		MessageId: notification.Id,
		Status:    "sent",
		SentAt:    notification.CreatedAt,
	}, nil
}

// SendSMS implements the SendSMS RPC method
func (s *Service) SendSMS(ctx context.Context, req *notificationpb.SendSMSRequest) (*notificationpb.SendSMSResponse, error) {
	// In a real implementation, you would:
	// 1. Validate phone number format
	// 2. Check user preferences
	// 3. Use an SMS service provider
	// 4. Handle rate limiting
	// For this example, we'll just record the notification

	notification := &notificationpb.NotificationHistory{
		Id:        uuid.New().String(),
		Type:      "sms",
		Status:    "sent",
		Content:   req.Message,
		Metadata:  req.Metadata,
		CreatedAt: time.Now().Unix(),
	}

	s.mu.Lock()
	if _, ok := s.history[req.To]; !ok {
		s.history[req.To] = make([]*notificationpb.NotificationHistory, 0)
	}
	s.history[req.To] = append(s.history[req.To], notification)
	s.mu.Unlock()

	return &notificationpb.SendSMSResponse{
		MessageId: notification.Id,
		Status:    "sent",
		SentAt:    notification.CreatedAt,
	}, nil
}

// SendPushNotification implements the SendPushNotification RPC method
func (s *Service) SendPushNotification(ctx context.Context, req *notificationpb.SendPushNotificationRequest) (*notificationpb.SendPushNotificationResponse, error) {
	// In a real implementation, you would:
	// 1. Validate user's device tokens
	// 2. Check user preferences
	// 3. Use a push notification service (FCM, APNS, etc.)
	// 4. Handle rate limiting
	// For this example, we'll just record the notification

	notification := &notificationpb.NotificationHistory{
		Id:        uuid.New().String(),
		Type:      "push",
		Status:    "sent",
		Content:   req.Message,
		Metadata:  req.Metadata,
		CreatedAt: time.Now().Unix(),
	}

	s.mu.Lock()
	if _, ok := s.history[req.UserId]; !ok {
		s.history[req.UserId] = make([]*notificationpb.NotificationHistory, 0)
	}
	s.history[req.UserId] = append(s.history[req.UserId], notification)
	s.mu.Unlock()

	return &notificationpb.SendPushNotificationResponse{
		NotificationId: notification.Id,
		Status:         "sent",
		SentAt:         notification.CreatedAt,
	}, nil
}

// GetNotificationHistory implements the GetNotificationHistory RPC method
func (s *Service) GetNotificationHistory(ctx context.Context, req *notificationpb.GetNotificationHistoryRequest) (*notificationpb.GetNotificationHistoryResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, ok := s.history[req.UserId]
	if !ok {
		return &notificationpb.GetNotificationHistoryResponse{
			Notifications: []*notificationpb.NotificationHistory{},
			TotalCount:    0,
			Page:          req.Page,
			TotalPages:    0,
		}, nil
	}

	// Filter by type if specified
	var filtered []*notificationpb.NotificationHistory
	for _, notification := range history {
		if req.Type != "" && notification.Type != req.Type {
			continue
		}
		if req.StartDate > 0 && notification.CreatedAt < req.StartDate {
			continue
		}
		if req.EndDate > 0 && notification.CreatedAt > req.EndDate {
			continue
		}
		filtered = append(filtered, notification)
	}

	// Calculate pagination
	totalCount := len(filtered)
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	totalPages := (totalCount + pageSize - 1) / pageSize

	start := int(req.Page) * pageSize
	if start >= totalCount {
		return &notificationpb.GetNotificationHistoryResponse{
			Notifications: []*notificationpb.NotificationHistory{},
			TotalCount:    int32(totalCount),
			Page:          req.Page,
			TotalPages:    int32(totalPages),
		}, nil
	}

	end := start + pageSize
	if end > totalCount {
		end = totalCount
	}

	return &notificationpb.GetNotificationHistoryResponse{
		Notifications: filtered[start:end],
		TotalCount:    int32(totalCount),
		Page:          req.Page,
		TotalPages:    int32(totalPages),
	}, nil
}

// UpdateNotificationPreferences implements the UpdateNotificationPreferences RPC method
func (s *Service) UpdateNotificationPreferences(ctx context.Context, req *notificationpb.UpdateNotificationPreferencesRequest) (*notificationpb.UpdateNotificationPreferencesResponse, error) {
	if req.Preferences == nil {
		return nil, status.Error(codes.InvalidArgument, "preferences cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.preferences[req.UserId] = req.Preferences

	return &notificationpb.UpdateNotificationPreferencesResponse{
		Preferences: req.Preferences,
		UpdatedAt:   time.Now().Unix(),
	}, nil
}
