package broadcast

import (
	"context"
	"errors"
	"math"
	"sync"

	broadcastpb "github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v1"
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
func (s *ServiceImpl) BroadcastAction(_ context.Context, _ *broadcastpb.BroadcastActionRequest) (*broadcastpb.BroadcastActionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "BroadcastAction not yet implemented")
}

// SubscribeToActions implements the BroadcastAction streaming RPC method.
func (s *ServiceImpl) SubscribeToActions(_ *broadcastpb.SubscribeToActionsRequest, _ broadcastpb.BroadcastService_SubscribeToActionsServer) error {
	return status.Error(codes.Unimplemented, "SubscribeToActions not yet implemented")
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
	if b.ID > math.MaxInt32 || b.ID < math.MinInt32 {
		return nil, status.Error(codes.Internal, "broadcast ID out of int32 range")
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
		if b.ID > math.MaxInt32 || b.ID < math.MinInt32 {
			return nil, status.Error(codes.Internal, "broadcast ID out of int32 range")
		}
		resp.Broadcasts = append(resp.Broadcasts, &broadcastpb.Broadcast{
			Id: int32(b.ID),
			// Map other fields as needed
		})
	}
	return resp, nil
}

// nanoQ-style broadcaster for live asset chunks
// Subscribers receive []byte chunks; slow subscribers are dropped
// This can be moved to a shared package if needed

type AssetBroadcaster struct {
	subs map[string]chan []byte
	lock sync.RWMutex
}

func NewAssetBroadcaster() *AssetBroadcaster {
	return &AssetBroadcaster{
		subs: make(map[string]chan []byte),
	}
}

func (b *AssetBroadcaster) Subscribe(id string) <-chan []byte {
	b.lock.Lock()
	defer b.lock.Unlock()
	ch := make(chan []byte, 8) // buffer for burst
	b.subs[id] = ch
	return ch
}

func (b *AssetBroadcaster) Unsubscribe(id string) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if ch, ok := b.subs[id]; ok {
		close(ch)
		delete(b.subs, id)
	}
}

func (b *AssetBroadcaster) Publish(chunk []byte) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	for id, ch := range b.subs {
		select {
		case ch <- chunk:
			// delivered
		default:
			// drop if slow (nanoQ pattern)
			go b.Unsubscribe(id)
		}
	}
}

// Place after AssetBroadcaster definition.

// SubscribeToLiveAssetChunks streams live asset chunks to the client.
func (s *ServiceImpl) SubscribeToLiveAssetChunks(_ *broadcastpb.SubscribeToLiveAssetChunksRequest, _ broadcastpb.BroadcastService_SubscribeToLiveAssetChunksServer) error {
	return status.Error(codes.Unimplemented, "SubscribeToLiveAssetChunks not yet implemented")
}

// PublishLiveAssetChunk pushes a live asset chunk to all subscribers.
func (s *ServiceImpl) PublishLiveAssetChunk(_ context.Context, _ *broadcastpb.PublishLiveAssetChunkRequest) (*broadcastpb.PublishLiveAssetChunkResponse, error) {
	return nil, status.Error(codes.Unimplemented, "PublishLiveAssetChunk not yet implemented")
}

// CreateBroadcast implements the CreateBroadcast RPC method.
func (s *ServiceImpl) CreateBroadcast(_ context.Context, _ *broadcastpb.CreateBroadcastRequest) (*broadcastpb.CreateBroadcastResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CreateBroadcast not yet implemented")
}

// Compile-time check to ensure ServiceImpl implements BroadcastServiceServer.
var _ broadcastpb.BroadcastServiceServer = (*ServiceImpl)(nil)
