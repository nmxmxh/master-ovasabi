package media

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	mediarepo "github.com/nmxmxh/master-ovasabi/internal/repository/media"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Service defines the interface for asset operations.
type Service interface {
	UploadLightMedia(ctx context.Context, req *mediapb.UploadLightMediaRequest) (*mediapb.UploadLightMediaResponse, error)
	StartHeavyMediaUpload(ctx context.Context, req *mediapb.StartHeavyMediaUploadRequest) (*mediapb.StartHeavyMediaUploadResponse, error)
	StreamMediaChunk(ctx context.Context, req *mediapb.StreamMediaChunkRequest) (*mediapb.StreamMediaChunkResponse, error)
	CompleteMediaUpload(ctx context.Context, req *mediapb.CompleteMediaUploadRequest) (*mediapb.CompleteMediaUploadResponse, error)
	GetMedia(ctx context.Context, req *mediapb.GetMediaRequest) (*mediapb.GetMediaResponse, error)
	StreamMediaContent(ctx context.Context, req *mediapb.StreamMediaContentRequest) (*mediapb.StreamMediaContentResponse, error)
	DeleteMedia(ctx context.Context, req *mediapb.DeleteMediaRequest) (*mediapb.DeleteMediaResponse, error)
	ListUserMedia(ctx context.Context, req *mediapb.ListUserMediaRequest) (*mediapb.ListUserMediaResponse, error)
	ListSystemMedia(ctx context.Context, req *mediapb.ListSystemMediaRequest) (*mediapb.ListSystemMediaResponse, error)
	SubscribeToUserMedia(ctx context.Context, req *mediapb.SubscribeToUserMediaRequest) (*mediapb.SubscribeToUserMediaResponse, error)
	SubscribeToSystemMedia(ctx context.Context, req *mediapb.SubscribeToSystemMediaRequest) (*mediapb.SubscribeToSystemMediaResponse, error)
	BroadcastSystemMedia(ctx context.Context, req *mediapb.BroadcastSystemMediaRequest) (*mediapb.BroadcastSystemMediaResponse, error)
}

const (
	// Upload size thresholds.
	ultraLightThreshold = 100 * 1024      // 100KB - For tiny assets like icons
	lightThreshold      = 500 * 1024      // 500KB - For small assets
	mediumThreshold     = 5 * 1024 * 1024 // 5MB - For medium assets

	// Chunk sizes for different upload types.
	smallChunkSize   = 256 * 1024  // 256KB chunks for medium assets
	largeChunkSize   = 1024 * 1024 // 1MB chunks for large assets
	defaultChunkSize = 512 * 1024  // 512KB default chunk size for streaming

	// Security and consistency.
	maxRetries                = 3
	uploadTimeout             = 30 * time.Minute
	chunkTimeout              = 1 * time.Minute
	maxConcurrentUploadChunks = 4
)

// ServiceImpl implements the Service interface.
type ServiceImpl struct {
	mediapb.UnimplementedMediaServiceServer
	log   *zap.Logger
	cache *redis.Cache
	repo  mediarepo.Repository
}

// UploadMetadata stores upload session information.
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

// Broadcaster struct.
type Broadcaster struct {
	subs map[string]chan []byte
	lock sync.RWMutex
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subs: make(map[string]chan []byte),
	}
}

func (b *Broadcaster) Subscribe(id string) <-chan []byte {
	b.lock.Lock()
	defer b.lock.Unlock()
	ch := make(chan []byte, 8) // buffer for burst
	b.subs[id] = ch
	return ch
}

func (b *Broadcaster) Unsubscribe(id string) {
	b.lock.Lock()
	defer b.lock.Unlock()
	if ch, ok := b.subs[id]; ok {
		close(ch)
		delete(b.subs, id)
	}
}

func (b *Broadcaster) Publish(chunk []byte) {
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

// Global broadcaster instance (could be per-asset or per-session).
var liveAssetBroadcaster = NewBroadcaster()

// BroadcastAssetChunk allows publishing a live asset chunk to all subscribers (for live streaming).
func (s *ServiceImpl) BroadcastAssetChunk(_ context.Context, chunk []byte) {
	liveAssetBroadcaster.Publish(chunk)
	// Optionally, notify the broadcast service here
}

// InitService creates a new instance of Service.
func InitService(log *zap.Logger, repo mediarepo.Repository, cache *redis.Cache) *ServiceImpl {
	return &ServiceImpl{
		log:   log,
		repo:  repo,
		cache: cache,
	}
}

// validateMimeType checks if the MIME type is allowed.
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

// validateUploadSize checks if the size is within allowed limits.
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

// UploadLightMedia handles small media uploads (< 500KB).
func (s *ServiceImpl) UploadLightMedia(_ context.Context, _ *mediapb.UploadLightMediaRequest) (*mediapb.UploadLightMediaResponse, error) {
	// TODO: Implement light media upload when proto fields are available
	return &mediapb.UploadLightMediaResponse{}, nil
}

// StartHeavyMediaUpload initiates a chunked upload for large media.
func (s *ServiceImpl) StartHeavyMediaUpload(ctx context.Context, req *mediapb.StartHeavyMediaUploadRequest) (*mediapb.StartHeavyMediaUploadResponse, error) {
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

	if chunkSize > math.MaxInt32 || chunkSize < math.MinInt32 {
		return nil, status.Error(codes.Internal, "chunk size out of int32 range")
	}
	if chunksTotal > math.MaxInt32 || chunksTotal < 0 {
		return nil, status.Error(codes.Internal, "chunks total out of int32 range")
	}

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

	asset := &mediarepo.Model{
		ID:        assetID,
		UserID:    userID,
		Type:      mediarepo.StorageTypeHeavy,
		Name:      req.Name,
		MimeType:  req.MimeType,
		Size:      req.Size,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateMedia(ctx, asset); err != nil {
		s.log.Error("failed to create heavy asset",
			zap.Error(err),
			zap.String("uploadId", assetID.String()),
		)
		return nil, status.Error(codes.Internal, "failed to initialize upload")
	}

	return &mediapb.StartHeavyMediaUploadResponse{
		UploadId:    assetID.String(),
		ChunkSize:   int32(chunkSize),   //nolint:gosec // safe: checked range above
		ChunksTotal: int32(chunksTotal), //nolint:gosec // safe: checked range above
	}, nil
}

// StreamMediaChunk handles streaming chunks for heavy media uploads.
func (s *ServiceImpl) StreamMediaChunk(_ context.Context, _ *mediapb.StreamMediaChunkRequest) (*mediapb.StreamMediaChunkResponse, error) {
	// TODO: Implement streaming media chunk when proto fields are available
	return &mediapb.StreamMediaChunkResponse{}, nil
}

// CompleteMediaUpload finalizes a heavy media upload.
func (s *ServiceImpl) CompleteMediaUpload(_ context.Context, _ *mediapb.CompleteMediaUploadRequest) (*mediapb.CompleteMediaUploadResponse, error) {
	// TODO: Implement CompleteMediaUpload
	// Pseudocode:
	// 1. Validate upload session and user permissions
	// 2. Verify all chunks received and integrity
	// 3. Assemble media from chunks and store in object storage
	// 4. Update media status in DB
	return &mediapb.CompleteMediaUploadResponse{}, nil
}

// GetMedia retrieves a media file.
func (s *ServiceImpl) GetMedia(ctx context.Context, req *mediapb.GetMediaRequest) (*mediapb.GetMediaResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid media ID")
	}
	media, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		if errors.Is(err, mediarepo.ErrMediaNotFound) {
			return nil, status.Error(codes.NotFound, "media not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get media: %v", err)
	}
	return &mediapb.GetMediaResponse{
		Media: &mediapb.Media{
			Id:        media.ID.String(),
			MasterId:  media.MasterID,
			UserId:    media.UserID.String(),
			Type:      mediapb.MediaType(mediapb.MediaType_value[string(media.Type)]),
			Name:      media.Name,
			MimeType:  media.MimeType,
			Size:      media.Size,
			Url:       media.URL,
			IsSystem:  media.IsSystem,
			CreatedAt: timestamppb.New(media.CreatedAt),
			UpdatedAt: timestamppb.New(media.UpdatedAt),
			Metadata:  media.Metadata,
		},
		Status: "ok",
	}, nil
}

// StreamMediaContent streams the content of a media file.
func (s *ServiceImpl) StreamMediaContent(_ context.Context, _ *mediapb.StreamMediaContentRequest) (*mediapb.StreamMediaContentResponse, error) {
	// TODO: Implement StreamMediaContent
	return &mediapb.StreamMediaContentResponse{}, nil
}

// DeleteMedia deletes a media file.
func (s *ServiceImpl) DeleteMedia(_ context.Context, _ *mediapb.DeleteMediaRequest) (*mediapb.DeleteMediaResponse, error) {
	// TODO: Implement DeleteMedia
	return &mediapb.DeleteMediaResponse{}, nil
}

// ListUserMedia lists user media files.
func (s *ServiceImpl) ListUserMedia(_ context.Context, _ *mediapb.ListUserMediaRequest) (*mediapb.ListUserMediaResponse, error) {
	// TODO: Implement ListUserMedia
	return &mediapb.ListUserMediaResponse{}, nil
}

// ListSystemMedia lists system media files.
func (s *ServiceImpl) ListSystemMedia(_ context.Context, _ *mediapb.ListSystemMediaRequest) (*mediapb.ListSystemMediaResponse, error) {
	// TODO: Implement ListSystemMedia
	return &mediapb.ListSystemMediaResponse{}, nil
}

// SubscribeToUserMedia subscribes to user media updates.
func (s *ServiceImpl) SubscribeToUserMedia(_ context.Context, _ *mediapb.SubscribeToUserMediaRequest) (*mediapb.SubscribeToUserMediaResponse, error) {
	// TODO: Implement SubscribeToUserMedia
	return &mediapb.SubscribeToUserMediaResponse{}, nil
}

// SubscribeToSystemMedia subscribes to system media updates.
func (s *ServiceImpl) SubscribeToSystemMedia(_ context.Context, _ *mediapb.SubscribeToSystemMediaRequest) (*mediapb.SubscribeToSystemMediaResponse, error) {
	// TODO: Implement SubscribeToSystemMedia
	return &mediapb.SubscribeToSystemMediaResponse{}, nil
}

// BroadcastSystemMedia broadcasts a system media file.
func (s *ServiceImpl) BroadcastSystemMedia(_ context.Context, _ *mediapb.BroadcastSystemMediaRequest) (*mediapb.BroadcastSystemMediaResponse, error) {
	// TODO: Implement BroadcastSystemMedia
	return &mediapb.BroadcastSystemMediaResponse{}, nil
}

// Compile-time check.
var _ mediapb.MediaServiceServer = (*ServiceImpl)(nil)

// TODO: Implement UploadAsset
// Pseudocode:
// 1. Authenticate user (via User/Auth service)
// 2. Validate asset metadata and permissions (Security service)
// 3. Store asset in object storage (e.g., S3)
// 4. Save asset metadata in DB
// 5. Optionally, trigger Babel for i18n metadata
// 6. Register asset in Nexus for orchestration
// 7. Return asset details

// TODO: Implement StreamAssetChunk
// Pseudocode:
// 1. Authenticate and authorize user
// 2. For each chunk received:
//    a. Validate chunk order and integrity
//    b. Store chunk in object storage
//    c. Update asset status in DB
//    d. Optionally, broadcast chunk via Broadcast service
// 3. On completion, mark asset as fully uploaded
// 4. Register upload event in Nexus
