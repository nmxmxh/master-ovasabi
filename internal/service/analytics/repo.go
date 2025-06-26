package analytics

import (
	"context"
	"database/sql"
	"encoding/json"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	metadatautil "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"google.golang.org/protobuf/encoding/protojson"

	analyticspb "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	"go.uber.org/zap"
)

// PostgresRepository provides analytics event storage.
type Repository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewRepository(db *sql.DB, log *zap.Logger) *Repository {
	return &Repository{db: db, log: log}
}

func (r *Repository) TrackEvent(ctx context.Context, event *analyticspb.Event) error {
	if err := metadatautil.ValidateMetadata(event.Metadata); err != nil {
		return err
	}

	var metadataJSON interface{}
	if event.Metadata != nil {
		b, err := metadatautil.MarshalCanonical(event.Metadata)
		if err != nil {
			return err
		}
		metadataJSON = b
	} else {
		metadataJSON = nil
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO service_analytics_event
		(id, master_id, master_uuid, user_id, event_type, entity_id, entity_type, properties, timestamp, metadata, campaign_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, to_timestamp($9), $10, $11)
	`, event.Id, event.MasterId, event.MasterUuid, event.UserId, event.EventType, event.EntityId, event.EntityType, event.Properties, event.Timestamp, metadataJSON, event.CampaignId)
	return err
}

func (r *Repository) BatchTrackEvents(ctx context.Context, events []*analyticspb.Event) (success, fail int, firstErr error) {
	success, fail = 0, 0
	firstErr = nil
	for _, event := range events {
		if err := r.TrackEvent(ctx, event); err != nil {
			fail++
			if firstErr == nil {
				firstErr = err
			}
			if r.log != nil {
				r.log.Warn("Failed to track event in batch", zap.String("event_id", event.Id), zap.Error(err))
			}
		} else {
			success++
		}
	}
	// success: number of events successfully tracked
	// fail: number of events that failed to track
	// firstErr: the first error encountered (if any), or nil if all succeeded
	return success, fail, firstErr
}

// scanEventFromScanner is a helper to scan a single event from a sql.Row or sql.Rows.
func (r *Repository) scanEventFromScanner(scanner interface{ Scan(...interface{}) error }) (*analyticspb.Event, error) {
	var e analyticspb.Event
	var props map[string]string
	var ts float64
	var metaRaw sql.NullString
	var campaignID int64
	if err := scanner.Scan(&e.Id, &e.MasterId, &e.MasterUuid, &e.UserId, &e.EventType, &e.EntityId, &e.EntityType, &props, &ts, &metaRaw, &campaignID); err != nil {
		return nil, err
	}
	e.Properties = props
	e.Timestamp = int64(ts)
	e.CampaignId = campaignID
	if metaRaw.Valid && metaRaw.String != "" {
		meta := &commonpb.Metadata{ServiceSpecific: metadatautil.NewStructFromMap(nil, r.log)}
		if err := protojson.Unmarshal([]byte(metaRaw.String), meta); err != nil {
			r.log.Warn("Failed to unmarshal event metadata",
				zap.Error(err),
				zap.String("event_id", e.Id),
				zap.String("metadata", metaRaw.String))
			// Continue with empty metadata rather than skipping the event
			meta = &commonpb.Metadata{ServiceSpecific: metadatautil.NewStructFromMap(nil, r.log)}
		}
		e.Metadata = meta
	}
	return &e, nil
}

// scanEventsFromRows is a helper to reduce duplication by iterating over rows.
func (r *Repository) scanEventsFromRows(rows *sql.Rows) ([]*analyticspb.Event, error) {
	var events []*analyticspb.Event
	for rows.Next() {
		event, err := r.scanEventFromScanner(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *Repository) GetUserEvents(ctx context.Context, userID string, campaignID int64, page, pageSize int) ([]*analyticspb.Event, int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, master_id, master_uuid, user_id, event_type, entity_id, entity_type, properties, EXTRACT(EPOCH FROM timestamp), metadata, campaign_id
		FROM service_analytics_event
		WHERE user_id = $1 AND campaign_id = $2
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4
	`, userID, campaignID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	events, err := r.scanEventsFromRows(rows)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_analytics_event WHERE user_id = $1 AND campaign_id = $2`, userID, campaignID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (r *Repository) GetProductEvents(ctx context.Context, productID string, campaignID int64, page, pageSize int) ([]*analyticspb.Event, int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, master_id, master_uuid, user_id, event_type, entity_id, entity_type, properties, EXTRACT(EPOCH FROM timestamp), metadata, campaign_id
		FROM service_analytics_event
		WHERE entity_id = $1 AND campaign_id = $2
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4
	`, productID, campaignID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	events, err := r.scanEventsFromRows(rows)
	if err != nil {
		return nil, 0, err
	}

	var total int
	row := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_analytics_event WHERE entity_id = $1 AND campaign_id = $2`, productID, campaignID)
	err = row.Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	return events, total, nil
}

func (r *Repository) GetEvent(ctx context.Context, eventID string) (*analyticspb.Event, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, master_id, master_uuid, user_id, event_type, entity_id, entity_type, properties, EXTRACT(EPOCH FROM timestamp), metadata, campaign_id
		FROM service_analytics_event
		WHERE id = $1
	`, eventID)
	return r.scanEventFromScanner(row)
}

func (r *Repository) GetReport(ctx context.Context, reportID string) (*analyticspb.Report, error) {
	// Example: fetch a report by ID (dummy implementation)
	row := r.db.QueryRowContext(ctx, `SELECT id, name, description, parameters, data, created_at FROM service_analytics_report WHERE id = $1`, reportID)
	var report analyticspb.Report
	var paramsRaw []byte
	var data []byte
	var createdAt float64
	if err := row.Scan(&report.Id, &report.Name, &report.Description, &paramsRaw, &data, &createdAt); err != nil {
		return nil, err
	}
	if len(paramsRaw) > 0 {
		var params map[string]string
		if err := json.Unmarshal(paramsRaw, &params); err != nil {
			r.log.Warn("Failed to unmarshal report parameters",
				zap.Error(err),
				zap.ByteString("paramsRaw", paramsRaw),
				zap.String("report_id", reportID))
			// Initialize empty parameters rather than leaving them nil
			params = make(map[string]string)
		}
		report.Parameters = params
	} else {
		// Initialize empty parameters if none provided
		report.Parameters = make(map[string]string)
	}
	report.Data = data
	report.CreatedAt = int64(createdAt)
	return &report, nil
}

func (r *Repository) ListReports(ctx context.Context, page, pageSize int) ([]*analyticspb.Report, int, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, description, parameters, data, created_at FROM service_analytics_report ORDER BY created_at DESC LIMIT $1 OFFSET $2`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var reports []*analyticspb.Report
	for rows.Next() {
		var report analyticspb.Report
		var paramsRaw []byte
		var data []byte
		var createdAt float64
		if err := rows.Scan(&report.Id, &report.Name, &report.Description, &paramsRaw, &data, &createdAt); err != nil {
			return nil, 0, err
		}
		if len(paramsRaw) > 0 {
			var params map[string]string
			if err := json.Unmarshal(paramsRaw, &params); err != nil {
				r.log.Warn("Failed to unmarshal report parameters", zap.Error(err), zap.ByteString("paramsRaw", paramsRaw), zap.String("report_id", report.Id))
				report.Parameters = make(map[string]string)
			} else {
				report.Parameters = params
			}
		} else {
			// Initialize empty parameters if none provided
			report.Parameters = make(map[string]string)
		}
		report.Data = data
		report.CreatedAt = int64(createdAt)
		reports = append(reports, &report)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	row := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_analytics_report`)
	err = row.Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	return reports, total, nil
}

// CountEventsByType returns the number of analytics events with the given event_type.
func (r *Repository) CountEventsByType(ctx context.Context, eventType string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM service_analytics_event WHERE event_type = $1`, eventType).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
