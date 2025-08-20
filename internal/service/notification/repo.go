package notification

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrNotificationExists   = errors.New("notification already exists")
)

// NotificationType represents the type of notification.
type Type string

const (
	TypeEmail Type = "email"
	TypeSMS   Type = "sms"
	TypePush  Type = "push"
	TypeInApp Type = "in_app"
)

// NotificationStatus represents the status of a notification.
type Status string

const (
	StatusPending   Status = "pending"
	StatusSent      Status = "sent"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// Notification represents a notification entry in the service_notification table.
type Notification struct {
	ID          int64  `db:"id"`
	MasterID    int64  `db:"master_id"`
	MasterUUID  string `db:"master_uuid"`
	UserID      int64  `db:"user_id"`
	CampaignID  int64  `db:"campaign_id"`
	Type        Type   `db:"type"`
	Title       string `db:"title"`
	Content     string `db:"content"`
	Status      Status `db:"status"`
	Metadata    *commonpb.Metadata
	ScheduledAt *time.Time `db:"scheduled_at"`
	SentAt      *time.Time `db:"sent_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// NotificationRepository handles operations on the service_notification table.
type Repository struct {
	*repository.BaseRepository
	masterRepo repository.MasterRepository
}

// NewRepository creates a new notification repository instance.
func NewRepository(db *sql.DB, masterRepo repository.MasterRepository, log *zap.Logger) *Repository {
	return &Repository{
		BaseRepository: repository.NewBaseRepository(db, log),
		masterRepo:     masterRepo,
	}
}

// Create inserts a new notification record.
func (r *Repository) Create(ctx context.Context, notification *Notification) (*Notification, error) {
	// Generate a descriptive name for the master record
	masterName := r.GenerateMasterName(repository.EntityTypeNotification,
		notification.Title,
		string(notification.Type),
		fmt.Sprintf("user-%d", notification.UserID))

	tx, err := r.GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				r.GetLogger().Error("failed to rollback transaction on panic", zap.Error(rbErr))
			}
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				r.GetLogger().Error("failed to rollback transaction on panic", zap.Error(rbErr))
			}
			panic(p) // Re-throw panic
		}
	}()

	masterID, masterUUID, err := r.masterRepo.Create(ctx, tx, repository.EntityTypeNotification, masterName)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			r.GetLogger().Error("failed to rollback transaction", zap.Error(rbErr))
		}
		return nil, fmt.Errorf("failed to create master entity for notification: %w", err)
	}

	notification.MasterID = masterID
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
		AllowPartial:    true,
	}
	var metadataJSON []byte
	if notification.Metadata != nil {
		metadataJSON, err = marshaler.Marshal(notification.Metadata)
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
				r.GetLogger().Error("failed to rollback transaction", zap.Error(rbErr))
			}
			return nil, err
		}
	}
	err = tx.QueryRowContext(ctx,
		`INSERT INTO service_notification (
			master_id, master_uuid, user_id, type, title, content,
			status, metadata, scheduled_at, sent_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			NOW(), NOW()
		) RETURNING id, created_at, updated_at`,
		notification.MasterID, masterUUID, notification.UserID,
		notification.Type, notification.Title,
		notification.Content, notification.Status,
		metadataJSON, notification.ScheduledAt,
		notification.SentAt,
	).Scan(&notification.ID, &notification.CreatedAt, &notification.UpdatedAt)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			r.GetLogger().Error("failed to rollback transaction", zap.Error(rbErr))
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction for notification: %w", err)
	}

	notification.MasterUUID = masterUUID
	return notification, nil
}

// GetByID retrieves a notification by ID.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Notification, error) {
	notification := &Notification{}
	var metadataStr string
	err := r.GetDB().QueryRowContext(ctx,
		`SELECT 
			id, master_id, master_uuid, user_id, type, title,
			content, status, metadata, scheduled_at,
			sent_at, created_at, updated_at
		FROM service_notification 
		WHERE id = $1`,
		id,
	).Scan(
		&notification.ID, &notification.MasterID,
		&notification.MasterUUID, &notification.UserID,
		&notification.Type, &notification.Title,
		&notification.Content, &notification.Status,
		&metadataStr, &notification.ScheduledAt,
		&notification.SentAt,
		&notification.CreatedAt, &notification.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}
	notification.Metadata = &commonpb.Metadata{}
	if metadataStr != "" {
		err := protojson.Unmarshal([]byte(metadataStr), notification.Metadata)
		if err != nil {
			r.GetLogger().Warn("failed to unmarshal notification metadata", zap.Error(err))
			return nil, err
		}
	}
	return notification, nil
}

// Update updates a notification record.
func (r *Repository) Update(ctx context.Context, notification *Notification) error {
	var err error
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
		AllowPartial:    true,
	}
	var metadataJSON []byte
	if notification.Metadata != nil {
		metadataJSON, err = marshaler.Marshal(notification.Metadata)
		if err != nil {
			return err
		}
	}
	result, err := r.GetDB().ExecContext(ctx,
		`UPDATE service_notification 
		SET type = $1, title = $2, content = $3,
			status = $4, metadata = $5, scheduled_at = $6,
			sent_at = $7, updated_at = NOW()
		WHERE id = $8`,
		notification.Type, notification.Title,
		notification.Content, notification.Status,
		metadataJSON, notification.ScheduledAt,
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

// Delete removes a notification and its master record.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	notification, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return r.masterRepo.Delete(ctx, notification.MasterID)
}

// List retrieves a paginated list of notifications.
func (r *Repository) List(ctx context.Context, limit, offset int) ([]*Notification, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, master_uuid, user_id, type, title,
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
			r.GetLogger().Error("error closing rows", zap.Error(cerr))
		}
	}()

	var notifications []*Notification
	for rows.Next() {
		notification := &Notification{}
		var metadataStr string
		err := rows.Scan(
			&notification.ID, &notification.MasterID,
			&notification.MasterUUID, &notification.UserID,
			&notification.Type, &notification.Title,
			&notification.Content, &notification.Status,
			&metadataStr, &notification.ScheduledAt,
			&notification.SentAt,
			&notification.CreatedAt, &notification.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notification.Metadata = &commonpb.Metadata{}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), notification.Metadata)
			if err != nil {
				r.GetLogger().Warn("failed to unmarshal notification metadata", zap.Error(err))
				return nil, err
			}
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}

// ListByUserID retrieves all notifications for a specific user.
func (r *Repository) ListByUserID(ctx context.Context, userID int64, limit, offset int) ([]*Notification, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, master_uuid, user_id, type, title,
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
			r.GetLogger().Error("error closing rows", zap.Error(cerr))
		}
	}()

	var notifications []*Notification
	for rows.Next() {
		notification := &Notification{}
		var metadataStr string
		err := rows.Scan(
			&notification.ID, &notification.MasterID,
			&notification.MasterUUID, &notification.UserID,
			&notification.Type, &notification.Title,
			&notification.Content, &notification.Status,
			&metadataStr, &notification.ScheduledAt,
			&notification.SentAt,
			&notification.CreatedAt, &notification.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notification.Metadata = &commonpb.Metadata{}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), notification.Metadata)
			if err != nil {
				r.GetLogger().Warn("failed to unmarshal notification metadata", zap.Error(err))
				return nil, err
			}
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}

// ListPendingScheduled retrieves all pending notifications that are scheduled to be sent.
func (r *Repository) ListPendingScheduled(ctx context.Context) ([]*Notification, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, master_uuid, user_id, type, title,
			content, status, metadata, scheduled_at,
			sent_at, created_at, updated_at
		FROM service_notification 
		WHERE status = $1 
		AND scheduled_at <= NOW()
		ORDER BY scheduled_at ASC`,
		StatusPending,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			r.GetLogger().Error("error closing rows", zap.Error(cerr))
		}
	}()

	var notifications []*Notification
	for rows.Next() {
		notification := &Notification{}
		var metadataStr string
		err := rows.Scan(
			&notification.ID, &notification.MasterID,
			&notification.MasterUUID, &notification.UserID,
			&notification.Type, &notification.Title,
			&notification.Content, &notification.Status,
			&metadataStr, &notification.ScheduledAt,
			&notification.SentAt,
			&notification.CreatedAt, &notification.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notification.Metadata = &commonpb.Metadata{}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), notification.Metadata)
			if err != nil {
				r.GetLogger().Warn("failed to unmarshal notification metadata", zap.Error(err))
				return nil, err
			}
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}

// --- Broadcast Support ---
// Treat broadcasts as notifications with channel/type 'broadcast'.
func (r *Repository) CreateBroadcast(ctx context.Context, broadcast *Notification) (*Notification, error) {
	broadcast.Type = TypeInApp // or NotificationType("broadcast") if you want a new type
	return r.Create(ctx, broadcast)
}

func (r *Repository) GetBroadcast(ctx context.Context, id int64) (*Notification, error) {
	return r.GetByID(ctx, id)
}

func (r *Repository) ListBroadcasts(ctx context.Context, limit, offset int) ([]*Notification, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT 
			id, master_id, master_uuid, user_id, type, title,
			content, status, metadata, scheduled_at,
			sent_at, created_at, updated_at
		FROM service_notification 
		WHERE type = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		TypeInApp, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			r.GetLogger().Error("error closing rows", zap.Error(cerr))
		}
	}()

	var broadcasts []*Notification
	for rows.Next() {
		broadcast := &Notification{}
		var metadataStr string
		err := rows.Scan(
			&broadcast.ID, &broadcast.MasterID,
			&broadcast.MasterUUID, &broadcast.UserID,
			&broadcast.Type, &broadcast.Title,
			&broadcast.Content, &broadcast.Status,
			&metadataStr, &broadcast.ScheduledAt,
			&broadcast.SentAt,
			&broadcast.CreatedAt, &broadcast.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		broadcast.Metadata = &commonpb.Metadata{}
		if metadataStr != "" {
			err := protojson.Unmarshal([]byte(metadataStr), broadcast.Metadata)
			if err != nil {
				r.GetLogger().Warn("failed to unmarshal broadcast metadata", zap.Error(err))
				return nil, err
			}
		}
		broadcasts = append(broadcasts, broadcast)
	}
	return broadcasts, rows.Err()
}

// --- Notification Event Analytics/Audit ---.
type Event struct {
	EventID        string    `db:"event_id"`
	NotificationID string    `db:"notification_id"`
	UserID         string    `db:"user_id"`
	EventType      string    `db:"event_type"`
	Payload        []byte    `db:"payload"`
	CreatedAt      time.Time `db:"created_at"`
}

func (r *Repository) LogNotificationEvent(ctx context.Context, event *Event) error {
	_, err := r.GetDB().ExecContext(ctx,
		`INSERT INTO notification_event (
			notification_id, user_id, event_type, payload, created_at
		) VALUES ($1, $2, $3, $4, NOW())`,
		event.NotificationID, event.UserID, event.EventType, event.Payload,
	)
	return err
}

func (r *Repository) ListNotificationEvents(ctx context.Context, notificationID int64, limit, offset int) ([]*Event, error) {
	rows, err := r.GetDB().QueryContext(ctx,
		`SELECT id, notification_id, user_id, event_type, payload, created_at
		FROM notification_event
		WHERE notification_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		notificationID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			r.GetLogger().Error("error closing rows", zap.Error(cerr))
		}
	}()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		err := rows.Scan(
			&event.EventID, &event.NotificationID, &event.UserID,
			&event.EventType, &event.Payload, &event.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
