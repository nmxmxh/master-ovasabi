/*
Media Service Redis Pub/Sub Pattern for Real-Time Events
-------------------------------------------------------

This service publishes media events to Redis Pub/Sub channels for real-time delivery via WebSocket or other event consumers.

Pattern:
- System-wide: channel 'media:events:system'
- Campaign-specific: channel 'media:events:campaign:{campaign_id}'
- User-specific: channel 'media:events:user:{user_id}'

Usage:
- On media upload, update, delete, or broadcast, publish to the appropriate channels based on available IDs.
- Enables targeted, scalable, real-time updates for system, campaign, or user.
*/
// MediaService: Azure-Optimized, Composable API
//
// Responsibilities:
//   - Asset ingestion (upload)
//   - Integration with Azure Media Services for encoding/streaming
//   - Expose composable HTTP APIs for upload, status, playback, thumbnail
//   - Update asset metadata (status, streaming URLs, etc.)
//   - Trigger notifications via NotificationService when streams are ready/live
//
// This is a stub implementation; fill in Azure SDK/REST integration as needed.

package media

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	azblob "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	blob "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	blockblob "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	events "github.com/nmxmxh/master-ovasabi/pkg/events"
	pkgmeta "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	structpb "google.golang.org/protobuf/types/known/structpb"
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
	log          *zap.Logger
	cache        *redis.Cache
	repo         *Repo
	eventEmitter EventEmitter
	eventEnabled bool
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

// Add a global in-memory map for upload sessions (for demo only).
var heavyUploadSessions = make(map[string]*UploadMetadata)

// In-memory pub/sub channels for demo (replace with real pub/sub or WebSocket in prod).
var (
	userMediaSubs   = make(map[string]chan *mediapb.Media)
	systemMediaSubs = make(map[string]chan *mediapb.Media)
)

// NewService constructs a new MediaServiceServer instance with event bus support.
func NewService(log *zap.Logger, repo *Repo, cache *redis.Cache, eventEmitter EventEmitter, eventEnabled bool) mediapb.MediaServiceServer {
	svc := &ServiceImpl{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
	}
	// Start Redis Pub/Sub listener for media events
	if cache != nil {
		go func() {
			ctx := context.Background()
			pubsub := cache.GetClient().Subscribe(ctx, "media:events:system")
			ch := pubsub.Channel()
			for msg := range ch {
				var mediaEvent struct {
					Type string          `json:"type"`
					Data json.RawMessage `json:"data"`
				}
				if err := json.Unmarshal([]byte(msg.Payload), &mediaEvent); err == nil {
					for _, ch := range systemMediaSubs {
						var m mediapb.Media
						if err := json.Unmarshal(mediaEvent.Data, &m); err == nil {
							select {
							case ch <- &m:
							default:
							}
						}
					}
				}
			}
		}()
	}
	return svc
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

// Add a helper type for io.ReadSeekCloser.
// readSeekCloser wraps a *bytes.Reader and implements io.ReadSeekCloser
// (io.ReadSeekCloser = io.Reader + io.Seeker + io.Closer).
type readSeekCloser struct {
	*bytes.Reader
}

func (r *readSeekCloser) Close() error { return nil }

// UploadLightMedia handles small media uploads (< 500KB).
func (s *ServiceImpl) UploadLightMedia(ctx context.Context, req *mediapb.UploadLightMediaRequest) (*mediapb.UploadLightMediaResponse, error) {
	// Parse and validate metadata
	var mediaMeta *Metadata
	if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
		mediaField, ok := req.Metadata.ServiceSpecific.Fields["media"]
		if ok && mediaField.GetKind() != nil {
			var err error
			mediaMeta, err = MetadataFromStruct(mediaField.GetStructValue())
			if err != nil {
				return nil, err
			}
		}
	}
	if mediaMeta == nil {
		mediaMeta = &Metadata{}
	}
	if mediaMeta.Versioning == nil {
		mediaMeta.Versioning = map[string]interface{}{
			"system_version":   "1.0.0",
			"service_version":  "1.0.0",
			"environment":      "prod",
			"last_migrated_at": time.Now().Format(time.RFC3339),
		}
	}

	// --- Azure Blob Storage Upload ---
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey := os.Getenv("AZURE_BLOB_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER")
	if accountName == "" || accountKey == "" || containerName == "" {
		return nil, status.Error(codes.Internal, "Azure Blob Storage config missing")
	}
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create Azure credential: %v", err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	blobName := uuid.New().String() + ".mp4"
	blockBlobClient, err := blockblob.NewClientWithSharedKeyCredential(serviceURL+containerName+"/"+blobName, cred, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create block blob client: %v", err)
	}
	// Upload file data (req.Data)
	reader := bytes.NewReader(req.Data)
	_, err = blockBlobClient.UploadStream(ctx, reader, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upload to Azure Blob: %v", err)
	}
	blobURL := serviceURL + containerName + "/" + blobName

	// --- ffprobe: Extract technical metadata ---
	ffprobePath := os.Getenv("FFPROBE_PATH")
	if ffprobePath == "" {
		ffprobePath = "ffprobe" // assume in PATH
	}
	tmpFile := "/tmp/" + blobName
	err = os.WriteFile(tmpFile, req.Data, 0o600)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to write temp file for ffprobe: %v", err)
	}
	cmd := exec.Command(ffprobePath, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", tmpFile)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ffprobe failed: %v", err)
	}
	var probe struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			BitRate   string `json:"bit_rate"`
			Duration  string `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			BitRate  string `json:"bit_rate"`
		} `json:"format"`
	}
	err = json.Unmarshal(out.Bytes(), &probe)
	if err == nil && len(probe.Streams) > 0 {
		for _, stream := range probe.Streams {
			if stream.CodecType == "video" {
				mediaMeta.Resolution = fmt.Sprintf("%dx%d", stream.Width, stream.Height)
				if stream.BitRate != "" {
					_, err := fmt.Sscanf(stream.BitRate, "%d", &mediaMeta.Bitrate)
					if err != nil {
						return nil, err
					}
				}
				if stream.Duration != "" {
					_, err := fmt.Sscanf(stream.Duration, "%f", &mediaMeta.Duration)
					if err != nil {
						return nil, err
					}
				}
			}
		}
		if probe.Format.Duration != "" {
			_, err := fmt.Sscanf(probe.Format.Duration, "%f", &mediaMeta.Duration)
			if err != nil {
				return nil, err
			}
		}
		if probe.Format.BitRate != "" {
			_, err := fmt.Sscanf(probe.Format.BitRate, "%d", &mediaMeta.Bitrate)
			if err != nil {
				return nil, err
			}
		}
	}
	if err != nil {
		s.log.Error("Failed to remove temporary file", zap.String("file", tmpFile), zap.Error(err))
	}

	// --- Populate other metadata fields as needed (captions, accessibility, etc.) ---
	// (You can extend this section to auto-detect or attach more fields)

	metaStruct, err := MetadataToStruct(mediaMeta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal media metadata: %v", err)
	}
	metaForValidation := &structpb.Struct{Fields: map[string]*structpb.Value{"media": structpb.NewStructValue(metaStruct)}}
	if err := pkgmeta.ValidateMetadata(&commonpb.Metadata{ServiceSpecific: metaForValidation}); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	asset := &Model{
		ID:        uuid.New(),
		UserID:    uuid.MustParse(req.UserId),
		Type:      StorageTypeLight,
		Name:      req.Name,
		MimeType:  req.MimeType,
		Size:      req.Size,
		URL:       blobURL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  &commonpb.Metadata{ServiceSpecific: metaForValidation},
	}
	if err := s.repo.CreateMedia(ctx, asset); err != nil {
		s.log.Error("failed to create light asset", zap.Error(err))
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error(), "user_id": req.UserId, "name": req.Name})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for media event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			asset.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "media.upload_failed", "", errMeta)
		}
		return nil, status.Error(codes.Internal, "failed to create asset")
	}
	// Emit media.uploaded event after successful creation
	if s.eventEnabled && s.eventEmitter != nil {
		asset.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "media.uploaded", asset.ID.String(), asset.Metadata)
	}

	if s.cache != nil {
		channels := []string{"media:events:system"}
		if req.UserId != "" {
			channels = append(channels, "media:events:user:"+req.UserId)
		}
		mediaEvent := struct {
			Type string         `json:"type"`
			Data *mediapb.Media `json:"data"`
		}{
			Type: "upload",
			Data: &mediapb.Media{
				Id:        asset.ID.String(),
				UserId:    req.UserId,
				Type:      mediapb.MediaType_MEDIA_TYPE_LIGHT,
				Name:      req.Name,
				MimeType:  req.MimeType,
				Size:      req.Size,
				Url:       blobURL,
				CreatedAt: timestamppb.New(asset.CreatedAt),
				UpdatedAt: timestamppb.New(asset.UpdatedAt),
				Metadata:  asset.Metadata,
			},
		}
		if payload, err := json.Marshal(mediaEvent); err == nil {
			for _, ch := range channels {
				if err := s.cache.GetClient().Publish(ctx, ch, payload).Err(); err != nil {
					return nil, err
				}
			}
		}
	}

	return &mediapb.UploadLightMediaResponse{
		Media: &mediapb.Media{
			Id:        asset.ID.String(),
			UserId:    req.UserId,
			Type:      mediapb.MediaType_MEDIA_TYPE_LIGHT,
			Name:      req.Name,
			MimeType:  req.MimeType,
			Size:      req.Size,
			Url:       blobURL,
			CreatedAt: timestamppb.New(asset.CreatedAt),
			UpdatedAt: timestamppb.New(asset.UpdatedAt),
			Metadata:  asset.Metadata,
		},
		Status: "created",
	}, nil
}

// StartHeavyMediaUpload initiates a chunked upload for large media.
func (s *ServiceImpl) StartHeavyMediaUpload(ctx context.Context, req *mediapb.StartHeavyMediaUploadRequest) (*mediapb.StartHeavyMediaUploadResponse, error) {
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
	var mediaMeta *Metadata
	if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
		mediaField, ok := req.Metadata.ServiceSpecific.Fields["media"]
		if ok && mediaField.GetKind() != nil {
			var err error
			mediaMeta, err = MetadataFromStruct(mediaField.GetStructValue())
			if err != nil {
				return nil, err
			}
		}
	}
	if mediaMeta == nil {
		mediaMeta = &Metadata{}
	}
	if mediaMeta.Versioning == nil {
		mediaMeta.Versioning = map[string]interface{}{
			"system_version":   "1.0.0",
			"service_version":  "1.0.0",
			"environment":      "prod",
			"last_migrated_at": time.Now().Format(time.RFC3339),
		}
	}
	metaStruct, err := MetadataToStruct(mediaMeta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal media metadata: %v", err)
	}
	metaForValidation := &structpb.Struct{Fields: map[string]*structpb.Value{"media": structpb.NewStructValue(metaStruct)}}
	if err := pkgmeta.ValidateMetadata(&commonpb.Metadata{ServiceSpecific: metaForValidation}); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}

	// --- Azure Blob Storage: Generate SAS URL for chunked upload ---
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey := os.Getenv("AZURE_BLOB_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER")
	if accountName == "" || accountKey == "" || containerName == "" {
		return nil, status.Error(codes.Internal, "Azure Blob Storage config missing")
	}
	// NOTE: If the proto is updated, return the SAS URL here.

	// Store session info in-memory (for demo; use Redis/DB for prod)
	uploadID := uuid.New().String()
	heavyUploadSessions[uploadID] = &UploadMetadata{
		ID:             uuid.MustParse(uploadID),
		UserID:         userID,
		Size:           req.Size,
		ChunksTotal:    int(chunksTotal),
		ChunksReceived: 0,
		Checksum:       "",
		StartedAt:      time.Now(),
		LastUpdate:     time.Now(),
	}

	// Create asset record in DB (URL will be set after upload/encoding)
	asset := &Model{
		ID:        uuid.MustParse(uploadID),
		UserID:    userID,
		Type:      StorageTypeHeavy,
		Name:      req.Name,
		MimeType:  req.MimeType,
		Size:      req.Size,
		URL:       "", // Will be set after upload/encoding
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  &commonpb.Metadata{ServiceSpecific: metaForValidation},
	}
	if err := s.repo.CreateMedia(ctx, asset); err != nil {
		s.log.Error("failed to create heavy asset", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to initialize upload")
	}

	return &mediapb.StartHeavyMediaUploadResponse{
		UploadId:    uploadID,
		ChunkSize:   utils.ToInt32(chunkSize),
		ChunksTotal: utils.ToInt32(int(chunksTotal)),
		Status:      "created",
		Error:       "",
	}, nil
}

// StreamMediaChunk handles streaming chunks for heavy media uploads.
func (s *ServiceImpl) StreamMediaChunk(ctx context.Context, req *mediapb.StreamMediaChunkRequest) (*mediapb.StreamMediaChunkResponse, error) {
	// Validate upload session
	session, ok := heavyUploadSessions[req.UploadId]
	if !ok {
		return nil, status.Error(codes.NotFound, "upload session not found")
	}
	// Validate chunk order (for demo, just increment)
	if session.ChunksReceived >= session.ChunksTotal {
		return nil, status.Error(codes.InvalidArgument, "all chunks already received")
	}
	// Validate chunk size (for demo, skip; in prod, check req.ChunkSize)
	// Upload chunk to Azure Blob Storage (append block)
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey := os.Getenv("AZURE_BLOB_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER")
	if accountName == "" || accountKey == "" || containerName == "" {
		return nil, status.Error(codes.Internal, "Azure Blob Storage config missing")
	}
	// For demo, reconstruct blob name from upload ID
	blobName := req.UploadId + ".mp4"
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create Azure credential: %v", err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	blockBlobClient, err := blockblob.NewClientWithSharedKeyCredential(serviceURL+containerName+"/"+blobName, cred, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create block blob client: %v", err)
	}
	// For demo: use StageBlock to upload chunk (in prod, use block IDs)
	var chunkData []byte
	if req.Chunk != nil {
		chunkData = req.Chunk.Data
	} else {
		return nil, status.Error(codes.InvalidArgument, "chunk data missing")
	}
	reader := &readSeekCloser{bytes.NewReader(chunkData)}
	blockID := fmt.Sprintf("block-%06d", session.ChunksReceived)
	_, err = blockBlobClient.StageBlock(ctx, blockID, reader, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upload chunk to Azure Blob: %v", err)
	}
	// Update session
	session.ChunksReceived++
	session.LastUpdate = time.Now()
	return &mediapb.StreamMediaChunkResponse{Status: "chunk_received"}, nil
}

// CompleteMediaUpload finalizes a heavy media upload.
func (s *ServiceImpl) CompleteMediaUpload(ctx context.Context, req *mediapb.CompleteMediaUploadRequest) (*mediapb.CompleteMediaUploadResponse, error) {
	// Validate upload session
	session, ok := heavyUploadSessions[req.UploadId]
	if !ok {
		return nil, status.Error(codes.NotFound, "upload session not found")
	}
	// Commit all blocks to finalize the blob in Azure
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey := os.Getenv("AZURE_BLOB_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER")
	if accountName == "" || accountKey == "" || containerName == "" {
		return nil, status.Error(codes.Internal, "Azure Blob Storage config missing")
	}
	blobName := req.UploadId + ".mp4"
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create Azure credential: %v", err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	blockBlobClient, err := blockblob.NewClientWithSharedKeyCredential(serviceURL+containerName+"/"+blobName, cred, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create block blob client: %v", err)
	}
	// Build block list (slice of base64-encoded block IDs)
	blockList := make([]string, session.ChunksTotal)
	for i := 0; i < session.ChunksTotal; i++ {
		blockID := fmt.Sprintf("block-%06d", i)
		blockList[i] = base64.StdEncoding.EncodeToString([]byte(blockID))
	}
	_, err = blockBlobClient.CommitBlockList(ctx, blockList, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit block list: %v", err)
	}

	// (Stub) Trigger Azure Media Services encoding and poll for completion
	// TODO: Call Azure Media Services REST API to start encoding job
	// TODO: Poll for job completion and get playback URLs, thumbnails, etc.
	// For now, mock playback URLs and thumbnails
	playbackURLs := map[string]string{
		"hls":  "https://mock.streaming.media.azure.net/hls/" + req.UploadId + ".m3u8",
		"dash": "https://mock.streaming.media.azure.net/dash/" + req.UploadId + ".mpd",
	}
	thumbnails := []ThumbnailInfo{{
		URL:         "https://mock.blob.core.windows.net/thumbnails/" + req.UploadId + ".jpg",
		Width:       320,
		Height:      180,
		TimeOffset:  0.0,
		Description: "Mock thumbnail",
	}}

	// Update asset in DB
	id, err := uuid.Parse(req.UploadId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid upload ID")
	}
	asset, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "media not found")
	}
	var mediaMeta *Metadata
	if asset.Metadata != nil && asset.Metadata.ServiceSpecific != nil {
		mediaField, ok := asset.Metadata.ServiceSpecific.Fields["media"]
		if ok && mediaField.GetKind() != nil {
			mediaMeta, err = MetadataFromStruct(mediaField.GetStructValue())
			if err != nil {
				return nil, err
			}
		}
	}
	if mediaMeta == nil {
		mediaMeta = &Metadata{}
	}
	mediaMeta.PlaybackURLs = playbackURLs
	mediaMeta.Thumbnails = thumbnails
	metaStruct, err := MetadataToStruct(mediaMeta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal updated media metadata: %v", err)
	}
	metaForValidation := &structpb.Struct{Fields: map[string]*structpb.Value{"media": structpb.NewStructValue(metaStruct)}}
	if err := pkgmeta.ValidateMetadata(&commonpb.Metadata{ServiceSpecific: metaForValidation}); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid updated metadata: %v", err)
	}
	asset.Metadata = &commonpb.Metadata{ServiceSpecific: metaForValidation}
	asset.URL = playbackURLs["hls"] // Use HLS as main URL for demo
	asset.UpdatedAt = time.Now()
	if err := s.repo.UpdateMedia(ctx, asset); err != nil {
		s.log.Error("failed to update asset after encoding", zap.Error(err))
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error(), "upload_id": req.UploadId})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for media event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			asset.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "media.complete_failed", req.UploadId, errMeta)
		}
		return nil, status.Error(codes.Internal, "failed to update asset after encoding")
	}
	// Emit media.completed event after successful completion
	if s.eventEnabled && s.eventEmitter != nil {
		asset.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "media.completed", asset.ID.String(), asset.Metadata)
	}

	if s.cache != nil {
		channels := []string{"media:events:system"}
		if asset.Metadata != nil && asset.Metadata.ServiceSpecific != nil {
			if campaignField, ok := asset.Metadata.ServiceSpecific.Fields["campaign"]; ok {
				if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
					if cid, ok := campaignStruct.Fields["id"]; ok && cid.GetStringValue() != "" {
						channels = append(channels, "media:events:campaign:"+cid.GetStringValue())
					}
				}
			}
		}
		if asset.UserID.String() != "" {
			channels = append(channels, "media:events:user:"+asset.UserID.String())
		}
		mediaEvent := struct {
			Type string         `json:"type"`
			Data *mediapb.Media `json:"data"`
		}{
			Type: "update",
			Data: &mediapb.Media{
				Id:        asset.ID.String(),
				UserId:    asset.UserID.String(),
				Type:      mediapb.MediaType_MEDIA_TYPE_HEAVY,
				Name:      asset.Name,
				MimeType:  asset.MimeType,
				Size:      asset.Size,
				Url:       asset.URL,
				CreatedAt: timestamppb.New(asset.CreatedAt),
				UpdatedAt: timestamppb.New(asset.UpdatedAt),
				Metadata:  asset.Metadata,
			},
		}
		if payload, err := json.Marshal(mediaEvent); err == nil {
			for _, ch := range channels {
				if err := s.cache.GetClient().Publish(ctx, ch, payload).Err(); err != nil {
					return nil, err
				}
			}
		}
	}

	return &mediapb.CompleteMediaUploadResponse{
		Media: &mediapb.Media{
			Id:        asset.ID.String(),
			UserId:    asset.UserID.String(),
			Type:      mediapb.MediaType_MEDIA_TYPE_HEAVY,
			Name:      asset.Name,
			MimeType:  asset.MimeType,
			Size:      asset.Size,
			Url:       asset.URL,
			CreatedAt: timestamppb.New(asset.CreatedAt),
			UpdatedAt: timestamppb.New(asset.UpdatedAt),
			Metadata:  asset.Metadata,
		},
		Status: "ready",
		Error:  "",
	}, nil
}

// GetMedia retrieves a media file.
func (s *ServiceImpl) GetMedia(ctx context.Context, req *mediapb.GetMediaRequest) (*mediapb.GetMediaResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid media ID")
	}
	media, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		if errors.Is(err, ErrMediaNotFound) {
			return nil, status.Error(codes.NotFound, "media not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get media: %v", err)
	}
	masterIDInt64, err := strconv.ParseInt(media.MasterID, 10, 64)
	if err != nil {
		masterIDInt64 = 0 // or handle error as needed
	}
	return &mediapb.GetMediaResponse{
		Media: &mediapb.Media{
			Id:        media.ID.String(),
			MasterId:  masterIDInt64,
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
func (s *ServiceImpl) StreamMediaContent(ctx context.Context, req *mediapb.StreamMediaContentRequest) (*mediapb.StreamMediaContentResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid media ID")
	}
	asset, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		return nil, status.Error(codes.NotFound, "media not found")
	}
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey := os.Getenv("AZURE_BLOB_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER")
	if accountName == "" || accountKey == "" || containerName == "" {
		return nil, status.Error(codes.Internal, "Azure Blob Storage config missing")
	}
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create Azure credential: %v", err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	blobName := asset.ID.String() + ".mp4"
	blobClient, err := blob.NewClientWithSharedKeyCredential(serviceURL+containerName+"/"+blobName, cred, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create blob client: %v", err)
	}
	// Use the correct type for download options
	var downloadOptions *blob.DownloadStreamOptions
	if req.Offset > 0 || req.Length > 0 {
		rangeOpt := blob.HTTPRange{Offset: req.Offset}
		if req.Length > 0 {
			rangeOpt.Count = req.Length
		}
		downloadOptions = &blob.DownloadStreamOptions{
			Range: rangeOpt,
		}
	}
	resp, err := blobClient.DownloadStream(ctx, downloadOptions)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to download from Azure Blob: %v", err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read blob data: %v", err)
	}
	return &mediapb.StreamMediaContentResponse{Data: data, Status: "ok"}, nil
}

// DeleteMedia deletes a media file.
func (s *ServiceImpl) DeleteMedia(ctx context.Context, req *mediapb.DeleteMediaRequest) (*mediapb.DeleteMediaResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid media ID")
	}
	// Get asset from DB
	asset, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get media: %v", err)
	}
	// Remove from Azure Blob Storage
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey := os.Getenv("AZURE_BLOB_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER")
	if accountName == "" || accountKey == "" || containerName == "" {
		return nil, status.Error(codes.Internal, "Azure Blob Storage config missing")
	}
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create Azure credential: %v", err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	blobName := asset.ID.String() + ".mp4"
	blockBlobClient, err := blockblob.NewClientWithSharedKeyCredential(serviceURL+containerName+"/"+blobName, cred, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create block blob client: %v", err)
	}
	_, err = blockBlobClient.Delete(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete blob: %v", err)
	}
	// Remove from DB
	if err := s.repo.DeleteMedia(ctx, id); err != nil {
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			errStruct, err := structpb.NewStruct(map[string]interface{}{"error": err.Error(), "media_id": req.Id})
			if err != nil {
				s.log.Error("Failed to create structpb.Struct for media event", zap.Error(err))
				return nil, status.Error(codes.Internal, "internal error")
			}
			errMeta := &commonpb.Metadata{}
			errMeta.ServiceSpecific = errStruct
			asset.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "media.delete_failed", req.Id, errMeta)
		}
		return nil, status.Errorf(codes.Internal, "failed to delete media: %v", err)
	}
	// Emit media.deleted event after successful deletion
	if s.eventEnabled && s.eventEmitter != nil {
		asset.Metadata, _ = events.EmitEventWithLogging(ctx, s.eventEmitter, s.log, "media.deleted", req.Id, asset.Metadata)
	}

	if s.cache != nil {
		channels := []string{"media:events:system"}
		if asset.Metadata != nil && asset.Metadata.ServiceSpecific != nil {
			if campaignField, ok := asset.Metadata.ServiceSpecific.Fields["campaign"]; ok {
				if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
					if cid, ok := campaignStruct.Fields["id"]; ok && cid.GetStringValue() != "" {
						channels = append(channels, "media:events:campaign:"+cid.GetStringValue())
					}
				}
			}
		}
		if asset.UserID.String() != "" {
			channels = append(channels, "media:events:user:"+asset.UserID.String())
		}
		mediaEvent := struct {
			Type string         `json:"type"`
			Data *mediapb.Media `json:"data"`
		}{
			Type: "delete",
			Data: &mediapb.Media{
				Id:   asset.ID.String(),
				Name: asset.Name,
			},
		}
		if payload, err := json.Marshal(mediaEvent); err == nil {
			for _, ch := range channels {
				if err := s.cache.GetClient().Publish(ctx, ch, payload).Err(); err != nil {
					return nil, err
				}
			}
		}
	}

	return &mediapb.DeleteMediaResponse{Status: "deleted"}, nil
}

// ListUserMedia lists user media files with pagination and metadata.
func (s *ServiceImpl) ListUserMedia(ctx context.Context, req *mediapb.ListUserMediaRequest) (*mediapb.ListUserMediaResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}
	// For demo: ignore PageToken, use offset 0. In prod, decode PageToken for offset.
	mediaList, err := s.repo.ListUserMedia(ctx, userID, "", int(req.GetPageSize()), 0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user media: %v", err)
	}
	result := make([]*mediapb.Media, 0, len(mediaList))
	for _, m := range mediaList {
		result = append(result, &mediapb.Media{
			Id:        m.ID.String(),
			UserId:    m.UserID.String(),
			Type:      mediapb.MediaType(mediapb.MediaType_value[string(m.Type)]),
			Name:      m.Name,
			MimeType:  m.MimeType,
			Size:      m.Size,
			Url:       m.URL,
			CreatedAt: timestamppb.New(m.CreatedAt),
			UpdatedAt: timestamppb.New(m.UpdatedAt),
			Metadata:  m.Metadata,
		})
	}
	return &mediapb.ListUserMediaResponse{Media: result, Status: "ok"}, nil
}

// ListSystemMedia lists system media files with pagination and metadata.
func (s *ServiceImpl) ListSystemMedia(ctx context.Context, req *mediapb.ListSystemMediaRequest) (*mediapb.ListSystemMediaResponse, error) {
	// For demo: ignore PageToken, use offset 0. In prod, decode PageToken for offset.
	mediaList, err := s.repo.ListSystemMedia(ctx, "", int(req.GetPageSize()), 0)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list system media: %v", err)
	}
	result := make([]*mediapb.Media, 0, len(mediaList))
	for _, m := range mediaList {
		result = append(result, &mediapb.Media{
			Id:        m.ID.String(),
			UserId:    m.UserID.String(),
			Type:      mediapb.MediaType(mediapb.MediaType_value[string(m.Type)]),
			Name:      m.Name,
			MimeType:  m.MimeType,
			Size:      m.Size,
			Url:       m.URL,
			CreatedAt: timestamppb.New(m.CreatedAt),
			UpdatedAt: timestamppb.New(m.UpdatedAt),
			Metadata:  m.Metadata,
		})
	}
	return &mediapb.ListSystemMediaResponse{Media: result, Status: "ok"}, nil
}

// SubscribeToUserMedia: basic in-memory pub/sub demo.
func (s *ServiceImpl) SubscribeToUserMedia(ctx context.Context, req *mediapb.SubscribeToUserMediaRequest) (*mediapb.SubscribeToUserMediaResponse, error) {
	ch := make(chan *mediapb.Media, 1)
	userMediaSubs[req.UserId] = ch
	defer delete(userMediaSubs, req.UserId)
	// Simulate a push update (in prod, use real pub/sub or WebSocket)
	select {
	case media := <-ch:
		return &mediapb.SubscribeToUserMediaResponse{Media: []*mediapb.Media{media}, Status: "update"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
		return &mediapb.SubscribeToUserMediaResponse{Media: []*mediapb.Media{}, Status: "timeout"}, nil
	}
}

// SubscribeToSystemMedia: basic in-memory pub/sub demo.
func (s *ServiceImpl) SubscribeToSystemMedia(ctx context.Context, _ *mediapb.SubscribeToSystemMediaRequest) (*mediapb.SubscribeToSystemMediaResponse, error) {
	// Use a static key for system-wide subscriptions (since req.MasterId does not exist)
	key := "system"
	ch := make(chan *mediapb.Media, 1)
	systemMediaSubs[key] = ch
	defer delete(systemMediaSubs, key)
	// Simulate a push update (in prod, use real pub/sub or WebSocket)
	select {
	case media := <-ch:
		return &mediapb.SubscribeToSystemMediaResponse{Media: []*mediapb.Media{media}, Status: "update"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
		return &mediapb.SubscribeToSystemMediaResponse{Media: []*mediapb.Media{}, Status: "timeout"}, nil
	}
}

// BroadcastSystemMedia: push a mock update to all system subscribers.
func (s *ServiceImpl) BroadcastSystemMedia(ctx context.Context, req *mediapb.BroadcastSystemMediaRequest) (*mediapb.BroadcastSystemMediaResponse, error) {
	// In prod, integrate with pub/sub or event bus
	mockMedia := &mediapb.Media{
		Id:   "system-broadcast-id",
		Name: "System Broadcast Media",
	}
	for _, ch := range systemMediaSubs {
		select {
		case ch <- mockMedia:
		default:
		}
	}

	if s.cache != nil {
		channels := []string{"media:events:system"}
		if req.UserId != "" {
			channels = append(channels, "media:events:user:"+req.UserId)
		}
		mediaEvent := struct {
			Type string         `json:"type"`
			Data *mediapb.Media `json:"data"`
		}{
			Type: "broadcast",
			Data: &mediapb.Media{
				Id:   "system-broadcast-id",
				Name: "System Broadcast Media",
			},
		}
		if payload, err := json.Marshal(mediaEvent); err == nil {
			for _, ch := range channels {
				if err := s.cache.GetClient().Publish(ctx, ch, payload).Err(); err != nil {
					return nil, err
				}
			}
		}
	}

	return &mediapb.BroadcastSystemMediaResponse{Status: "broadcasted"}, nil
}

// UploadChunks uploads media chunks concurrently with retry and timeout logic.
// It uses maxRetries, uploadTimeout, chunkTimeout, and maxConcurrentUploadChunks constants.
func (s *ServiceImpl) UploadChunks(ctx context.Context, chunks [][]byte, uploadChunkFunc func(context.Context, []byte) error) error {
	ctx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()

	sem := make(chan struct{}, maxConcurrentUploadChunks)
	var wg sync.WaitGroup
	var uploadErr error
	var mu sync.Mutex

	for _, chunk := range chunks {
		wg.Add(1)
		sem <- struct{}{}
		go func(chunkData []byte) {
			defer wg.Done()
			defer func() { <-sem }()
			var err error
			for attempt := 1; attempt <= maxRetries; attempt++ {
				chunkCtx, chunkCancel := context.WithTimeout(ctx, chunkTimeout)
				err = uploadChunkFunc(chunkCtx, chunkData)
				chunkCancel()
				if err == nil {
					break
				}
				if attempt < maxRetries {
					time.Sleep(time.Second * time.Duration(attempt)) // Exponential backoff
				}
			}
			if err != nil {
				mu.Lock()
				if uploadErr == nil {
					uploadErr = err
				}
				mu.Unlock()
			}
		}(chunk)
	}
	wg.Wait()
	return uploadErr
}

// Compile-time check.
var _ mediapb.MediaServiceServer = (*ServiceImpl)(nil)
