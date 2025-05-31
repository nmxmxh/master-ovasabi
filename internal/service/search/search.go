package search

import (
	context "context"
	"errors"
	"time"

	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
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
		err = graceful.WrapErr(ctx, codes.Internal, "search failed", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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
	success := graceful.WrapSuccess(ctx, codes.OK, "search completed", resp, nil)
	success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{Log: s.log, Cache: s.Cache, CacheKey: "search:" + query, CacheValue: resp, CacheTTL: 5 * time.Minute, Metadata: meta, EventEmitter: s.eventEmitter, EventEnabled: s.eventEnabled, EventType: "search.completed", EventID: query, PatternType: "search", PatternID: query, PatternMeta: meta})
	return resp, nil
}
