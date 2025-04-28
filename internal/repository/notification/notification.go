package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/nmxmxh/master-ovasabi/internal/repository"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrNotificationExists   = errors.New("notification already exists")
)

var log *zap.Logger

func SetLogger(l *zap.Logger) {
	log = l
}

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypePush  NotificationType = "push"
	NotificationTypeInApp NotificationType = "in_app"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusCancelled NotificationStatus = "cancelled"
)

// Notification represents a notification entry in the service_notification table
type Notification struct {
	ID          int64              `db:"id"`
	MasterID    int64              `db:"master_id"`
	UserID      int64              `db:"user_id"`
	Type        NotificationType   `db:"type"`
	Title       string             `db:"title"`
	Content     string             `db:"content"`
	Status      NotificationStatus `db:"status"`
	Metadata    string             `db:"metadata"`
	ScheduledAt *time.Time         `db:"scheduled_at"`
	SentAt      *time.Time         `db:"sent_at"`
	CreatedAt   time.Time          `db:"created_at"`
	UpdatedAt   time.Time          `db:"updated_at"`
}

// NotificationRepository handles operations on the service_notification table
type NotificationRepository struct {
	*repository.BaseRepository
	masterRepo repository.MasterRepository
}

// NewNotificationRepository creates a new notification repository instance
func NewNotificationRepository(db *sql.DB, masterRepo repository.MasterRepository) *NotificationRepository {
	return &NotificationRepository{
		BaseRepository: repository.NewBaseRepository(db),
		masterRepo:     masterRepo,
	}
}

// Create inserts a new notification record
func (r *NotificationRepository) Create(ctx context.Context, notification *Notification) (*Notification, error) {
	masterID, err := r.masterRepo.Create(ctx, repository.EntityTypeNotification)
	if err != nil {
		return nil, err
	}

	notification.MasterID = masterID
	err = r.GetDB().QueryRowContext(ctx,
		`INSERT INTO service_notification (
			master_id, user_id, type, title, content,
			status, metadata, scheduled_at, sent_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			NOW(), NOW()
		) RETURNING id, created_at, updated_at`,
		notification.MasterID, notification.UserID,
		notification.Type, notification.Title,
		notification.Content, notification.Status,
		notification.Metadata, notification.ScheduledAt,
		notification.SentAt,
	).Scan(&notification.ID, &notification.CreatedAt, &notification.UpdatedAt)

	if err != nil {
		_ = r.masterRepo.Delete(ctx, masterID)
		return nil, err
	}

	return notification, nil
}

// GetByID retrieves a notification by ID
func (r *NotificationRepository) GetByID(ctx context.Context, id int64) (*Notification, error) {
	notification := &Notification{}
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, user_id, type, title,
			content, status, metadata, scheduled_at,
			sent_at, created_at, updated_at
		FROM service_notification 
		WHERE id = $1`,
		id,
	).Scan(
		&notification.ID, &notification.MasterID,
		&notification.UserID, &notification.Type,
		&notification.Title, &notification.Content,
		&notification.Status, &notification.Metadata,
		&notification.ScheduledAt, &notification.SentAt,
		&notification.CreatedAt, &notification.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}
	return notification, nil
}

// Update updates a notification record
func (r *NotificationRepository) Update(ctx context.Context, notification *Notification) error {
	result, err := r.GetDB().ExecContext(ctx,
		`UPDATE service_notification 
		SET type = $1, title = $2, content = $3,
			status = $4, metadata = $5, scheduled_at = $6,
			sent_at = $7, updated_at = NOW()
		WHERE id = $8`,
		notification.Type, notification.Title,
		notification.Content, notification.Status,
		notification.Metadata, notification.ScheduledAt,
		notification.SentAt, notification.ID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrNotificationNotFound
	}

	return nil
}

// Delete removes a notification and its master record
func (r *NotificationRepository) Delete(ctx context.Context, id int64) error {
	notification, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return r.masterRepo.Delete(ctx, notification.MasterID)
}

// List retrieves a paginated list of notifications
func (r *NotificationRepository) List(ctx context.Context, limit, offset int) ([]*Notification, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, user_id, type, title,
			content, status, metadata, scheduled_at,
			sent_at, created_at, updated_at
		FROM service_notification 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			if log != nil {
				log.Error("error closing rows", zap.Error(cerr))
			}
		}
	}()

	var notifications []*Notification
	for rows.Next() {
		notification := &Notification{}
		err := rows.Scan(
			&notification.ID, &notification.MasterID,
			&notification.UserID, &notification.Type,
			&notification.Title, &notification.Content,
			&notification.Status, &notification.Metadata,
			&notification.ScheduledAt, &notification.SentAt,
			&notification.CreatedAt, &notification.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}

// ListByUserID retrieves all notifications for a specific user
func (r *NotificationRepository) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]*Notification, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, user_id, type, title,
			content, status, metadata, scheduled_at,
			sent_at, created_at, updated_at
		FROM service_notification 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			if log != nil {
				log.Error("error closing rows", zap.Error(cerr))
			}
		}
	}()

	var notifications []*Notification
	for rows.Next() {
		notification := &Notification{}
		err := rows.Scan(
			&notification.ID, &notification.MasterID,
			&notification.UserID, &notification.Type,
			&notification.Title, &notification.Content,
			&notification.Status, &notification.Metadata,
			&notification.ScheduledAt, &notification.SentAt,
			&notification.CreatedAt, &notification.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}

// ListPendingScheduled retrieves all pending notifications that are scheduled to be sent
func (r *NotificationRepository) ListPendingScheduled(ctx context.Context) ([]*Notification, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, user_id, type, title,
			content, status, metadata, scheduled_at,
			sent_at, created_at, updated_at
		FROM service_notification 
		WHERE status = $1 
		AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC`,
		NotificationStatusPending,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			if log != nil {
				log.Error("error closing rows", zap.Error(cerr))
			}
		}
	}()

	var notifications []*Notification
	for rows.Next() {
		notification := &Notification{}
		err := rows.Scan(
			&notification.ID, &notification.MasterID,
			&notification.UserID, &notification.Type,
			&notification.Title, &notification.Content,
			&notification.Status, &notification.Metadata,
			&notification.ScheduledAt, &notification.SentAt,
			&notification.CreatedAt, &notification.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}
