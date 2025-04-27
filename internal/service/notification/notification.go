package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	notificationpb "github.com/nmxmxh/master-ovasabi/api/protos/notification/v0"
	"github.com/nmxmxh/master-ovasabi/internal/shared/dbiface"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the NotificationService gRPC interface.
type Service struct {
	notificationpb.UnimplementedNotificationServiceServer
	log *zap.Logger
	db  dbiface.DB
}

// NewNotificationService creates a new instance of NotificationService.
func NewNotificationService(log *zap.Logger, db dbiface.DB) notificationpb.NotificationServiceServer {
	return &Service{
		log: log,
		db:  db,
	}
}

// SendEmail implements the SendEmail RPC method.
func (s *Service) SendEmail(ctx context.Context, req *notificationpb.SendEmailRequest) (*notificationpb.SendEmailResponse, error) {
	s.log.Info("Sending email",
		zap.String("to", req.To),
		zap.String("subject", req.Subject))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.log.Warn("failed to rollback tx", zap.Error(err))
		}
	}()

	// 1. Create master record for the notification
	var masterID int32
	err = tx.QueryRowContext(ctx,
		`INSERT INTO master (uuid, name, type) 
		 VALUES ($1, $2, 'notification') 
		 RETURNING id`,
		uuid.New().String(), "email_notification").Scan(&masterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create master record: %v", err)
	}

	// 2. Create service_notification record
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal metadata: %v", err)
	}

	var notificationID string
	var sentAt time.Time
	err = tx.QueryRowContext(ctx,
		`INSERT INTO service_notification 
		 (master_id, type, message, payload, is_read, sent_at, created_at) 
		 VALUES ($1, 'email', $2, $3, false, NOW(), NOW()) 
		 RETURNING id, sent_at`,
		masterID, req.Body, metadata).Scan(&notificationID, &sentAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create notification record: %v", err)
	}

	// 3. Log event
	_, err = tx.ExecContext(ctx,
		`INSERT INTO service_event 
		 (master_id, event_type, payload) 
		 VALUES ($1, 'email_sent', $2)`,
		masterID, metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to log event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return &notificationpb.SendEmailResponse{
		MessageId: notificationID,
		Status:    "sent",
		SentAt:    sentAt.Unix(),
	}, nil
}

// SendSMS implements the SendSMS RPC method.
func (s *Service) SendSMS(ctx context.Context, req *notificationpb.SendSMSRequest) (*notificationpb.SendSMSResponse, error) {
	s.log.Info("Sending SMS",
		zap.String("to", req.To),
		zap.Int("message_length", len(req.Message)))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.log.Warn("failed to rollback tx", zap.Error(err))
		}
	}()

	// 1. Create master record for SMS notification
	var masterID int32
	err = tx.QueryRowContext(ctx,
		`INSERT INTO master (uuid, name, type) 
		 VALUES ($1, $2, 'notification') 
		 RETURNING id`,
		uuid.New().String(), "sms_notification").Scan(&masterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create master record: %v", err)
	}

	// 2. Create service_notification record
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal metadata: %v", err)
	}

	var notificationID string
	var sentAt time.Time
	err = tx.QueryRowContext(ctx,
		`INSERT INTO service_notification 
		 (master_id, type, message, payload, is_read, sent_at, created_at) 
		 VALUES ($1, 'sms', $2, $3, false, NOW(), NOW()) 
		 RETURNING id, sent_at`,
		masterID, req.Message, metadata).Scan(&notificationID, &sentAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create notification record: %v", err)
	}

	// 3. Log event
	_, err = tx.ExecContext(ctx,
		`INSERT INTO service_event 
		 (master_id, event_type, payload) 
		 VALUES ($1, 'sms_sent', $2)`,
		masterID, metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to log event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return &notificationpb.SendSMSResponse{
		MessageId: notificationID,
		Status:    "sent",
		SentAt:    sentAt.Unix(),
	}, nil
}

// SendPushNotification implements the SendPushNotification RPC method.
func (s *Service) SendPushNotification(ctx context.Context, req *notificationpb.SendPushNotificationRequest) (*notificationpb.SendPushNotificationResponse, error) {
	s.log.Info("Sending push notification",
		zap.String("user_id", req.UserId),
		zap.String("title", req.Title),
		zap.String("message", req.Message))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.log.Warn("failed to rollback tx", zap.Error(err))
		}
	}()

	// 1. Create master record for push notification
	var masterID int32
	err = tx.QueryRowContext(ctx,
		`INSERT INTO master (uuid, name, type) 
		 VALUES ($1, $2, 'notification') 
		 RETURNING id`,
		uuid.New().String(), "push_notification").Scan(&masterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create master record: %v", err)
	}

	// 2. Create service_notification record
	metadata, err := json.Marshal(map[string]interface{}{
		"title":    req.Title,
		"metadata": req.Metadata,
		"user_id":  req.UserId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal metadata: %v", err)
	}

	var notificationID string
	var sentAt time.Time
	err = tx.QueryRowContext(ctx,
		`INSERT INTO service_notification 
		 (master_id, type, message, payload, is_read, sent_at, created_at) 
		 VALUES ($1, 'push', $2, $3, false, NOW(), NOW()) 
		 RETURNING id, sent_at`,
		masterID, req.Message, metadata).Scan(&notificationID, &sentAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create notification record: %v", err)
	}

	// 3. Log event
	_, err = tx.ExecContext(ctx,
		`INSERT INTO service_event 
		 (master_id, event_type, payload) 
		 VALUES ($1, 'push_notification_sent', $2)`,
		masterID, metadata)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to log event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return &notificationpb.SendPushNotificationResponse{
		NotificationId: notificationID,
		Status:         "sent",
		SentAt:         sentAt.Unix(),
	}, nil
}

// GetNotificationHistory implements the GetNotificationHistory RPC method.
func (s *Service) GetNotificationHistory(ctx context.Context, req *notificationpb.GetNotificationHistoryRequest) (*notificationpb.GetNotificationHistoryResponse, error) {
	query := `
		SELECT n.id, n.type, n.message, n.payload, n.is_read, n.sent_at, n.created_at,
		       COUNT(*) OVER() as total_count
		FROM service_notification n
		JOIN master m ON m.id = n.master_id
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if req.Type != "" {
		query += ` AND n.type = $` + strconv.Itoa(argPos)
		args = append(args, req.Type)
		argPos++
	}

	if req.StartDate > 0 {
		query += ` AND n.created_at >= to_timestamp($` + strconv.Itoa(argPos) + `)`
		args = append(args, req.StartDate)
		argPos++
	}

	if req.EndDate > 0 {
		query += ` AND n.created_at <= to_timestamp($` + strconv.Itoa(argPos) + `)`
		args = append(args, req.EndDate)
		argPos++
	}

	// Add pagination
	pageSize := int32(10)
	if req.PageSize > 0 {
		pageSize = req.PageSize
	}
	offset := req.Page * pageSize

	query += ` ORDER BY n.created_at DESC LIMIT $` + strconv.Itoa(argPos) + ` OFFSET $` + strconv.Itoa(argPos+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.log.Warn("failed to close rows", zap.Error(err))
		}
	}()

	var notifications []*notificationpb.NotificationHistory
	var totalCount int32

	for rows.Next() {
		var notification notificationpb.NotificationHistory
		var payload []byte
		err := rows.Scan(
			&notification.Id,
			&notification.Type,
			&notification.Content,
			&payload,
			&notification.CreatedAt,
			&totalCount,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan row: %v", err)
		}

		// Parse metadata from payload
		if err := json.Unmarshal(payload, &notification.Metadata); err != nil {
			s.log.Warn("failed to unmarshal notification payload",
				zap.String("notification_id", notification.Id),
				zap.Error(err))
		}

		notifications = append(notifications, &notification)
	}

	if err = rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "error iterating rows: %v", err)
	}

	totalPages := (totalCount + pageSize - 1) / pageSize
	return &notificationpb.GetNotificationHistoryResponse{
		Notifications: notifications,
		TotalCount:    totalCount,
		Page:          req.Page,
		TotalPages:    totalPages,
	}, nil
}

// UpdateNotificationPreferences implements the UpdateNotificationPreferences RPC method.
func (s *Service) UpdateNotificationPreferences(ctx context.Context, req *notificationpb.UpdateNotificationPreferencesRequest) (*notificationpb.UpdateNotificationPreferencesResponse, error) {
	if req.Preferences == nil {
		s.log.Error("Invalid preferences",
			zap.String("user_id", req.UserId),
			zap.Error(status.Error(codes.InvalidArgument, "preferences cannot be nil")))
		return nil, status.Error(codes.InvalidArgument, "preferences cannot be nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.log.Warn("failed to rollback tx", zap.Error(err))
		}
	}()

	// Convert preferences to JSON for storage
	preferencesJSON, err := json.Marshal(req.Preferences)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal preferences: %v", err)
	}

	// Update or insert notification preferences
	var updatedAt time.Time
	err = tx.QueryRowContext(ctx, `
		INSERT INTO service_notification (master_id, type, payload, created_at)
		VALUES (
			(SELECT id FROM master WHERE uuid = $1),
			'preferences',
			$2,
			NOW()
		)
		ON CONFLICT (master_id, type) DO UPDATE
		SET payload = $2,
		    updated_at = NOW()
		RETURNING updated_at`,
		req.UserId, preferencesJSON).Scan(&updatedAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update preferences: %v", err)
	}

	// Log event
	_, err = tx.ExecContext(ctx,
		`INSERT INTO service_event 
		 (master_id, event_type, payload) 
		 VALUES (
			(SELECT id FROM master WHERE uuid = $1),
			'notification_preferences_updated',
			$2
		 )`,
		req.UserId, preferencesJSON)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to log event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return &notificationpb.UpdateNotificationPreferencesResponse{
		Preferences: req.Preferences,
		UpdatedAt:   updatedAt.Unix(),
	}, nil
}
