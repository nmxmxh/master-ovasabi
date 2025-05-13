package analyticsservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	analyticspb "github.com/nmxmxh/master-ovasabi/api/protos/analytics/v1"
	pattern "github.com/nmxmxh/master-ovasabi/internal/service/pattern"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Repository interface {
	TrackEvent(ctx context.Context, event *analyticspb.Event) error
	BatchTrackEvents(ctx context.Context, events []*analyticspb.Event) (int, int, error)
	GetUserEvents(ctx context.Context, userID string, page, pageSize int) ([]*analyticspb.Event, int, error)
	GetProductEvents(ctx context.Context, productID string, page, pageSize int) ([]*analyticspb.Event, int, error)
	GetReport(ctx context.Context, reportID string) (*analyticspb.Report, error)
	ListReports(ctx context.Context, page, pageSize int) ([]*analyticspb.Report, int, error)
}

type Service struct {
	analyticspb.UnimplementedAnalyticsServiceServer
	log   *zap.Logger
	repo  Repository
	Cache *redis.Cache
}

func NewAnalyticsService(log *zap.Logger, repo Repository) analyticspb.AnalyticsServiceServer {
	return &Service{
		log:  log,
		repo: repo,
	}
}

var _ analyticspb.AnalyticsServiceServer = (*Service)(nil)

func (s *Service) TrackEvent(ctx context.Context, req *analyticspb.TrackEventRequest) (*analyticspb.TrackEventResponse, error) {
	event := req.GetEvent()
	if event == nil || event.MasterId == "" {
		return nil, status.Error(codes.InvalidArgument, "event and master_id are required")
	}
	if err := s.repo.TrackEvent(ctx, event); err != nil {
		s.log.Error("failed to track event", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to track event")
	}
	if s.Cache != nil && event.Metadata != nil {
		err := pattern.CacheMetadata(ctx, s.log, s.Cache, "analytics_event", event.Id, event.Metadata, 10*time.Minute)
		if err != nil {
			s.log.Error("failed to cache metadata", zap.Error(err))
		}
	}
	err := pattern.RegisterSchedule(ctx, s.log, "analytics_event", event.Id, event.Metadata)
	if err != nil {
		s.log.Error("failed to register schedule", zap.Error(err))
	}
	err = pattern.EnrichKnowledgeGraph(ctx, s.log, "analytics_event", event.Id, event.Metadata)
	if err != nil {
		s.log.Error("failed to enrich knowledge graph", zap.Error(err))
	}
	err = pattern.RegisterWithNexus(ctx, s.log, "analytics_event", event.Metadata)
	if err != nil {
		s.log.Error("failed to register with nexus", zap.Error(err))
	}
	return &analyticspb.TrackEventResponse{Success: true}, nil
}

func (s *Service) BatchTrackEvents(ctx context.Context, req *analyticspb.BatchTrackEventsRequest) (*analyticspb.BatchTrackEventsResponse, error) {
	for _, event := range req.GetEvents() {
		if event.MasterId == "" {
			return nil, status.Error(codes.InvalidArgument, "all events must have master_id")
		}
	}
	success, fail, err := s.repo.BatchTrackEvents(ctx, req.GetEvents())
	if err != nil {
		s.log.Error("failed to batch track events", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to batch track events")
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
	return &analyticspb.BatchTrackEventsResponse{
		//nolint:gosec // overflow checked above
		SuccessCount: int32(success),
		FailureCount: int32(fail),
	}, nil
}

func (s *Service) GetUserEvents(ctx context.Context, req *analyticspb.GetUserEventsRequest) (*analyticspb.GetUserEventsResponse, error) {
	events, total, err := s.repo.GetUserEvents(ctx, req.GetUserId(), int(req.GetPage()), int(req.GetPageSize()))
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
		return nil, fmt.Errorf("total pages overflows int32 (final check)")
	}
	if total > int(^int32(0)) || total < 0 {
		return nil, fmt.Errorf("total count overflows int32 (final check 2)")
	}
	if totalPages > int(^int32(0)) || totalPages < 0 {
		return nil, fmt.Errorf("totalPages overflows int32 (final check 2)")
	}
	if totalPages > int(^int32(0)) || totalPages < 0 {
		return nil, fmt.Errorf("totalPages overflows int32 (final check 3)")
	}
	return &analyticspb.GetUserEventsResponse{
		Events: events,
		//nolint:gosec // overflow checked above
		TotalCount: int32(total),
		Page:       req.GetPage(),
		TotalPages: int32(totalPages),
	}, nil
}

func (s *Service) GetProductEvents(_ context.Context, _ *analyticspb.GetProductEventsRequest) (*analyticspb.GetProductEventsResponse, error) {
	// TODO: Fetch analytics events for a product
	// Pseudocode:
	// 1. Query events by product ID
	// 2. Return event list (each event includes metadata if present)
	return nil, errors.New("not implemented")
}

func (s *Service) GetReport(_ context.Context, _ *analyticspb.GetReportRequest) (*analyticspb.GetReportResponse, error) {
	// TODO: Generate analytics report
	// Pseudocode:
	// 1. Parse report params
	// 2. Aggregate data (each event/report includes metadata if present)
	// 3. Return report
	return nil, errors.New("not implemented")
}

func (s *Service) ListReports(_ context.Context, _ *analyticspb.ListReportsRequest) (*analyticspb.ListReportsResponse, error) {
	// TODO: List available analytics reports
	// Pseudocode:
	// 1. Query available reports (each report includes metadata if present)
	// 2. Return list
	return nil, errors.New("not implemented")
}
