package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	notificationpb "github.com/ovasabi/master-ovasabi/api/protos/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestNewNotificationService(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewNotificationService(logger)
	assert.NotNil(t, service, "Service should not be nil")
}

func TestSendEmail(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewNotificationService(logger)

	tests := []struct {
		name    string
		req     *notificationpb.SendEmailRequest
		wantErr bool
	}{
		{
			name: "successful email send",
			req: &notificationpb.SendEmailRequest{
				To:      "test@example.com",
				Subject: "Test Subject",
				Body:    "Test Body",
				Metadata: map[string]string{
					"category": "test",
				},
				Html: true,
			},
			wantErr: false,
		},
		{
			name: "successful email send without metadata",
			req: &notificationpb.SendEmailRequest{
				To:      "test@example.com",
				Subject: "Test Subject",
				Body:    "Test Body",
				Html:    false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.SendEmail(context.Background(), tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, resp.MessageId)
			assert.Equal(t, "sent", resp.Status)
			assert.Greater(t, resp.SentAt, int64(0))
		})
	}
}

func TestSendSMS(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewNotificationService(logger)

	tests := []struct {
		name    string
		req     *notificationpb.SendSMSRequest
		wantErr bool
	}{
		{
			name: "successful SMS send",
			req: &notificationpb.SendSMSRequest{
				To:      "+1234567890",
				Message: "Test Message",
				Metadata: map[string]string{
					"category": "test",
				},
			},
			wantErr: false,
		},
		{
			name: "successful SMS send without metadata",
			req: &notificationpb.SendSMSRequest{
				To:      "+1234567890",
				Message: "Test Message",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.SendSMS(context.Background(), tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, resp.MessageId)
			assert.Equal(t, "sent", resp.Status)
			assert.Greater(t, resp.SentAt, int64(0))
		})
	}
}

func TestSendPushNotification(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewNotificationService(logger)

	tests := []struct {
		name    string
		req     *notificationpb.SendPushNotificationRequest
		wantErr bool
	}{
		{
			name: "successful push notification send",
			req: &notificationpb.SendPushNotificationRequest{
				UserId:  uuid.New().String(),
				Title:   "Test Title",
				Message: "Test Message",
				Metadata: map[string]string{
					"category": "test",
				},
				DeepLink: "app://test",
			},
			wantErr: false,
		},
		{
			name: "successful push notification send without metadata",
			req: &notificationpb.SendPushNotificationRequest{
				UserId:  uuid.New().String(),
				Title:   "Test Title",
				Message: "Test Message",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.SendPushNotification(context.Background(), tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, resp.NotificationId)
			assert.Equal(t, "sent", resp.Status)
			assert.Greater(t, resp.SentAt, int64(0))
		})
	}
}

func TestGetNotificationHistory(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewNotificationService(logger)
	userId := uuid.New().String()

	// Send some notifications first
	emailReq := &notificationpb.SendEmailRequest{
		To:      "test@example.com",
		Subject: "Test Subject",
		Body:    "Test Body",
	}
	_, err := service.SendEmail(context.Background(), emailReq)
	require.NoError(t, err)

	smsReq := &notificationpb.SendSMSRequest{
		To:      "+1234567890",
		Message: "Test Message",
	}
	_, err = service.SendSMS(context.Background(), smsReq)
	require.NoError(t, err)

	pushReq := &notificationpb.SendPushNotificationRequest{
		UserId:  userId,
		Title:   "Test Title",
		Message: "Test Message",
	}
	_, err = service.SendPushNotification(context.Background(), pushReq)
	require.NoError(t, err)

	tests := []struct {
		name          string
		req           *notificationpb.GetNotificationHistoryRequest
		wantErr       bool
		expectedCount int32
		expectedPages int32
		filterByType  bool
		filterByDates bool
	}{
		{
			name: "get all notifications",
			req: &notificationpb.GetNotificationHistoryRequest{
				UserId:   userId,
				Page:     0,
				PageSize: 10,
			},
			wantErr:       false,
			expectedCount: 1, // Only push notification is associated with userId
		},
		{
			name: "get notifications with type filter",
			req: &notificationpb.GetNotificationHistoryRequest{
				UserId:   userId,
				Page:     0,
				PageSize: 10,
				Type:     "push",
			},
			wantErr:       false,
			expectedCount: 1,
			filterByType:  true,
		},
		{
			name: "get notifications with date filter",
			req: &notificationpb.GetNotificationHistoryRequest{
				UserId:    userId,
				Page:      0,
				PageSize:  10,
				StartDate: time.Now().Add(-1 * time.Hour).Unix(),
				EndDate:   time.Now().Add(1 * time.Hour).Unix(),
			},
			wantErr:       false,
			expectedCount: 1,
			filterByDates: true,
		},
		{
			name: "get notifications for non-existent user",
			req: &notificationpb.GetNotificationHistoryRequest{
				UserId:   uuid.New().String(),
				Page:     0,
				PageSize: 10,
			},
			wantErr:       false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.GetNotificationHistory(context.Background(), tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCount, resp.TotalCount)
			assert.Equal(t, tt.req.Page, resp.Page)

			if tt.filterByType {
				for _, notification := range resp.Notifications {
					assert.Equal(t, tt.req.Type, notification.Type)
				}
			}

			if tt.filterByDates {
				for _, notification := range resp.Notifications {
					assert.GreaterOrEqual(t, notification.CreatedAt, tt.req.StartDate)
					assert.LessOrEqual(t, notification.CreatedAt, tt.req.EndDate)
				}
			}
		})
	}
}

func TestUpdateNotificationPreferences(t *testing.T) {
	logger := zaptest.NewLogger(t)
	service := NewNotificationService(logger)
	userId := uuid.New().String()

	tests := []struct {
		name    string
		req     *notificationpb.UpdateNotificationPreferencesRequest
		wantErr bool
	}{
		{
			name: "successful preferences update",
			req: &notificationpb.UpdateNotificationPreferencesRequest{
				UserId: userId,
				Preferences: &notificationpb.NotificationPreferences{
					EmailEnabled: true,
					SmsEnabled:   true,
					PushEnabled:  true,
					NotificationTypes: map[string]bool{
						"marketing": true,
						"system":    true,
					},
					QuietHours: []string{"22:00-06:00"},
					Timezone:   "UTC",
				},
			},
			wantErr: false,
		},
		{
			name: "nil preferences",
			req: &notificationpb.UpdateNotificationPreferencesRequest{
				UserId:      userId,
				Preferences: nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.UpdateNotificationPreferences(context.Background(), tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, resp.Preferences)
			assert.Equal(t, tt.req.Preferences.EmailEnabled, resp.Preferences.EmailEnabled)
			assert.Equal(t, tt.req.Preferences.SmsEnabled, resp.Preferences.SmsEnabled)
			assert.Equal(t, tt.req.Preferences.PushEnabled, resp.Preferences.PushEnabled)
			assert.Equal(t, tt.req.Preferences.NotificationTypes, resp.Preferences.NotificationTypes)
			assert.Equal(t, tt.req.Preferences.QuietHours, resp.Preferences.QuietHours)
			assert.Equal(t, tt.req.Preferences.Timezone, resp.Preferences.Timezone)
			assert.Greater(t, resp.UpdatedAt, int64(0))
		})
	}
}
