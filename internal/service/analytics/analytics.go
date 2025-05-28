// Analytics Service (GDPR-Compliant, Extensible)
// ----------------------------------------------
// Implements the canonical analytics service for event capture, enrichment, and listing.
// - Uses robust, versioned, and GDPR-compliant metadata (see metadata.go).
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
	"encoding/json"
	"fmt"
	"sync"
	"time"

	analytics "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	events "github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Event struct {
	ID        string
	Timestamp time.Time
	Metadata  *commonpb.Metadata
}

type RepositoryItf interface {
	TrackEvent(ctx context.Context, event *analytics.Event) error
	BatchTrackEvents(ctx context.Context, events []*analytics.Event) (int, int, error)
	GetUserEvents(ctx context.Context, userID string, campaignID string, page, pageSize int) ([]*analytics.Event, int, error)
	GetProductEvents(ctx context.Context, productID string, campaignID string, page, pageSize int) ([]*analytics.Event, int, error)
	GetReport(ctx context.Context, reportID string) (*analytics.Report, error)
	ListReports(ctx context.Context, page, pageSize int) ([]*analytics.Report, int, error)
}

type Service struct {
	analytics.UnimplementedAnalyticsServiceServer
	log             *zap.Logger
	repo            *Repository
	Cache           *redis.Cache
	mu              sync.Mutex
	analyticsEvents map[string]*Event // in-memory event store (replace with DB)
	eventEmitter    EventEmitter
	eventEnabled    bool
}

func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) analytics.AnalyticsServiceServer {
	return &Service{
		log:             log,
		repo:            repo,
		Cache:           cache,
		analyticsEvents: make(map[string]*Event),
		eventEmitter:    eventEmitter,
		eventEnabled:    eventEnabled,
	}
}

var _ analytics.AnalyticsServiceServer = (*Service)(nil)

// Define a package-level error for event not found.
var ErrEventNotFound = status.Error(codes.NotFound, "analytics event not found")

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
		nil, // serviceSpecific extensions
	)
	if err != nil {
		s.log.Error("Failed to build analytics metadata", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			// Emit failed event with error metadata
			_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.event_capture_failed", "", meta, zap.Error(err))
			if !ok {
				s.log.Warn("Failed to emit workflow step event", zap.Error(err))
			}
		}
		return nil, err
	}
	eventID := generateEventID()
	event := &Event{
		ID:        eventID,
		Timestamp: time.Now(),
		Metadata:  meta,
	}
	s.mu.Lock()
	s.analyticsEvents[eventID] = event
	s.mu.Unlock()
	s.log.Info("Captured analytics event", zap.String("event_id", eventID), zap.String("event_type", req.EventType))
	if s.eventEnabled && s.eventEmitter != nil {
		// Emit tracked event (success)
		_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.event_tracked", eventID, event.Metadata)
		if !ok {
			s.log.Warn("Failed to emit workflow step event")
		}
	}
	return &analytics.CaptureEventResponse{EventId: eventID}, nil
}

// ListEvents returns all captured analytics events (paginated in production).
func (s *Service) ListEvents(ctx context.Context, req *analytics.ListEventsRequest) (*analytics.ListEventsResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}
	// In the future, use req for pagination/filtering.
	s.mu.Lock()
	analyticsEvents := make([]*analytics.AnalyticsEvent, 0, len(s.analyticsEvents))
	for _, e := range s.analyticsEvents {
		analyticsEvents = append(analyticsEvents, &analytics.AnalyticsEvent{
			EventId:   e.ID,
			Timestamp: e.Timestamp.Unix(),
			Metadata:  e.Metadata,
		})
	}
	s.mu.Unlock()
	return &analytics.ListEventsResponse{Events: analyticsEvents}, nil
}

// EnrichEventMetadata allows for post-hoc enrichment of event metadata.
func (s *Service) EnrichEventMetadata(ctx context.Context, req *analytics.EnrichEventMetadataRequest) (*analytics.EnrichEventMetadataResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	s.mu.Lock()
	event, ok := s.analyticsEvents[req.EventId]
	s.mu.Unlock()
	if !ok {
		s.log.Error("Event not found for enrichment", zap.String("event_id", req.EventId))
		if s.eventEnabled && s.eventEmitter != nil {
			_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.event_enrich_failed", "", nil)
			if !ok {
				s.log.Warn("Failed to emit workflow step event")
			}
		}
		return nil, ErrEventNotFound
	}
	// Merge new fields into existing metadata (simple merge for demo)
	if event.Metadata != nil && event.Metadata.ServiceSpecific != nil && req.NewFields != nil {
		for k, v := range req.NewFields.Fields {
			event.Metadata.ServiceSpecific.Fields[k] = v
		}
	}
	s.log.Info("Enriched analytics event metadata", zap.String("event_id", req.EventId))
	if s.eventEnabled && s.eventEmitter != nil {
		_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.event_enriched", req.EventId, event.Metadata)
		if !ok {
			s.log.Warn("Failed to emit workflow step event")
		}
	}
	return &analytics.EnrichEventMetadataResponse{Success: true}, nil
}

// generateEventID generates a unique event ID (replace with UUID in production).
func generateEventID() string {
	return time.Now().Format("20060102T150405.000000000")
}

func (s *Service) TrackEvent(ctx context.Context, req *analytics.TrackEventRequest) (*analytics.TrackEventResponse, error) {
	event := req.GetEvent()
	if event == nil || event.MasterId == 0 {
		if s.eventEnabled && s.eventEmitter != nil {
			_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.event_track_failed", "", nil)
			if !ok {
				s.log.Warn("Failed to emit workflow step event")
			}
		}
		return nil, status.Error(codes.InvalidArgument, "event and master_id are required")
	}
	if err := s.repo.TrackEvent(ctx, event); err != nil {
		s.log.Error("failed to track event", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.event_track_failed", "", nil)
			if !ok {
				s.log.Warn("Failed to emit workflow step event")
			}
		}
		return nil, status.Error(codes.Internal, "failed to track event")
	}
	if s.eventEnabled && s.eventEmitter != nil {
		_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.event_tracked", event.Id, event.Metadata)
		if !ok {
			s.log.Warn("Failed to emit workflow step event")
		}
	}
	return &analytics.TrackEventResponse{Success: true}, nil
}

func (s *Service) BatchTrackEvents(ctx context.Context, req *analytics.BatchTrackEventsRequest) (*analytics.BatchTrackEventsResponse, error) {
	for _, event := range req.GetEvents() {
		if event.MasterId == 0 {
			if s.eventEnabled && s.eventEmitter != nil {
				_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.batch_track_failed", "", nil)
				if !ok {
					s.log.Warn("Failed to emit workflow step event")
				}
			}
			return nil, status.Error(codes.InvalidArgument, "all events must have master_id")
		}
	}
	success, fail, err := s.repo.BatchTrackEvents(ctx, req.GetEvents())
	if err != nil {
		s.log.Error("failed to batch track events", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.batch_track_failed", "", nil)
			if !ok {
				s.log.Warn("Failed to emit workflow step event")
			}
		}
		return nil, status.Error(codes.Internal, "failed to batch track events")
	}
	if s.eventEnabled && s.eventEmitter != nil {
		_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.batch_tracked", "", nil)
		if !ok {
			s.log.Warn("Failed to emit workflow step event")
		}
	}
	if success > int(^int32(0)) || success < 0 {
		return nil, fmt.Errorf("success count overflows int32")
	}
	if fail > int(^int32(0)) || fail < 0 {
		return nil, fmt.Errorf("failure count overflows int32")
	}
	if success > int(^int32(0)) || success < 0 {
		return nil, fmt.Errorf("success count overflows int32 (final check)")
	}
	if fail > int(^int32(0)) || fail < 0 {
		return nil, fmt.Errorf("failure count overflows int32 (final check)")
	}
	return &analytics.BatchTrackEventsResponse{
		SuccessCount: utils.ToInt32(success),
		FailureCount: utils.ToInt32(fail),
	}, nil
}

func (s *Service) GetUserEvents(ctx context.Context, req *analytics.GetUserEventsRequest) (*analytics.GetUserEventsResponse, error) {
	userEvents, total, err := s.repo.GetUserEvents(ctx, req.GetUserId(), req.GetCampaignId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		s.log.Error("failed to get user events", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get user events")
	}
	if total > int(^int32(0)) || total < 0 {
		return nil, fmt.Errorf("total count overflows int32")
	}
	totalPages := (total + int(req.GetPageSize()) - 1) / int(req.GetPageSize())
	if totalPages > int(^int32(0)) || totalPages < 0 {
		return nil, fmt.Errorf("total pages overflows int32")
	}
	if total > int(^int32(0)) || total < 0 {
		return nil, fmt.Errorf("total count overflows int32 (final check)")
	}
	if totalPages > int(^int32(0)) || totalPages < 0 {
		return nil, fmt.Errorf("totalPages overflows int32 (final check 2)")
	}
	if totalPages > int(^int32(0)) || totalPages < 0 {
		return nil, fmt.Errorf("totalPages overflows int32 (final check 3)")
	}
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
		s.mu.Lock()
		count := 0
		for _, e := range s.analyticsEvents {
			if e.Metadata != nil && e.Metadata.ServiceSpecific != nil {
				if analyticsFields, ok := e.Metadata.ServiceSpecific.Fields["analytics"]; ok {
					analyticsStruct := analyticsFields.GetStructValue()
					if analyticsStruct != nil {
						if v, ok := analyticsStruct.Fields["event_type"]; ok {
							if eventType == "" || v.GetStringValue() == eventType {
								count++
							}
						}
					}
				}
			}
		}
		s.mu.Unlock()
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
		if s.eventEnabled && s.eventEmitter != nil {
			_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.report_generated", req.ReportId, nil)
			if !ok {
				s.log.Warn("Failed to emit workflow step event")
			}
		}
		return &analytics.GetReportResponse{Report: report}, nil
	}
	// Default dummy report
	report := &analytics.Report{
		Id:          req.ReportId,
		Name:        "Dummy Report",
		Description: "This is a dummy analytics report.",
		Parameters:  map[string]string{"example": "value"},
		Data:        []byte("dummy data"),
		CreatedAt:   time.Now().Unix(),
	}
	if s.eventEnabled && s.eventEmitter != nil {
		_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "analytics_event", "analytics.report_generated", req.ReportId, nil)
		if !ok {
			s.log.Warn("Failed to emit workflow step event")
		}
	}
	return &analytics.GetReportResponse{Report: report}, nil
}

func (s *Service) ListReports(ctx context.Context, req *analytics.ListReportsRequest) (*analytics.ListReportsResponse, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	reports := []*analytics.Report{
		{
			Id:          "report1",
			Name:        "Report 1",
			Description: "First dummy report.",
			Parameters:  map[string]string{"foo": "bar"},
			Data:        []byte("data1"),
			CreatedAt:   time.Now().Unix(),
		},
		{
			Id:          "report2",
			Name:        "Report 2",
			Description: "Second dummy report.",
			Parameters:  map[string]string{"baz": "qux"},
			Data:        []byte("data2"),
			CreatedAt:   time.Now().Unix(),
		},
	}
	total := len(reports)
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize < 1 {
		pageSize = 10
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	paged := reports[start:end]
	totalPages := (total + pageSize - 1) / pageSize
	return &analytics.ListReportsResponse{
		Reports:    paged,
		TotalCount: utils.ToInt32(total),
		Page:       utils.ToInt32(page),
		TotalPages: utils.ToInt32(totalPages),
	}, nil
}
