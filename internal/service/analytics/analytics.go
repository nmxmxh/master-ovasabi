// Analytics Service (GDPR-Compliant, Extensible)
// ----------------------------------------------
// Implements the canonical analytics service for event capture, enrichment, and listing.
// - Uses robust, versioned, and GDPR-compliant metadata (see metadata.go).
// - All repository interactions are handled via the RepositoryItf interface.
// - All event creation uses BuildAnalyticsMetadata.
// - Stores events in-memory (replace with DB/repo in production).
// - Exposes gRPC-compatible methods: CaptureEvent, ListEvents, EnrichEventMetadata.
// - Logs all actions and errors.
//
// Usage: Use this service for all analytics event ingestion and querying.
//
// For more, see docs/services/metadata.md and docs/amadeus/amadeus_context.md.

package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	analytics "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type Event struct {
	ID        string
	Timestamp time.Time
	Metadata  *commonpb.Metadata
}

type RepositoryItf interface {
	TrackEvent(ctx context.Context, event *analytics.Event) error
	BatchTrackEvents(ctx context.Context, events []*analytics.Event) (int, int, error)
	GetUserEvents(ctx context.Context, userID string, campaignID int64, page, pageSize int) ([]*analytics.Event, int, error)
	GetProductEvents(ctx context.Context, productID string, campaignID int64, page, pageSize int) ([]*analytics.Event, int, error)
	GetReport(ctx context.Context, reportID string) (*analytics.Report, error)
	ListReports(ctx context.Context, page, pageSize int) ([]*analytics.Report, int, error)
	CountEventsByType(ctx context.Context, eventType string) (int, error)
	GetEvent(ctx context.Context, eventID string) (*analytics.Event, error)
}

type Service struct {
	analytics.UnimplementedAnalyticsServiceServer
	log          *zap.Logger
	repo         *Repository
	Cache        *redis.Cache
	eventEmitter events.EventEmitter
	eventEnabled bool
}

func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool) analytics.AnalyticsServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

var _ analytics.AnalyticsServiceServer = (*Service)(nil)

// Define a package-level error for event not found.
var ErrEventNotFound = status.Error(codes.NotFound, "analytics event not found")

// Helper: Convert *structpb.Struct to map[string]string.
func structToStringMap(s *structpb.Struct) map[string]string {
	m := make(map[string]string)
	if s == nil {
		return m
	}
	for k, v := range s.Fields {
		if str, ok := v.Kind.(*structpb.Value_StringValue); ok {
			m[k] = str.StringValue
		}
	}
	return m
}

// Adapter to bridge s.eventEmitter to the required orchestration EventEmitter interface.
type EventEmitterAdapter struct {
	Emitter events.EventEmitter
}

func (a *EventEmitterAdapter) EmitRawEventWithLogging(ctx context.Context, log *zap.Logger, eventType, eventID string, payload []byte) (string, bool) {
	if a.Emitter == nil {
		log.Warn("Event emitter not configured", zap.String("event_type", eventType))
		return "", false
	}
	return a.Emitter.EmitRawEventWithLogging(ctx, log, eventType, eventID, payload)
}

func (a *EventEmitterAdapter) EmitEventWithLogging(ctx context.Context, event interface{}, log *zap.Logger, eventType, eventID string, meta *commonpb.Metadata) (string, bool) {
	if a.Emitter == nil {
		log.Warn("Event emitter not configured", zap.String("event_type", eventType))
		return "", false
	}
	return a.Emitter.EmitEventWithLogging(ctx, event, log, eventType, eventID, meta)
}

// CaptureEvent ingests a new analytics event with robust, GDPR-compliant metadata.
func (s *Service) CaptureEvent(ctx context.Context, req *analytics.CaptureEventRequest) (*analytics.CaptureEventResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	meta, err := BuildAnalyticsMetadata(
		req.EventType,
		req.UserId,
		req.UserEmail,
		req.Properties.AsMap(),
		req.Groups.AsMap(),
		req.Context.AsMap(),
		req.GdprObscure,
		map[string]interface{}{}, // serviceSpecific
		s.log,                    // logger
	)
	if err != nil {
		s.log.Error("Failed to build analytics metadata", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to build analytics metadata", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
		return nil, err
	}
	eventID := generateEventID()
	event := &analytics.Event{
		Id:         eventID,
		UserId:     req.UserId,
		EventType:  req.EventType,
		Properties: structToStringMap(req.Properties),
		Metadata:   meta,
		Timestamp:  time.Now().Unix(),
		CampaignId: req.CampaignId,
	}
	if err := s.repo.TrackEvent(ctx, event); err != nil {
		s.log.Error("Failed to track analytics event", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to track analytics event", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
		return nil, err
	}
	s.log.Info("Captured analytics event", zap.String("event_id", eventID), zap.String("event_type", req.EventType))
	success := graceful.WrapSuccess(ctx, codes.OK, "analytics event captured", &analytics.CaptureEventResponse{EventId: eventID}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     eventID,
		CacheValue:   event,
		CacheTTL:     10 * time.Minute,
		Metadata:     event.Metadata,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
		EventType:    "analytics.event_tracked",
		EventID:      eventID,
		PatternType:  "analytics_event",
		PatternID:    eventID,
		PatternMeta:  event.Metadata,
	})
	return &analytics.CaptureEventResponse{EventId: eventID}, nil
}

// ListEvents returns all captured analytics events (paginated in production).
func (s *Service) ListEvents(ctx context.Context, req *analytics.ListEventsRequest) (*analytics.ListEventsResponse, error) {
	s.log.Info("ListEvents called", zap.Any("request", req))
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// ListEventsRequest has no fields; fetch all events for demo (userId/campaignId empty, page 1, pageSize 100)
	userEvents, _, err := s.repo.GetUserEvents(ctx, "", 0, 1, 100)
	if err != nil {
		s.log.Error("failed to list analytics events", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to list analytics events", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
		return nil, err
	}
	analyticsEvents := make([]*analytics.AnalyticsEvent, 0, len(userEvents))
	for _, e := range userEvents {
		analyticsEvents = append(analyticsEvents, &analytics.AnalyticsEvent{
			EventId:    e.Id,
			Timestamp:  e.Timestamp,
			Metadata:   e.Metadata,
			CampaignId: e.CampaignId,
		})
	}
	return &analytics.ListEventsResponse{Events: analyticsEvents}, nil
}

// EnrichEventMetadata allows for post-hoc enrichment of event metadata.
func (s *Service) EnrichEventMetadata(ctx context.Context, req *analytics.EnrichEventMetadataRequest) (*analytics.EnrichEventMetadataResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// Fetch the specific event by ID instead of loading all events.
	eventToUpdate, err := s.repo.GetEvent(ctx, req.EventId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Warn("Event not found for enrichment", zap.String("event_id", req.EventId))
			errCtx := graceful.WrapErr(ctx, codes.NotFound, "event not found for enrichment", ErrEventNotFound)
			errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
			return nil, ErrEventNotFound
		}
		s.log.Error("Failed to fetch event for enrichment", zap.Error(err), zap.String("event_id", req.EventId))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to fetch event for enrichment", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
		return nil, err
	}
	// Merge new fields into existing metadata using canonical helper
	if eventToUpdate.Metadata != nil && req.NewFields != nil {
		for k, v := range req.NewFields.Fields {
			if err := metadata.SetServiceSpecificField(eventToUpdate.Metadata, "analytics", k, v.AsInterface()); err != nil {
				s.log.Warn("Failed to set analytics field", zap.String("field", k), zap.Error(err))
			}
		}
	}
	// Persist updated event
	if err := s.repo.TrackEvent(ctx, eventToUpdate); err != nil {
		s.log.Error("failed to update analytics event", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to update analytics event", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
		return nil, err
	}
	s.log.Info("Enriched analytics event metadata", zap.String("event_id", req.EventId))
	success := graceful.WrapSuccess(ctx, codes.OK, "analytics event enriched", &analytics.EnrichEventMetadataResponse{Success: true}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     req.EventId,
		CacheValue:   eventToUpdate.Metadata,
		CacheTTL:     10 * time.Minute,
		Metadata:     eventToUpdate.Metadata,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
		EventType:    "analytics.event_enriched",
		EventID:      req.EventId,
		PatternType:  "analytics_event",
		PatternID:    req.EventId,
		PatternMeta:  eventToUpdate.Metadata,
	})
	return &analytics.EnrichEventMetadataResponse{Success: true}, nil
}

// generateEventID generates a unique event ID using UUIDv7 (canonical pattern).
func generateEventID() string {
	return utils.NewUUIDOrDefault()
}

func (s *Service) TrackEvent(ctx context.Context, req *analytics.TrackEventRequest) (*analytics.TrackEventResponse, error) {
	event := req.GetEvent()
	if event == nil || event.MasterId == 0 {
		return nil, status.Error(codes.InvalidArgument, "event and master_id are required")
	}
	if err := s.repo.TrackEvent(ctx, event); err != nil {
		s.log.Error("failed to track event", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to track event", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
		return nil, status.Error(codes.Internal, "failed to track event")
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "analytics event tracked", &analytics.TrackEventResponse{Success: true}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     event.Id,
		CacheValue:   event.Metadata,
		CacheTTL:     10 * time.Minute,
		Metadata:     event.Metadata,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
		EventType:    "analytics.event_tracked",
		EventID:      event.Id,
		PatternType:  "analytics_event",
		PatternID:    event.Id,
		PatternMeta:  event.Metadata,
	})
	return &analytics.TrackEventResponse{Success: true}, nil
}

func (s *Service) BatchTrackEvents(ctx context.Context, req *analytics.BatchTrackEventsRequest) (*analytics.BatchTrackEventsResponse, error) {
	for _, event := range req.GetEvents() {
		if event.MasterId == 0 {
			return nil, status.Error(codes.InvalidArgument, "all events must have master_id")
		}
	}
	successCount, failCount, firstErr := s.repo.BatchTrackEvents(ctx, req.GetEvents())
	if firstErr != nil {
		s.log.Error("failed to batch track events", zap.Error(firstErr))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to batch track events", firstErr)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
		return nil, status.Error(codes.Internal, "failed to batch track events")
	}
	if failCount > 0 {
		s.log.Warn("Partial failure in batch track events", zap.Int("success", successCount), zap.Int("fail", failCount))
		// Optionally, include a warning in the response metadata or orchestrate a warning event
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "analytics batch tracked", &analytics.BatchTrackEventsResponse{
		SuccessCount: utils.ToInt32(successCount),
		FailureCount: utils.ToInt32(failCount),
	}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     "",
		CacheValue:   nil,
		CacheTTL:     10 * time.Minute,
		Metadata:     nil,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
		EventType:    "analytics.batch_tracked",
		EventID:      "",
		PatternType:  "analytics_event",
		PatternID:    "",
		PatternMeta:  nil,
	})
	return &analytics.BatchTrackEventsResponse{
		SuccessCount: utils.ToInt32(successCount),
		FailureCount: utils.ToInt32(failCount),
	}, nil
}

func (s *Service) GetUserEvents(ctx context.Context, req *analytics.GetUserEventsRequest) (*analytics.GetUserEventsResponse, error) {
	userEvents, total, err := s.repo.GetUserEvents(ctx, req.GetUserId(), req.GetCampaignId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		s.log.Error("failed to get user events", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to get user events", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
		return nil, status.Error(codes.Internal, "failed to get user events")
	}
	if total > int(^int32(0)) || total < 0 {
		return nil, fmt.Errorf("total count overflows int32")
	}

	totalPages := (total + int(req.GetPageSize()) - 1) / int(req.GetPageSize())

	return &analytics.GetUserEventsResponse{
		Events:     userEvents,
		TotalCount: utils.ToInt32(total),
		Page:       req.GetPage(),
		TotalPages: utils.ToInt32(totalPages),
	}, nil
}

func (s *Service) GetProductEvents(ctx context.Context, req *analytics.GetProductEventsRequest) (*analytics.GetProductEventsResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	productEvents, total, err := s.repo.GetProductEvents(ctx, req.GetProductId(), req.GetCampaignId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		s.log.Error("failed to get product events", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to get product events", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
		return nil, status.Error(codes.Internal, "failed to get product events")
	}
	totalPages := (total + int(req.GetPageSize()) - 1) / int(req.GetPageSize())
	return &analytics.GetProductEventsResponse{
		Events:     productEvents,
		TotalCount: utils.ToInt32(total),
		Page:       req.GetPage(),
		TotalPages: utils.ToInt32(totalPages),
	}, nil
}

func (s *Service) GetReport(ctx context.Context, req *analytics.GetReportRequest) (*analytics.GetReportResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	// For demo: support a report_id "event_count_by_type" with parameter "event_type"
	if req.ReportId == "event_count_by_type" {
		eventType := ""
		if req.Parameters != nil {
			if v, ok := req.Parameters["event_type"]; ok {
				eventType = v
			}
		}
		// DB-backed event counting by type
		count, err := s.repo.CountEventsByType(ctx, eventType)
		if err != nil {
			errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to count events by type", err)
			errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
			return nil, err
		}
		// Build metadata-like result
		result := map[string]interface{}{
			"report_id":  req.ReportId,
			"parameters": req.Parameters,
			"count":      count,
			"metadata": map[string]interface{}{
				"service_specific": map[string]interface{}{
					"analytics": map[string]interface{}{
						"event_type": eventType,
					},
				},
			},
		}
		data, err := json.Marshal(result)
		if err != nil {
			s.log.Error("Failed to marshal report result", zap.Error(err))
			errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to marshal result", err)
			errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		report := &analytics.Report{
			Id:          req.ReportId,
			Name:        "Event Count by Type",
			Description: "Counts analytics events by event_type parameter.",
			Parameters:  req.Parameters,
			Data:        data,
			CreatedAt:   time.Now().Unix(),
		}
		success := graceful.WrapSuccess(ctx, codes.OK, "analytics report generated", &analytics.GetReportResponse{Report: report}, nil)
		success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
			Log:          s.log,
			Cache:        s.Cache,
			CacheKey:     req.ReportId,
			CacheValue:   report.Data,
			CacheTTL:     10 * time.Minute,
			Metadata:     nil,
			EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
			EventEnabled: s.eventEnabled,
			EventType:    "analytics.report_generated",
			EventID:      req.ReportId,
			PatternType:  "analytics_event",
			PatternID:    req.ReportId,
			PatternMeta:  nil,
		})
		return &analytics.GetReportResponse{Report: report}, nil
	}
	// Default dummy report
	dummyData, err := json.Marshal(map[string]interface{}{
		"report_id":  req.ReportId,
		"parameters": map[string]string{"example": "value"},
		"data":       "dummy data",
	})
	if err != nil {
		s.log.Error("Failed to marshal dummy report data", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to marshal dummy report data", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
		return nil, fmt.Errorf("failed to marshal dummy report data: %w", err)
	}
	report := &analytics.Report{
		Id:          req.ReportId,
		Name:        "Dummy Report",
		Description: "This is a dummy analytics report.",
		Parameters:  map[string]string{"example": "value"},
		Data:        dummyData,
		CreatedAt:   time.Now().Unix(),
	}
	success := graceful.WrapSuccess(ctx, codes.OK, "analytics report generated", &analytics.GetReportResponse{Report: report}, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     req.ReportId,
		CacheValue:   report.Data,
		CacheTTL:     10 * time.Minute,
		Metadata:     nil,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
		EventType:    "analytics.report_generated",
		EventID:      req.ReportId,
		PatternType:  "analytics_event",
		PatternID:    req.ReportId,
		PatternMeta:  nil,
	})
	return &analytics.GetReportResponse{Report: report}, nil
}

func (s *Service) ListReports(ctx context.Context, req *analytics.ListReportsRequest) (*analytics.ListReportsResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 10
	}

	// Use the repository to fetch reports instead of returning dummy data.
	reports, total, err := s.repo.ListReports(ctx, page, pageSize)
	if err != nil {
		s.log.Error("failed to list reports", zap.Error(err))
		errCtx := graceful.WrapErr(ctx, codes.Internal, "failed to list reports", err)
		errCtx.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log, Context: ctx})
		return nil, status.Error(codes.Internal, "failed to list reports")
	}

	totalPages := (total + pageSize - 1) / pageSize

	resp := &analytics.ListReportsResponse{
		Reports:    reports,
		TotalCount: utils.ToInt32(total),
		Page:       utils.ToInt32(page),
		TotalPages: utils.ToInt32(totalPages),
	}

	success := graceful.WrapSuccess(ctx, codes.OK, "analytics reports listed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
		Log:          s.log,
		Cache:        s.Cache,
		CacheKey:     fmt.Sprintf("reports_list_%d_%d", page, pageSize),
		CacheValue:   reports,
		CacheTTL:     10 * time.Minute,
		Metadata:     nil,
		EventEmitter: &EventEmitterAdapter{Emitter: s.eventEmitter},
		EventEnabled: s.eventEnabled,
		EventType:    "analytics.reports_listed",
		EventID:      "",
		PatternType:  "analytics_event",
		PatternID:    "",
		PatternMeta:  nil,
	})
	return resp, nil
}
