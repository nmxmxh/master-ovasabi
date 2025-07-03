package nexus

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

// CanonicalEvent wraps the existing Event struct and adds extensibility for multi-event orchestration and metadata.
type CanonicalEvent struct {
	ID            uuid.UUID             `json:"id" db:"id"`
	MasterID      int64                 `json:"master_id" db:"master_id"`
	EntityType    repository.EntityType `json:"entity_type" db:"entity_type"`
	EventType     string                `json:"event_type" db:"event_type"`
	Payload       *commonpb.Payload     `json:"payload" db:"payload"`
	Metadata      *commonpb.Metadata    `json:"metadata" db:"metadata"`
	Status        string                `json:"status" db:"status"`
	CreatedAt     time.Time             `json:"created_at" db:"created_at"`
	ProcessedAt   *time.Time            `json:"processed_at" db:"processed_at"`
	PatternID     *string               `json:"pattern_id,omitempty" db:"pattern_id"` // For pattern-based orchestration
	Step          *string               `json:"step,omitempty" db:"step"`             // For multi-step events
	Retries       int                   `json:"retries" db:"retries"`
	Error         *string               `json:"error,omitempty" db:"error"`
	NexusSequence *uint64               `json:"nexus_sequence,omitempty" db:"nexus_sequence"` // Temporal ordering sequence
}

// EventRepository defines the interface for event persistence and orchestration.
type EventRepository interface {
	SaveEvent(ctx context.Context, event *CanonicalEvent) error
	GetEvent(ctx context.Context, id uuid.UUID) (*CanonicalEvent, error)
	ListEventsByMaster(ctx context.Context, masterID int64) ([]*CanonicalEvent, error)
	ListPendingEvents(ctx context.Context, entityType repository.EntityType) ([]*CanonicalEvent, error)
	UpdateEventStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error
	ListEventsByPattern(ctx context.Context, patternID string) ([]*CanonicalEvent, error)
}

// SQLEventRepository is a SQL-backed implementation of EventRepository.
type SQLEventRepository struct {
	db  *sql.DB
	log *zap.Logger
}

// NewSQLEventRepository creates a new SQL event repository.
func NewSQLEventRepository(db *sql.DB, log *zap.Logger) *SQLEventRepository {
	return &SQLEventRepository{
		db:  db,
		log: log,
	}
}

// Helper function to safely get request ID from context.
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

// SaveEvent persists an event to the database.
func (r *SQLEventRepository) SaveEvent(ctx context.Context, event *CanonicalEvent) error {
	r.log.Info("Saving event",
		zap.String("event_id", event.ID.String()),
		zap.String("event_type", event.EventType),
		zap.String("request_id", getRequestID(ctx)),
	)

	metaBytes, err := protojson.Marshal(event.Metadata)
	if err != nil {
		if r.log != nil {
			r.log.Error("Failed to marshal event metadata",
				zap.Error(err),
				zap.String("event_id", event.ID.String()),
				zap.String("request_id", getRequestID(ctx)),
			)
		}
		return err
	}

	payloadBytes, err := protojson.Marshal(event.Payload)
	if err != nil {
		if r.log != nil {
			r.log.Error("Failed to marshal event payload",
				zap.Error(err),
				zap.String("event_id", event.ID.String()),
				zap.String("request_id", getRequestID(ctx)),
			)
		}
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO service_event (
			id, master_id, entity_type, event_type, payload, metadata, status, created_at, processed_at, pattern_id, step, retries, error, nexus_sequence
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`,
		event.ID, event.MasterID, event.EntityType, event.EventType, payloadBytes, metaBytes, event.Status, event.CreatedAt, event.ProcessedAt, event.PatternID, event.Step, event.Retries, event.Error, event.NexusSequence,
	)
	if err != nil {
		if r.log != nil {
			r.log.Error("Failed to save event",
				zap.Error(err),
				zap.String("event_id", event.ID.String()),
				zap.String("event_type", event.EventType),
				zap.String("request_id", getRequestID(ctx)),
			)
		}
		return err
	}

	if r.log != nil {
		r.log.Info("Event saved successfully",
			zap.String("event_id", event.ID.String()),
			zap.String("event_type", event.EventType),
			zap.String("request_id", getRequestID(ctx)),
		)
	}

	return nil
}

// GetEvent retrieves an event from the database.
func (r *SQLEventRepository) GetEvent(ctx context.Context, id uuid.UUID) (*CanonicalEvent, error) {
	r.log.Info("Getting event",
		zap.String("event_id", id.String()),
		zap.String("request_id", getRequestID(ctx)),
	)

	row := r.db.QueryRowContext(ctx, `
		SELECT id, master_id, entity_type, event_type, payload, metadata, status, created_at, processed_at, pattern_id, step, retries, error, nexus_sequence
		FROM service_event WHERE id = $1
	`, id)

	var event CanonicalEvent
	var payloadBytes, metaBytes []byte
	if err := row.Scan(&event.ID, &event.MasterID, &event.EntityType, &event.EventType, &payloadBytes, &metaBytes, &event.Status, &event.CreatedAt, &event.ProcessedAt, &event.PatternID, &event.Step, &event.Retries, &event.Error, &event.NexusSequence); err != nil {
		if r.log != nil {
			r.log.Error("Failed to scan event row",
				zap.Error(err),
				zap.String("event_id", id.String()),
				zap.String("request_id", getRequestID(ctx)),
			)
		}
		return nil, err
	}

	if len(payloadBytes) > 0 {
		event.Payload = &commonpb.Payload{}
		if err := protojson.Unmarshal(payloadBytes, event.Payload); err != nil {
			r.log.Error("Failed to unmarshal event payload", zap.Error(err), zap.String("event_id", id.String()))
			return nil, err
		}
	}

	if len(metaBytes) > 0 {
		event.Metadata = &commonpb.Metadata{}
		if err := protojson.Unmarshal(metaBytes, event.Metadata); err != nil {
			if r.log != nil {
				r.log.Error("Failed to unmarshal event metadata",
					zap.Error(err),
					zap.String("event_id", id.String()),
					zap.String("request_id", getRequestID(ctx)),
				)
			}
			return nil, err
		}
	}

	if r.log != nil {
		r.log.Info("Event retrieved successfully",
			zap.String("event_id", id.String()),
			zap.String("event_type", event.EventType),
			zap.String("request_id", getRequestID(ctx)),
		)
	}

	return &event, nil
}

func (r *SQLEventRepository) ListEventsByMaster(ctx context.Context, masterID int64) ([]*CanonicalEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, master_id, entity_type, event_type, payload, metadata, status, created_at, processed_at, pattern_id, step, retries, error, nexus_sequence
		FROM service_event WHERE master_id = $1 ORDER BY nexus_sequence DESC, created_at DESC
	`, masterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []*CanonicalEvent
	for rows.Next() {
		var event CanonicalEvent
		var payloadBytes, metaBytes []byte
		if err := rows.Scan(&event.ID, &event.MasterID, &event.EntityType, &event.EventType, &payloadBytes, &metaBytes, &event.Status, &event.CreatedAt, &event.ProcessedAt, &event.PatternID, &event.Step, &event.Retries, &event.Error, &event.NexusSequence); err != nil {
			return nil, err
		}
		if len(payloadBytes) > 0 {
			event.Payload = &commonpb.Payload{}
			if err := protojson.Unmarshal(payloadBytes, event.Payload); err != nil {
				return nil, err
			}
		}
		if len(metaBytes) > 0 {
			event.Metadata = &commonpb.Metadata{}
			if err := protojson.Unmarshal(metaBytes, event.Metadata); err != nil {
				return nil, err
			}
		}
		events = append(events, &event)
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (r *SQLEventRepository) ListPendingEvents(ctx context.Context, entityType repository.EntityType) ([]*CanonicalEvent, error) {
	r.log.Info("Listing pending events",
		zap.String("event_type", string(entityType)),
		zap.String("request_id", getRequestID(ctx)),
	)

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, master_id, entity_type, event_type, payload, metadata, status, created_at, processed_at, pattern_id, step, retries, error, nexus_sequence
		FROM service_event WHERE entity_type = $1 AND status = 'pending' ORDER BY nexus_sequence ASC, created_at ASC
	`, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []*CanonicalEvent
	for rows.Next() {
		var event CanonicalEvent
		var payloadBytes, metaBytes []byte
		if err := rows.Scan(&event.ID, &event.MasterID, &event.EntityType, &event.EventType, &payloadBytes, &metaBytes, &event.Status, &event.CreatedAt, &event.ProcessedAt, &event.PatternID, &event.Step, &event.Retries, &event.Error, &event.NexusSequence); err != nil {
			return nil, err
		}
		if len(payloadBytes) > 0 {
			event.Payload = &commonpb.Payload{}
			if err := protojson.Unmarshal(payloadBytes, event.Payload); err != nil {
				return nil, err
			}
		}
		if len(metaBytes) > 0 {
			event.Metadata = &commonpb.Metadata{}
			if err := protojson.Unmarshal(metaBytes, event.Metadata); err != nil {
				return nil, err
			}
		}
		events = append(events, &event)
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func (r *SQLEventRepository) UpdateEventStatus(ctx context.Context, id uuid.UUID, status string, errMsg *string) error {
	r.log.Info("Updating event status",
		zap.String("event_id", id.String()),
		zap.String("status", status),
		zap.String("request_id", getRequestID(ctx)),
	)

	_, err := r.db.ExecContext(ctx, `
		UPDATE service_event SET status = $1, error = $2, processed_at = NOW() WHERE id = $3
	`, status, errMsg, id)
	return err
}

func (r *SQLEventRepository) ListEventsByPattern(ctx context.Context, patternID string) ([]*CanonicalEvent, error) {
	r.log.Info("Listing events by pattern",
		zap.String("pattern", patternID),
		zap.String("request_id", getRequestID(ctx)),
	)

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, master_id, entity_type, event_type, payload, metadata, status, created_at, processed_at, pattern_id, step, retries, error, nexus_sequence
		FROM service_event WHERE pattern_id = $1 ORDER BY nexus_sequence ASC
	`, patternID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []*CanonicalEvent
	for rows.Next() {
		var event CanonicalEvent
		var payloadBytes, metaBytes []byte
		if err := rows.Scan(&event.ID, &event.MasterID, &event.EntityType, &event.EventType, &payloadBytes, &metaBytes, &event.Status, &event.CreatedAt, &event.ProcessedAt, &event.PatternID, &event.Step, &event.Retries, &event.Error, &event.NexusSequence); err != nil {
			return nil, err
		}
		if len(payloadBytes) > 0 {
			event.Payload = &commonpb.Payload{}
			if err := protojson.Unmarshal(payloadBytes, event.Payload); err != nil {
				return nil, err
			}
		}
		if len(metaBytes) > 0 {
			event.Metadata = &commonpb.Metadata{}
			if err := protojson.Unmarshal(metaBytes, event.Metadata); err != nil {
				return nil, err
			}
		}
		events = append(events, &event)
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return events, nil
}
