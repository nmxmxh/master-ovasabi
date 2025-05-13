package search

import (
	context "context"

	searchpb "github.com/nmxmxh/master-ovasabi/api/protos/search/v1"
	"github.com/nmxmxh/master-ovasabi/internal/repository/search"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	searchpb.UnimplementedSearchServiceServer
	log   *zap.Logger
	repo  *search.Repository
	Cache *redis.Cache
}

func NewService(log *zap.Logger, repo *search.Repository, cache *redis.Cache) searchpb.SearchServiceServer {
	return &Service{
		log:   log,
		repo:  repo,
		Cache: cache,
	}
}

func (s *Service) SearchEntities(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
	results, total, err := s.repo.SearchEntities(
		ctx,
		req.GetEntityType(),
		req.GetQuery(),
		req.GetMasterId(),
		nil, // fields (not yet exposed in proto)
		nil, // metadata (not yet exposed in proto)
		int(req.GetPage()),
		int(req.GetPageSize()),
		false, // fuzzy (not yet exposed in proto)
		"",    // language (not yet exposed in proto)
	)
	if err != nil {
		s.log.Error("SearchEntities failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}
	if total > int(^int32(0)) || total < 0 {
		return nil, status.Errorf(codes.Internal, "total overflows int32")
	}
	if total > int(^int32(0)) || total < 0 {
		return nil, status.Errorf(codes.Internal, "total overflows int32 (final check)")
	}
	protos := make([]*searchpb.SearchResult, 0, len(results))
	for _, r := range results {
		protos = append(protos, &searchpb.SearchResult{
			Id:         r.ID,
			MasterId:   r.MasterID,
			EntityType: r.EntityType,
			Title:      r.Title,
			Snippet:    r.Snippet,
			Metadata:   r.Metadata,
			Score:      r.Score,
		})
	}
	if total > int(^int32(0)) || total < 0 {
		return nil, status.Errorf(codes.Internal, "total overflows int32 (final check 2)")
	}
	return &searchpb.SearchResponse{
		Results: protos,
		Total:   int32(total),
	}, nil
}
