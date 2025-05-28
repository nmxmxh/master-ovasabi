package search

import (
	context "context"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	events "github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Add missing type definitions if not imported
// type Repository interface { ... } // If not already defined elsewhere
// type EventEmitter interface { ... } // If not already defined elsewhere

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
	meta := req.GetMetadata()
	types := req.GetTypes()
	if len(types) == 0 {
		types = []string{"content"} // default to content if not specified
	}

	results, total, err := s.repo.SearchAllEntities(ctx, types, query, meta, req.GetCampaignId(), page, pageSize)
	if err != nil {
		s.log.Error("Search failed", zap.Error(err))
		if s.eventEnabled && s.eventEmitter != nil {
			errMeta := &commonpb.Metadata{
				ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": err.Error(), "query": req.Query}),
				Tags:            []string{},
				Features:        []string{},
			}
			_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "search", "search.failed", "", errMeta, zap.Error(err))
			if !ok {
				s.log.Warn("Failed to emit search.failed event")
			}
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
	if s.eventEnabled && s.eventEmitter != nil {
		successMeta := &commonpb.Metadata{
			ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"query": req.Query}),
			Tags:            []string{},
			Features:        []string{},
		}
		_, ok := events.EmitCallbackEvent(ctx, s.eventEmitter, s.log, s.Cache, "search", "search.completed", "", successMeta, zap.String("query", req.Query))
		if !ok {
			s.log.Warn("Failed to emit search.completed event")
		}
	}
	return resp, nil
}

// Optionally, implement Suggest and other endpoints as needed.
