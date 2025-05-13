package analyticsrepo

import (
	"context"
	"database/sql"
	"errors"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"google.golang.org/protobuf/encoding/protojson"

	analyticspb "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
)

// PostgresRepository provides analytics event storage.
type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) TrackEvent(ctx context.Context, event *analyticspb.Event) error {
	if err := metadatautil.ValidateMetadata(event.Metadata); err != nil {
		return err
	}

	var metadataJSON interface{}
	if event.Metadata != nil {
		b, err := protojson.Marshal(event.Metadata)
		if err != nil {
			return err
		}
		metadataJSON = b
	} else {
		metadataJSON = nil
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO service_analytics_event
		(id, master_id, user_id, event_type, entity_id, entity_type, properties, timestamp, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, to_timestamp($8), $9)
	`, event.Id, event.MasterId, event.UserId, event.EventType, event.EntityId, event.EntityType, event.Properties, event.Timestamp, metadataJSON)
	return err
}

func (r *PostgresRepository) BatchTrackEvents(ctx context.Context, events []*analyticspb.Event) (success, fail int, err error) {
	success, fail = 0, 0
	for _, event := range events {
		if err := r.TrackEvent(ctx, event); err != nil {
			fail++
		} else {
			success++
		}
	}
	return success, fail, nil
}

func (r *PostgresRepository) GetUserEvents(ctx context.Context, userID string, page, pageSize int) ([]*analyticspb.Event, int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, master_id, user_id, event_type, entity_id, entity_type, properties, EXTRACT(EPOCH FROM timestamp), metadata
		FROM service_analytics_event
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*analyticspb.Event
	for rows.Next() {
		var e analyticspb.Event
		var props map[string]string
		var ts float64
		var metaRaw sql.NullString
		if err := rows.Scan(&e.Id, &e.MasterId, &e.UserId, &e.EventType, &e.EntityId, &e.EntityType, &props, &ts, &metaRaw); err != nil {
			return nil, 0, err
		}
		e.Properties = props
		e.Timestamp = int64(ts)
		if metaRaw.Valid && metaRaw.String != "" {
			meta := &commonpb.Metadata{}
			if err := protojson.Unmarshal([]byte(metaRaw.String), meta); err == nil {
				e.Metadata = meta
			}
		}
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	// Count total
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_analytics_event WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (r *PostgresRepository) GetProductEvents(_ context.Context, _ string, _, _ int) ([]*analyticspb.Event, int, error) {
	// TODO: implement GetProductEvents logic
	return nil, 0, errors.New("not implemented")
}

func (r *PostgresRepository) GetReport(_ context.Context, _ string) (*analyticspb.Report, error) {
	// TODO: implement GetReport logic
	return nil, errors.New("not implemented")
}

func (r *PostgresRepository) ListReports(_ context.Context, _, _ int) ([]*analyticspb.Report, int, error) {
	// TODO: implement ListReports logic
	return nil, 0, errors.New("not implemented")
}
