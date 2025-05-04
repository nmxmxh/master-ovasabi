package broadcast

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	broadcastpb "github.com/nmxmxh/master-ovasabi/api/protos/broadcast/v0"
	broadcastrepo "github.com/nmxmxh/master-ovasabi/internal/repository/broadcast"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
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

// Place after AssetBroadcaster definition
var (
	liveAssetBroadcasters = make(map[string]*AssetBroadcaster)
	broadcasterLock       sync.RWMutex
)

func getBroadcaster(assetID string) *AssetBroadcaster {
	broadcasterLock.Lock()
	defer broadcasterLock.Unlock()
	b, ok := liveAssetBroadcasters[assetID]
	if !ok {
		b = NewAssetBroadcaster()
		liveAssetBroadcasters[assetID] = b
	}
	return b
}

// SubscribeToLiveAssetChunks streams live asset chunks to the client
func (s *ServiceImpl) SubscribeToLiveAssetChunks(req *broadcastpb.SubscribeToLiveAssetChunksRequest, stream broadcastpb.BroadcastService_SubscribeToLiveAssetChunksServer) error {
	b := getBroadcaster(req.AssetId)
	clientID := uuid.New().String()
	ch := b.Subscribe(clientID)
	defer b.Unsubscribe(clientID)
	sequence := uint32(0)
	for chunk := range ch {
		msg := &broadcastpb.AssetChunk{
			UploadId: req.AssetId,
			Data:     chunk,
			Sequence: sequence,
		}
		if err := stream.Send(msg); err != nil {
			return err
		}
		sequence++
	}
	return nil
}

// PublishLiveAssetChunk pushes a live asset chunk to all subscribers
func (s *ServiceImpl) PublishLiveAssetChunk(ctx context.Context, chunk *broadcastpb.AssetChunk) (*emptypb.Empty, error) {
	b := getBroadcaster(chunk.UploadId)
	b.Publish(chunk.Data)
	return &emptypb.Empty{}, nil
}
