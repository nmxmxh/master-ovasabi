package nexus

import (
	"context"
	"database/sql"
	"fmt"

	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventFilter represents the filtering criteria for listing events.
type EventFilter struct {
	Type      string
	StartTime *timestamppb.Timestamp
	EndTime   *timestamppb.Timestamp
	Limit     int32
}

type EventRepository struct {
	db *sql.DB
}

func (r *EventRepository) GetEvent(ctx context.Context, eventID string) (*nexusv1.EventResponse, error) {
	var event nexusv1.EventResponse
	err := r.db.QueryRowContext(ctx, `
        SELECT success, event_id, event_type, message, metadata, payload
        FROM events
        WHERE event_id = $1
    `, eventID).Scan(
		&event.Success,
		&event.EventId,
		&event.EventType,
		&event.Message,
		&event.Metadata,
		&event.Payload,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, graceful.WrapErr(ctx, codes.NotFound, "event not found", err)
		}
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to get event", err)
	}
	return &event, nil
}

func (r *EventRepository) ListEvents(ctx context.Context, filter *EventFilter) ([]*nexusv1.EventResponse, error) {
	query := `
        SELECT success, event_id, event_type, message, metadata, payload
        FROM events
        WHERE 1=1
    `
	args := []interface{}{}
	argCount := 1

	if filter != nil {
		if filter.Type != "" {
			query += fmt.Sprintf(" AND event_type = $%d", argCount)
			args = append(args, filter.Type)
			argCount++
		}
		if filter.StartTime != nil {
			query += fmt.Sprintf(" AND created_at >= $%d", argCount)
			args = append(args, filter.StartTime.AsTime())
			argCount++
		}
		if filter.EndTime != nil {
			query += fmt.Sprintf(" AND created_at <= $%d", argCount)
			args = append(args, filter.EndTime.AsTime())
			argCount++
		}
	}

	query += " ORDER BY created_at DESC"
	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to list events", err)
	}
	defer rows.Close()

	var events []*nexusv1.EventResponse
	for rows.Next() {
		var event nexusv1.EventResponse
		err := rows.Scan(
			&event.Success,
			&event.EventId,
			&event.EventType,
			&event.Message,
			&event.Metadata,
			&event.Payload,
		)
		if err != nil {
			return nil, graceful.WrapErr(ctx, codes.Internal, "failed to scan event", err)
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, graceful.WrapErr(ctx, codes.Internal, "error iterating events", err)
	}

	return events, nil
}

func (r *EventRepository) CreateEvent(ctx context.Context, event *nexusv1.EventResponse) error {
	_, err := r.db.ExecContext(ctx, `
        INSERT INTO events (success, event_id, event_type, message, metadata, payload)
        VALUES ($1, $2, $3, $4, $5, $6)
    `,
		event.Success,
		event.EventId,
		event.EventType,
		event.Message,
		event.Metadata,
		event.Payload,
	)
	if err != nil {
		return graceful.WrapErr(ctx, codes.Internal, "failed to create event", err)
	}
	return nil
}

func (r *EventRepository) UpdateEvent(ctx context.Context, event *nexusv1.EventResponse) error {
	result, err := r.db.ExecContext(ctx, `
        UPDATE events
        SET success = $1, event_type = $2, message = $3, metadata = $4, payload = $5
        WHERE event_id = $6
    `,
		event.Success,
		event.EventType,
		event.Message,
		event.Metadata,
		event.Payload,
		event.EventId,
	)
	if err != nil {
		return graceful.WrapErr(ctx, codes.Internal, "failed to update event", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return graceful.WrapErr(ctx, codes.Internal, "failed to get rows affected", err)
	}
	if rows == 0 {
		return graceful.WrapErr(ctx, codes.NotFound, "event not found", nil)
	}
	return nil
}

func (r *EventRepository) DeleteEvent(ctx context.Context, eventID string) error {
	result, err := r.db.ExecContext(ctx, `
        DELETE FROM events
        WHERE event_id = $1
    `, eventID)
	if err != nil {
		return graceful.WrapErr(ctx, codes.Internal, "failed to delete event", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return graceful.WrapErr(ctx, codes.Internal, "failed to get rows affected", err)
	}
	if rows == 0 {
		return graceful.WrapErr(ctx, codes.NotFound, "event not found", nil)
	}
	return nil
}
