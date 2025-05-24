package search

import (
	context "context"

	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	events "github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	searchpb.UnimplementedSearchServiceServer
	log          *zap.Logger
	repo         *Repository
	Cache        *redis.Cache
	eventEmitter EventEmitter
	eventEnabled bool
}

// NewService creates a new SearchService instance with event bus support.
func NewService(log *zap.Logger, repo *Repository, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) searchpb.SearchServiceServer {
	return &Service{
		log:          log,
		repo:         repo,
		Cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
}

// Search implements robust multi-entity, FTS, and metadata filtering search.
// Supports searching across multiple entity types as specified in req.Types.
func (s *Service) Search(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
	query := req.GetQuery()
	page := int(req.GetPageNumber())
	pageSize := int(req.GetPageSize())
	metadata := req.GetMetadata()
	types := req.GetTypes()
	if len(types) == 0 {
		types = []string{"content"} // default to content if not specified
	}

	results, total, err := s.repo.SearchAllEntities(ctx, types, query, metadata, req.GetCampaignId(), page, pageSize)
	if err != nil {
		s.log.Error("Search failed", zap.Error(err))
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			// Optionally, enrich metadata with error details
			// Canonical metadata pattern: use commonpb.Metadata for error context
			// Here, we emit the event with the request metadata
			events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "search.failed", "", metadata)
		}
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}
	protos := make([]*searchpb.SearchResult, 0, len(results))
	for _, r := range results {
		protos = append(protos, &searchpb.SearchResult{
			Id:         r.ID,
			EntityType: r.EntityType,
			Score:      float32(r.Score),
			Metadata:   r.Metadata,
		})
	}
	resp := &searchpb.SearchResponse{
		Results:    protos,
		Total:      utils.ToInt32(total),
		PageNumber: utils.ToInt32(page),
		PageSize:   utils.ToInt32(pageSize),
	}
	// Emit search.performed event after successful search
	if s.eventEnabled && s.eventEmitter != nil {
		events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "search.performed", "", metadata)
	}
	return resp, nil
}

// Optionally, implement Suggest and other endpoints as needed.
