package asset

import (
	"context"
	"sync"
	"time"

	assetpb "github.com/nmxmxh/master-ovasabi/api/protos/asset/v1"
	assetrepo "github.com/nmxmxh/master-ovasabi/internal/repository/asset"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
)

// AssetService defines the interface for asset operations
type AssetService interface {
	UploadLightAsset(ctx context.Context, req *assetpb.UploadLightAssetRequest) (*assetpb.UploadLightAssetResponse, error)
	StartHeavyAssetUpload(ctx context.Context, req *assetpb.StartHeavyAssetUploadRequest) (*assetpb.StartHeavyAssetUploadResponse, error)
	StreamAssetChunk(ctx context.Context, req *assetpb.StreamAssetChunkRequest) (*assetpb.StreamAssetChunkResponse, error)
	CompleteAssetUpload(ctx context.Context, req *assetpb.CompleteAssetUploadRequest) (*assetpb.CompleteAssetUploadResponse, error)
	GetAsset(ctx context.Context, req *assetpb.GetAssetRequest) (*assetpb.GetAssetResponse, error)
	StreamAssetContent(ctx context.Context, req *assetpb.StreamAssetContentRequest) (*assetpb.StreamAssetContentResponse, error)
	DeleteAsset(ctx context.Context, req *assetpb.DeleteAssetRequest) (*assetpb.DeleteAssetResponse, error)
	ListUserAssets(ctx context.Context, req *assetpb.ListUserAssetsRequest) (*assetpb.ListUserAssetsResponse, error)
	ListSystemAssets(ctx context.Context, req *assetpb.ListSystemAssetsRequest) (*assetpb.ListSystemAssetsResponse, error)
	SubscribeToUserAssets(ctx context.Context, req *assetpb.SubscribeToUserAssetsRequest) (*assetpb.SubscribeToUserAssetsResponse, error)
	SubscribeToSystemAssets(ctx context.Context, req *assetpb.SubscribeToSystemAssetsRequest) (*assetpb.SubscribeToSystemAssetsResponse, error)
	BroadcastSystemAsset(ctx context.Context, req *assetpb.BroadcastSystemAssetRequest) (*assetpb.BroadcastSystemAssetResponse, error)
}

const (
	// Upload size thresholds
	ultraLightThreshold = 100 * 1024      // 100KB - For tiny assets like icons
	lightThreshold      = 500 * 1024      // 500KB - For small assets
	mediumThreshold     = 5 * 1024 * 1024 // 5MB - For medium assets

	// Chunk sizes for different upload types
	smallChunkSize   = 256 * 1024  // 256KB chunks for medium assets
	largeChunkSize   = 1024 * 1024 // 1MB chunks for large assets
	defaultChunkSize = 512 * 1024  // 512KB default chunk size for streaming

	// Security and consistency
	maxRetries                = 3
	uploadTimeout             = 30 * time.Minute
	chunkTimeout              = 1 * time.Minute
	maxConcurrentUploadChunks = 4
)

// ServiceImpl implements the AssetService interface
type ServiceImpl struct {
	assetpb.UnimplementedAssetServiceServer
	log   *zap.Logger
	cache *redis.Cache
	repo  assetrepo.AssetRepository
}

// UploadMetadata stores upload session information
type UploadMetadata struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	Size           int64
	ChunksTotal    int
	ChunksReceived int
	Checksum       string
	StartedAt      time.Time
	LastUpdate     time.Time
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

// Global broadcaster instance (could be per-asset or per-session)
var liveAssetBroadcaster = NewAssetBroadcaster()

// BroadcastAssetChunk allows publishing a live asset chunk to all subscribers (for live streaming)
func (s *ServiceImpl) BroadcastAssetChunk(ctx context.Context, chunk []byte) {
	liveAssetBroadcaster.Publish(chunk)
	// Optionally, notify the broadcast service here
}

// InitService creates a new instance of AssetService
func InitService(log *zap.Logger, repo assetrepo.AssetRepository, cache *redis.Cache) *ServiceImpl {
	return &ServiceImpl{
		log:   log,
		repo:  repo,
		cache: cache,
	}
}

// validateMimeType checks if the MIME type is allowed
func (s *ServiceImpl) validateMimeType(mimeType string) bool {
	allowedTypes := map[string]bool{
		"model/gltf-binary":        true,
		"model/gltf+json":          true,
		"model/obj":                true,
		"model/fbx":                true,
		"application/octet-stream": true,
		// Add more allowed types as needed
	}
	return allowedTypes[mimeType]
}

// validateUploadSize checks if the size is within allowed limits
func (s *ServiceImpl) validateUploadSize(size int64) error {
	maxSize := int64(100 * 1024 * 1024) // 100MB max
	if size <= 0 {
		return status.Error(codes.InvalidArgument, "size must be positive")
	}
	if size > maxSize {
		return status.Errorf(codes.InvalidArgument, "size exceeds maximum allowed (%d bytes)", maxSize)
	}
	return nil
}

// UploadLightAsset handles small asset uploads (< 500KB)
func (s *ServiceImpl) UploadLightAsset(ctx context.Context, req *assetpb.UploadLightAssetRequest) (*assetpb.UploadLightAssetResponse, error) {
	// v1 proto has no fields; add logic when fields are defined
	// TODO: Implement light asset upload when proto fields are available
	return &assetpb.UploadLightAssetResponse{}, nil
}

// StartHeavyAssetUpload initiates a chunked upload for large assets
func (s *ServiceImpl) StartHeavyAssetUpload(ctx context.Context, req *assetpb.StartHeavyAssetUploadRequest) (*assetpb.StartHeavyAssetUploadResponse, error) {
	// Validate request
	if !s.validateMimeType(req.MimeType) {
		return nil, status.Error(codes.InvalidArgument, "unsupported MIME type")
	}

	if err := s.validateUploadSize(req.Size); err != nil {
		return nil, err
	}

	if req.Size <= ultraLightThreshold {
		return nil, status.Error(codes.InvalidArgument, "asset too small for heavy upload, use light upload instead")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	// Calculate number of chunks based on size
	chunkSize := smallChunkSize
	if req.Size > mediumThreshold {
		chunkSize = largeChunkSize
	}
	chunksTotal := (req.Size + int64(chunkSize) - 1) / int64(chunkSize)

	assetID := uuid.New()
	metadata := &UploadMetadata{
		ID:             assetID,
		UserID:         userID,
		Size:           req.Size,
		ChunksTotal:    int(chunksTotal),
		ChunksReceived: 0,
		StartedAt:      time.Now(),
		LastUpdate:     time.Now(),
	}

	// Store upload metadata in cache
	if err := s.cache.Set(ctx, "upload_metadata", assetID.String(), metadata, uploadTimeout); err != nil {
		s.log.Error("failed to store upload metadata",
			zap.Error(err),
			zap.String("uploadId", assetID.String()),
		)
		return nil, status.Error(codes.Internal, "failed to initialize upload")
	}

	asset := &assetrepo.AssetModel{
		ID:        assetID,
		UserID:    userID,
		Type:      assetrepo.StorageTypeHeavy,
		Name:      req.Name,
		MimeType:  req.MimeType,
		Size:      req.Size,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateAsset(ctx, asset); err != nil {
		s.log.Error("failed to create heavy asset",
			zap.Error(err),
			zap.String("uploadId", assetID.String()),
		)
		return nil, status.Error(codes.Internal, "failed to initialize upload")
	}

	return &assetpb.StartHeavyAssetUploadResponse{
		UploadId:    assetID.String(),
		ChunkSize:   int32(chunkSize),
		ChunksTotal: int32(chunksTotal),
	}, nil
}

// StreamAssetChunk handles streaming chunks for heavy asset uploads
func (s *ServiceImpl) StreamAssetChunk(ctx context.Context, req *assetpb.StreamAssetChunkRequest) (*assetpb.StreamAssetChunkResponse, error) {
	// v1 proto has no fields; add logic when fields are defined
	// TODO: Implement streaming asset chunk when proto fields are available
	return &assetpb.StreamAssetChunkResponse{}, nil
}

// CompleteAssetUpload finalizes a heavy asset upload
func (s *ServiceImpl) CompleteAssetUpload(ctx context.Context, req *assetpb.CompleteAssetUploadRequest) (*assetpb.CompleteAssetUploadResponse, error) {
	// This method is removed as per the instructions
	return nil, nil
}

// GetAsset retrieves an asset by ID
func (s *ServiceImpl) GetAsset(ctx context.Context, req *assetpb.GetAssetRequest) (*assetpb.GetAssetResponse, error) {
	// This method is removed as per the instructions
	return nil, nil
}

// StreamAssetContent streams the content of a stored asset from R2 in chunks via gRPC
func (s *ServiceImpl) StreamAssetContent(ctx context.Context, req *assetpb.StreamAssetContentRequest) (*assetpb.StreamAssetContentResponse, error) {
	// v1 proto has no fields; add logic when fields are defined
	// TODO: Implement stream asset content when proto fields are available
	return &assetpb.StreamAssetContentResponse{}, nil
}

// DeleteAsset deletes an asset
func (s *ServiceImpl) DeleteAsset(ctx context.Context, req *assetpb.DeleteAssetRequest) (*assetpb.DeleteAssetResponse, error) {
	// Implementation needed
	return nil, nil
}

// ListUserAssets lists assets for a user with pagination
func (s *ServiceImpl) ListUserAssets(ctx context.Context, req *assetpb.ListUserAssetsRequest) (*assetpb.ListUserAssetsResponse, error) {
	// This method is removed as per the instructions
	return nil, nil
}

// ListSystemAssets lists system assets with pagination
func (s *ServiceImpl) ListSystemAssets(ctx context.Context, req *assetpb.ListSystemAssetsRequest) (*assetpb.ListSystemAssetsResponse, error) {
	// This method is removed as per the instructions
	return nil, nil
}

// SubscribeToUserAssets subscribes to user assets
func (s *ServiceImpl) SubscribeToUserAssets(ctx context.Context, req *assetpb.SubscribeToUserAssetsRequest) (*assetpb.SubscribeToUserAssetsResponse, error) {
	// Implementation needed
	return nil, nil
}

// SubscribeToSystemAssets subscribes to system assets
func (s *ServiceImpl) SubscribeToSystemAssets(ctx context.Context, req *assetpb.SubscribeToSystemAssetsRequest) (*assetpb.SubscribeToSystemAssetsResponse, error) {
	// Implementation needed
	return nil, nil
}

// BroadcastSystemAsset broadcasts a system asset
func (s *ServiceImpl) BroadcastSystemAsset(ctx context.Context, req *assetpb.BroadcastSystemAssetRequest) (*assetpb.BroadcastSystemAssetResponse, error) {
	// Implementation needed
	return nil, nil
}

// Compile-time check
var _ assetpb.AssetServiceServer = (*ServiceImpl)(nil)
