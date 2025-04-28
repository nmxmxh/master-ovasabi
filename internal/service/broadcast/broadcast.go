package broadcast

import (
	"context"
	"errors"

	broadcastpb "github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v0"
	broadcastrepo "github.com/nmxmxh/master-ovasabi/internal/repository/broadcast"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ServiceImpl implements the BroadcastService interface.
type ServiceImpl struct {
	broadcastpb.UnimplementedBroadcastServiceServer
	log   *zap.Logger
	cache *redis.Cache
	repo  *broadcastrepo.BroadcastRepository
}

// NewService creates a new instance of BroadcastService.
func NewService(log *zap.Logger, repo *broadcastrepo.BroadcastRepository, cache *redis.Cache) *ServiceImpl {
	return &ServiceImpl{
		log:   log,
		repo:  repo,
		cache: cache,
	}
}

// BroadcastAction implements the BroadcastAction RPC method.
func (s *ServiceImpl) BroadcastAction(ctx context.Context, req *broadcastpb.BroadcastActionRequest) (*broadcastpb.BroadcastActionResponse, error) {
	// TODO: Implement a CreateAction method in the repository for this use case
	return nil, status.Error(codes.Unimplemented, "BroadcastAction repository integration not yet implemented")
}

// SubscribeToActions implements the SubscribeToActions streaming RPC method.
func (s *ServiceImpl) SubscribeToActions(req *broadcastpb.SubscribeRequest, stream broadcastpb.BroadcastService_SubscribeToActionsServer) error {
	// TODO: Implement repository method for listing recent actions
	return status.Error(codes.Unimplemented, "SubscribeToActions repository integration not yet implemented")
}

// GetBroadcast retrieves a specific broadcast by ID.
func (s *ServiceImpl) GetBroadcast(ctx context.Context, req *broadcastpb.GetBroadcastRequest) (*broadcastpb.GetBroadcastResponse, error) {
	b, err := s.repo.GetByID(ctx, int64(req.BroadcastId))
	if err != nil {
		if errors.Is(err, broadcastrepo.ErrBroadcastNotFound) {
			return nil, status.Error(codes.NotFound, "broadcast not found")
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	resp := &broadcastpb.Broadcast{
		Id: int32(b.ID),
		// Map other fields as needed
	}
	return &broadcastpb.GetBroadcastResponse{Broadcast: resp}, nil
}

// ListBroadcasts retrieves a list of broadcasts with pagination.
func (s *ServiceImpl) ListBroadcasts(ctx context.Context, req *broadcastpb.ListBroadcastsRequest) (*broadcastpb.ListBroadcastsResponse, error) {
	limit := 10
	if req.PageSize > 0 {
		limit = int(req.PageSize)
	}
	offset := int(req.Page) * limit
	broadcasts, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list broadcasts: %v", err)
	}
	resp := &broadcastpb.ListBroadcastsResponse{
		Broadcasts: make([]*broadcastpb.Broadcast, 0, len(broadcasts)),
	}
	for _, b := range broadcasts {
		resp.Broadcasts = append(resp.Broadcasts, &broadcastpb.Broadcast{
			Id: int32(b.ID),
			// Map other fields as needed
		})
	}
	return resp, nil
}
