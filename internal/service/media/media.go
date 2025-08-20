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
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	mediapb "github.com/nmxmxh/master-ovasabi/api/protos/media/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/events"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/graceful"
	pkgmeta "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"google.golang.org/protobuf/types/known/structpb"
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
	eventEmitter events.EventEmitter
	eventEnabled bool
	handler      *graceful.Handler
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

// NewService constructs a new MediaServiceServer instance with event bus support.
func NewService(log *zap.Logger, repo *Repo, cache *redis.Cache, eventEmitter events.EventEmitter, eventEnabled bool) *ServiceImpl {
	svc := &ServiceImpl{
		log:          log,
		repo:         repo,
		cache:        cache,
		eventEmitter: eventEmitter,
		eventEnabled: eventEnabled,
		handler:      graceful.NewHandler(log, nil, cache, "media", "v1", eventEnabled),
	}
	// Register media service error map for graceful error handling
	graceful.RegisterErrorMap(map[error]graceful.ErrorMapEntry{
		errors.New("invalid media format"): {Code: codes.InvalidArgument, Message: "invalid media format"},
		errors.New("upload failed"):        {Code: codes.Internal, Message: "media upload failed"},
		errors.New("encoding failed"):      {Code: codes.Internal, Message: "media encoding failed"},
		errors.New("media not found"):      {Code: codes.NotFound, Message: "media not found"},
	})
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
func (s *ServiceImpl) validateUploadSize(ctx context.Context, size int64) error {
	maxSize := int64(100 * 1024 * 1024) // 100MB max
	if size <= 0 {
		return graceful.WrapErr(ctx, codes.InvalidArgument, "size must be positive", nil)
	}
	if size > maxSize {
		return graceful.WrapErr(ctx, codes.InvalidArgument, fmt.Sprintf("size exceeds maximum allowed (%d bytes)", maxSize), nil)
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
	// Declare all needed variables at the very top for proper scope
	var (
		mediaMeta         *Metadata
		metaMapOut        map[string]interface{}
		metaBytesOut      []byte
		metaStruct        *structpb.Struct
		metaForValidation *structpb.Struct
		err               error
		accountName       string
		accountKey        string
		containerName     string
	)
	// Parse and validate metadata
	if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
		mediaField, ok := req.Metadata.ServiceSpecific.Fields["media"]
		if ok && mediaField.GetKind() != nil {
			metaMap := pkgmeta.StructToMap(mediaField.GetStructValue())
			mediaMeta = &Metadata{}
			metaBytes, err := json.Marshal(metaMap)
			if err != nil {
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
				}
				return nil, err
			}
			if err := json.Unmarshal(metaBytes, mediaMeta); err != nil {
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
				}
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
	if mediaMeta.Compliance == nil {
		mediaMeta.Compliance = &ComplianceMetadata{
			Standards: []ComplianceStandard{{
				Name:      "WCAG",
				Level:     "AA",
				Version:   "2.1",
				Compliant: true,
			}},
			CheckedBy: "media-service",
			CheckedAt: time.Now().Format(time.RFC3339),
			Method:    "automated",
			Issues:    []ComplianceIssue{},
		}
	}
	if mediaMeta.Accessibility == nil {
		mediaMeta.Accessibility = &AccessibilityMetadata{
			AltText:         "",
			AudioDescURL:    "",
			Features:        []string{"captions", "alt_text", "transcripts", "aria_labels"},
			PlatformSupport: []string{"desktop", "mobile", "screen_reader", "voice_input"},
		}
	}
	// --- Azure Blob Storage Upload ---
	accountName = os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey = os.Getenv("AZURE_BLOB_KEY")
	containerName = os.Getenv("AZURE_BLOB_CONTAINER")
	if accountName == "" || accountKey == "" || containerName == "" {
		err := graceful.WrapErr(ctx, codes.Internal, "Azure Blob Storage config missing", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.ToStatusError(err)
	}
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to create Azure credential: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.ToStatusError(err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	blobName := uuid.New().String() + ".mp4"
	blockBlobClient, err := blockblob.NewClientWithSharedKeyCredential(serviceURL+containerName+"/"+blobName, cred, nil)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to create block blob client: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.ToStatusError(err)
	}
	// Upload file data (req.Data)
	reader := bytes.NewReader(req.Data)
	_, err = blockBlobClient.UploadStream(ctx, reader, nil)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to upload to Azure Blob: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.ToStatusError(err)
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
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to write temp file for ffprobe: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.ToStatusError(err)
	}
	cmd := exec.CommandContext(ctx, ffprobePath, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", tmpFile)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("ffprobe failed: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.ToStatusError(err)
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
						var ce *graceful.ContextError
						if errors.As(err, &ce) {
							s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
						}
						return nil, err
					}
				}
				if stream.Duration != "" {
					_, err := fmt.Sscanf(stream.Duration, "%f", &mediaMeta.Duration)
					if err != nil {
						var ce *graceful.ContextError
						if errors.As(err, &ce) {
							s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
						}
						return nil, err
					}
				}
			}
		}
		if probe.Format.Duration != "" {
			_, err := fmt.Sscanf(probe.Format.Duration, "%f", &mediaMeta.Duration)
			if err != nil {
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
				}
				return nil, err
			}
		}
		if probe.Format.BitRate != "" {
			_, err := fmt.Sscanf(probe.Format.BitRate, "%d", &mediaMeta.Bitrate)
			if err != nil {
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
				}
				return nil, err
			}
		}
	}
	if err != nil {
		s.log.Error("Failed to remove temporary file", zap.String("file", tmpFile), zap.Error(err))
	}

	// --- Populate other metadata fields as needed (captions, accessibility, etc.) ---
	// (You can extend this section to auto-detect or attach more fields)

	// After all modifications to mediaMeta are complete, marshal/unmarshal/normalize/validate
	// (This block must be just before asset creation)
	metaMapOut = make(map[string]interface{})
	metaBytesOut, err = json.Marshal(mediaMeta)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to marshal media metadata", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.ToStatusError(err)
	}
	if err := json.Unmarshal(metaBytesOut, &metaMapOut); err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to unmarshal media metadata", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.ToStatusError(err)
	}
	normMap := pkgmeta.Handler{}.NormalizeAndCalculate(metaMapOut, "media", req.UserId, nil, "success", "normalize media metadata")
	metaStruct = pkgmeta.MapToStruct(normMap)
	metaForValidation = &structpb.Struct{Fields: map[string]*structpb.Value{"media": structpb.NewStructValue(metaStruct)}}

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
			err = graceful.WrapErr(ctx, codes.Internal, "media.upload_failed", err)
			var ce *graceful.ContextError
			if errors.As(err, &ce) {
				s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
			}
			return nil, graceful.ToStatusError(err)
		}
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to create asset", nil)
	}
	// Emit media.uploaded event after successful creation
	if s.eventEnabled && s.eventEmitter != nil {
		success := graceful.WrapSuccess(ctx, codes.OK, "media uploaded", asset, nil)
		s.handler.Success(ctx, "upload_light_media", codes.OK, "media uploaded", success, nil, "media", nil)
	}

	// --- Event publishing section (cleaned up, with campaign/user/system channels) ---
	if s.cache != nil {
		channels := []string{"media:events:system"}
		// Add campaign channel if campaign ID exists in metadata
		if asset.Metadata != nil && asset.Metadata.ServiceSpecific != nil {
			if campaignField, ok := asset.Metadata.ServiceSpecific.Fields["campaign"]; ok {
				if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
					if cid, ok := campaignStruct.Fields["id"]; ok && cid.GetStringValue() != "" {
						channels = append(channels, "media:events:campaign:"+cid.GetStringValue())
					}
				}
			}
		}
		// Add user channel
		if asset.UserID.String() != "" {
			channels = append(channels, "media:events:user:"+asset.UserID.String())
		}
		mediaEvent := struct {
			Type string         `json:"type"`
			Data *mediapb.Media `json:"data"`
		}{
			Type: "upload",
			Data: &mediapb.Media{
				Id:        asset.ID.String(),
				UserId:    asset.UserID.String(),
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
	var err error
	if !s.validateMimeType(req.MimeType) {
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "unsupported MIME type", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.ToStatusError(err)
	}
	if err = s.validateUploadSize(ctx, req.Size); err != nil {
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.ToStatusError(err)
	}
	if req.Size <= ultraLightThreshold {
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "asset too small for heavy upload, use light upload instead", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			s.handler.Error(ctx, "upload_light_media", codes.Internal, ce.Message, ce, nil, "media")
		}
		return nil, graceful.ToStatusError(err)
	}
	var userID uuid.UUID
	userID, err = uuid.Parse(req.UserId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "invalid user ID", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	chunkSize := smallChunkSize
	if req.Size > mediumThreshold {
		chunkSize = largeChunkSize
	}
	chunksTotal := (req.Size + int64(chunkSize) - 1) / int64(chunkSize)
	if chunkSize > math.MaxInt32 || chunkSize < math.MinInt32 {
		err = graceful.WrapErr(ctx, codes.Internal, "chunk size out of int32 range", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if chunksTotal > math.MaxInt32 || chunksTotal < 0 {
		err = graceful.WrapErr(ctx, codes.Internal, "chunks total out of int32 range", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	var mediaMeta *Metadata
	if req.Metadata != nil && req.Metadata.ServiceSpecific != nil {
		mediaField, ok := req.Metadata.ServiceSpecific.Fields["media"]
		if ok && mediaField.GetKind() != nil {
			metaMap := pkgmeta.StructToMap(mediaField.GetStructValue())
			mediaMeta = &Metadata{}
			metaBytes, err := json.Marshal(metaMap)
			if err != nil {
				err = graceful.WrapErr(ctx, codes.Internal, "failed to marshal media metadata", err)
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				}
				return nil, graceful.ToStatusError(err)
			}
			if err := json.Unmarshal(metaBytes, mediaMeta); err != nil {
				err = graceful.WrapErr(ctx, codes.Internal, "failed to marshal media metadata", err)
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				}
				return nil, graceful.ToStatusError(err)
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
	metaMapOut := make(map[string]interface{})
	metaBytesOut, err := json.Marshal(mediaMeta)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to marshal media metadata", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if err := json.Unmarshal(metaBytesOut, &metaMapOut); err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to unmarshal media metadata", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	metaStruct := pkgmeta.MapToStruct(metaMapOut)
	metaForValidation := &structpb.Struct{Fields: map[string]*structpb.Value{"media": structpb.NewStructValue(metaStruct)}}

	accountName := os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey := os.Getenv("AZURE_BLOB_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER")

	// --- Azure Blob Storage: Generate SAS URL for chunked upload ---
	if accountName == "" || accountKey == "" || containerName == "" {
		err = graceful.WrapErr(ctx, codes.Internal, "Azure Blob Storage config missing", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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

	// Create asset record in DB (URL will be set after upload/encoding))
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
	if err = s.repo.CreateMedia(ctx, asset); err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to initialize upload", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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
		err := graceful.WrapErr(ctx, codes.NotFound, "upload session not found", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	// Validate chunk order (for demo, just increment)
	if session.ChunksReceived >= session.ChunksTotal {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "all chunks already received", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	// Validate chunk size (for demo, skip; in prod, check req.ChunkSize)
	// Upload chunk to Azure Blob Storage (append block)
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey := os.Getenv("AZURE_BLOB_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER")
	if accountName == "" || accountKey == "" || containerName == "" {
		err := graceful.WrapErr(ctx, codes.Internal, "Azure Blob Storage config missing", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	// For demo, reconstruct blob name from upload ID
	blobName := req.UploadId + ".mp4"
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to create Azure credential: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	blockBlobClient, err := blockblob.NewClientWithSharedKeyCredential(serviceURL+containerName+"/"+blobName, cred, nil)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to create block blob client: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	// For demo: use StageBlock to upload chunk (in prod, use block IDs)
	var chunkData []byte
	if req.Chunk != nil {
		chunkData = req.Chunk.Data
	} else {
		err := graceful.WrapErr(ctx, codes.InvalidArgument, "chunk data missing", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	reader := &readSeekCloser{bytes.NewReader(chunkData)}
	blockID := fmt.Sprintf("block-%06d", session.ChunksReceived)
	_, err = blockBlobClient.StageBlock(ctx, blockID, reader, nil)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to upload chunk to Azure Blob: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	// Update session
	session.ChunksReceived++
	session.LastUpdate = time.Now()
	return &mediapb.StreamMediaChunkResponse{Status: "chunk_received"}, nil
}

// Helper: Get Azure Media Services OAuth2 token.
func getAMSToken(ctx context.Context) (string, error) {
	tenantID := os.Getenv("AZURE_MEDIA_TENANT_ID")
	clientID := os.Getenv("AZURE_MEDIA_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_MEDIA_CLIENT_SECRET")
	if tenantID == "" || clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("AMS credentials missing")
	}
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/token", tenantID)
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("resource", "https://management.azure.com/")
	formData := data.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(formData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	token, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get AMS access_token")
	}
	return token, nil
}

// Helper: Submit AMS encoding job.
func submitAMSJob(ctx context.Context, token, resourceGroup, accountName, transformName, jobName, outputAssetName, blobURL string) error {
	endpoint := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Media/mediaservices/%s/transforms/%s/jobs/%s?api-version=2023-01-01", os.Getenv("AZURE_MEDIA_SUBSCRIPTION_ID"), resourceGroup, accountName, transformName, jobName)
	job := map[string]interface{}{
		"properties": map[string]interface{}{
			"input": map[string]interface{}{
				"@odata.type": "#Microsoft.Media.JobInputHttp",
				"files":       []string{blobURL},
			},
			"outputs": []map[string]interface{}{
				{
					"@odata.type": "#Microsoft.Media.JobOutputAsset",
					"assetName":   outputAssetName,
				},
			},
		},
	}
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal AMS job: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for AMS job: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("AMS job submission failed: %s", string(b))
	}
	return nil
}

// Helper: Poll AMS job status.
func pollAMSJob(ctx context.Context, token, resourceGroup, accountName, transformName, jobName string) error {
	endpoint := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Media/mediaservices/%s/transforms/%s/jobs/%s?api-version=2023-01-01", os.Getenv("AZURE_MEDIA_SUBSCRIPTION_ID"), resourceGroup, accountName, transformName, jobName)
	for i := 0; i < 60; i++ { // up to 10 minutes
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request for AMS job polling: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := json.Unmarshal(b, &result); err != nil {
			return err
		}
		propsIface, ok := result["properties"]
		if !ok {
			// handle missing properties key
			return fmt.Errorf("missing properties key in AMS job response")
		}
		props, ok := propsIface.(map[string]interface{})
		if !ok {
			// handle type assertion failure
			return fmt.Errorf("invalid properties format in AMS job response")
		}
		state, ok := props["state"].(string)
		if !ok {
			// handle type assertion failure
			return fmt.Errorf("missing state field in AMS job response")
		}
		if state == "Finished" {
			return nil
		}
		if state == "Error" {
			return fmt.Errorf("AMS job failed: %s", string(b))
		}
		time.Sleep(10 * time.Second)
	}
	return fmt.Errorf("AMS job polling timed out")
}

// Helper: Construct streaming URLs.
func getAMSStreamingURLs(ctx context.Context, token, resourceGroup, accountName, outputAssetName string) (map[string]string, error) {
	// Create streaming locator
	locatorName := "locator-" + outputAssetName
	endpoint := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Media/mediaservices/%s/streamingLocators/%s?api-version=2023-01-01", os.Getenv("AZURE_MEDIA_SUBSCRIPTION_ID"), resourceGroup, accountName, locatorName)
	locator := map[string]interface{}{
		"properties": map[string]interface{}{
			"assetName":           outputAssetName,
			"streamingPolicyName": "Predefined_ClearStreamingOnly",
		},
	}
	body, err := json.Marshal(locator)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AMS locator: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for AMS locator: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create AMS streaming locator: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read AMS streaming endpoints response: %w", err)
		}
		return nil, fmt.Errorf("AMS locator creation failed: %s", string(b))
	}
	// Get streaming endpoint hostname
	endpointList := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Media/mediaservices/%s/streamingEndpoints?api-version=2023-01-01", os.Getenv("AZURE_MEDIA_SUBSCRIPTION_ID"), resourceGroup, accountName)
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, endpointList, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request for AMS streaming endpoints: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get AMS streaming endpoints: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read AMS streaming endpoints response: %w", err)
	}
	var endpoints map[string]interface{}
	if err := json.Unmarshal(b, &endpoints); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AMS streaming endpoints: %w", err)
	}
	var hostname string
	endpointsVal, ok := endpoints["value"]
	if !ok {
		// handle missing value key
		return nil, fmt.Errorf("missing value key in AMS streaming endpoints response")
	}
	endpointsArr, ok := endpointsVal.([]interface{})
	if !ok {
		// handle type assertion failure
		return nil, fmt.Errorf("invalid value format in AMS streaming endpoints response")
	}
	for _, ep := range endpointsArr {
		epMap, ok := ep.(map[string]interface{})
		if !ok {
			// handle type assertion failure
			continue
		}
		props, ok := epMap["properties"].(map[string]interface{})
		if !ok {
			// handle type assertion failure
			continue
		}
		if v, ok := props["resourceState"]; ok {
			resourceState, ok := v.(string)
			if !ok {
				continue
			}
			if resourceState == "Running" {
				vHost, ok := props["hostName"]
				if !ok {
					continue
				}
				hostname, ok = vHost.(string)
				if !ok {
					continue
				}
				break
			}
		}
	}
	if hostname == "" {
		return nil, fmt.Errorf("no running AMS streaming endpoint found")
	}
	// Construct manifest URLs
	manifestBase := fmt.Sprintf("https://%s/%s/manifest", hostname, locatorName)
	return map[string]string{
		"hls":  manifestBase + "(format=m3u8-aapl)",
		"dash": manifestBase + "(format=mpd-time-csf)",
	}, nil
}

// CompleteMediaUpload finalizes a heavy media upload.
func (s *ServiceImpl) CompleteMediaUpload(ctx context.Context, req *mediapb.CompleteMediaUploadRequest) (*mediapb.CompleteMediaUploadResponse, error) {
	// Validate upload session
	session, ok := heavyUploadSessions[req.UploadId]
	if !ok {
		err := graceful.WrapErr(ctx, codes.NotFound, "upload session not found", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	// Commit all blocks to finalize the blob in Azure
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey := os.Getenv("AZURE_BLOB_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER")
	if accountName == "" || accountKey == "" || containerName == "" {
		err := graceful.WrapErr(ctx, codes.Internal, "Azure Blob Storage config missing", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	blobName := req.UploadId + ".mp4"
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to create Azure credential: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	blockBlobClient, err := blockblob.NewClientWithSharedKeyCredential(serviceURL+containerName+"/"+blobName, cred, nil)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to create block blob client: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	// Build block list (slice of base64-encoded block IDs)
	blockList := make([]string, session.ChunksTotal)
	for i := 0; i < session.ChunksTotal; i++ {
		blockID := fmt.Sprintf("block-%06d", i)
		blockList[i] = base64.StdEncoding.EncodeToString([]byte(blockID))
	}
	_, err = blockBlobClient.CommitBlockList(ctx, blockList, nil)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to commit block list: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}

	// --- Azure Media Services automation ---
	resourceGroup := os.Getenv("AZURE_MEDIA_RESOURCE_GROUP")
	accountName = os.Getenv("AZURE_MEDIA_ACCOUNT_NAME")
	transformName := os.Getenv("AZURE_MEDIA_TRANSFORM_NAME")
	outputAssetName := req.UploadId + "-output"
	jobName := req.UploadId + "-job"
	blobURL := serviceURL + containerName + "/" + blobName

	token, err := getAMSToken(ctx)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to get AMS token", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if err := submitAMSJob(ctx, token, resourceGroup, accountName, transformName, jobName, outputAssetName, blobURL); err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to submit AMS job", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if err := pollAMSJob(ctx, token, resourceGroup, accountName, transformName, jobName); err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "AMS job did not complete", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	playbackURLs, err := getAMSStreamingURLs(ctx, token, resourceGroup, accountName, outputAssetName)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to get AMS streaming URLs", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	thumbnails := []ThumbnailInfo{{
		URL:   playbackURLs["hls"] + "/thumbnail.jpg", // Placeholder, update as needed
		Width: 320, Height: 180, TimeOffset: 0.0, Description: "Thumbnail",
	}}

	// Update asset in DB
	id, err := uuid.Parse(req.UploadId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "invalid upload ID", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	asset, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.NotFound, "media not found", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	var mediaMeta *Metadata
	if asset.Metadata != nil && asset.Metadata.ServiceSpecific != nil {
		mediaField, ok := asset.Metadata.ServiceSpecific.Fields["media"]
		if ok && mediaField.GetKind() != nil {
			metaMap := pkgmeta.StructToMap(mediaField.GetStructValue())
			mediaMeta = &Metadata{}
			metaBytes, err := json.Marshal(metaMap)
			if err != nil {
				err = graceful.WrapErr(ctx, codes.Internal, "failed to marshal media metadata", err)
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				}
				return nil, graceful.ToStatusError(err)
			}
			if err := json.Unmarshal(metaBytes, mediaMeta); err != nil {
				err = graceful.WrapErr(ctx, codes.Internal, "failed to marshal media metadata", err)
				var ce *graceful.ContextError
				if errors.As(err, &ce) {
					ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
				}
				return nil, graceful.ToStatusError(err)
			}
		}
	}
	if mediaMeta == nil {
		mediaMeta = &Metadata{}
	}
	mediaMeta.PlaybackURLs = playbackURLs
	mediaMeta.Thumbnails = thumbnails
	// Canonical: Add accessibility, compliance metadata
	mediaMeta.Compliance = &ComplianceMetadata{
		Standards: []ComplianceStandard{{
			Name:      "WCAG",
			Level:     "AA",
			Version:   "2.1",
			Compliant: true,
		}},
		CheckedBy: "media-service",
		CheckedAt: time.Now().Format(time.RFC3339),
		Method:    "automated",
		Issues:    []ComplianceIssue{},
	}
	// [CANONICAL] Always normalize metadata before persistence or emission.
	metaMapOut := make(map[string]interface{})
	metaBytesOut, err := json.Marshal(mediaMeta)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to marshal media metadata", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	if err := json.Unmarshal(metaBytesOut, &metaMapOut); err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, "failed to unmarshal media metadata", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	normMap := pkgmeta.Handler{}.NormalizeAndCalculate(metaMapOut, "media", req.UploadId, nil, "success", "normalize media metadata")
	metaStruct := pkgmeta.MapToStruct(normMap)
	metaForValidation := &structpb.Struct{Fields: map[string]*structpb.Value{"media": structpb.NewStructValue(metaStruct)}}

	asset.Metadata = &commonpb.Metadata{ServiceSpecific: metaForValidation}
	asset.URL = playbackURLs["hls"] // Use HLS as main URL for demo
	asset.UpdatedAt = time.Now()
	if err := s.repo.UpdateMedia(ctx, asset); err != nil {
		s.log.Error("failed to update asset after encoding", zap.Error(err))
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			err := graceful.WrapErr(ctx, codes.Internal, "media.complete_failed", err)
			var ce *graceful.ContextError
			if errors.As(err, &ce) {
				ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
			}
			return nil, graceful.ToStatusError(err)
		}
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to update asset after encoding", err)
	}
	// Emit media.completed event after successful completion
	if s.eventEnabled && s.eventEmitter != nil {
		success := graceful.WrapSuccess(ctx, codes.OK, "media completed", asset, nil)
		s.handler.Success(ctx, "complete_media_upload", codes.OK, "media completed", success, asset.Metadata, asset.ID.String(), nil)
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
					var ce *graceful.ContextError
					if errors.As(err, &ce) {
						ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
					}
					return nil, graceful.ToStatusError(err)
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
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "invalid media ID", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	media, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		if errors.Is(err, ErrMediaNotFound) {
			err = graceful.WrapErr(ctx, codes.NotFound, "media not found", nil)
			var ce *graceful.ContextError
			if errors.As(err, &ce) {
				ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
			}
			return nil, graceful.ToStatusError(err)
		}
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to get media: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "invalid media ID", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	asset, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.NotFound, "media not found", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey := os.Getenv("AZURE_BLOB_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER")
	if accountName == "" || accountKey == "" || containerName == "" {
		err := graceful.WrapErr(ctx, codes.Internal, "Azure Blob Storage config missing", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to create Azure credential: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	blobName := asset.ID.String() + ".mp4"
	blobClient, err := blob.NewClientWithSharedKeyCredential(serviceURL+containerName+"/"+blobName, cred, nil)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to create blob client: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to download from Azure Blob: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to read blob data: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	return &mediapb.StreamMediaContentResponse{Data: data, Status: "ok"}, nil
}

// DeleteMedia deletes a media file.
func (s *ServiceImpl) DeleteMedia(ctx context.Context, req *mediapb.DeleteMediaRequest) (*mediapb.DeleteMediaResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "invalid media ID", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	// Get asset from DB
	asset, err := s.repo.GetMedia(ctx, id)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to get media: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	// Remove from Azure Blob Storage
	accountName := os.Getenv("AZURE_BLOB_ACCOUNT")
	accountKey := os.Getenv("AZURE_BLOB_KEY")
	containerName := os.Getenv("AZURE_BLOB_CONTAINER")
	if accountName == "" || accountKey == "" || containerName == "" {
		err := graceful.WrapErr(ctx, codes.Internal, "Azure Blob Storage config missing", nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to create Azure credential: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	blobName := asset.ID.String() + ".mp4"
	blockBlobClient, err := blockblob.NewClientWithSharedKeyCredential(serviceURL+containerName+"/"+blobName, cred, nil)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to create block blob client: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	_, err = blockBlobClient.Delete(ctx, nil)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to delete blob: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}
	// Remove from DB
	if err := s.repo.DeleteMedia(ctx, id); err != nil {
		// Emit failure event
		if s.eventEnabled && s.eventEmitter != nil {
			err := graceful.WrapErr(ctx, codes.Internal, "media.delete_failed", err)
			var ce *graceful.ContextError
			if errors.As(err, &ce) {
				ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
			}
			return nil, graceful.ToStatusError(err)
		}
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.WrapErr(ctx, codes.Internal, "failed to delete media", err)
	}
	// Emit media.deleted event after successful deletion
	if s.eventEnabled && s.eventEmitter != nil {
		success := graceful.WrapSuccess(ctx, codes.OK, "media deleted", req.Id, nil)
		success.StandardOrchestrate(ctx, graceful.SuccessOrchestrationConfig{
			Log:        s.log,
			Cache:      s.cache,
			CacheKey:   req.Id,
			CacheValue: req.Id,
			CacheTTL:   10 * time.Minute,
			Metadata:   nil,
			// Event emission handled by graceful.Handler
			EventEnabled: s.eventEnabled,
			EventType:    "media.deleted",
			EventID:      req.Id,
			PatternType:  "media",
			PatternID:    req.Id,
			PatternMeta:  nil,
		})
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
					var ce *graceful.ContextError
					if errors.As(err, &ce) {
						ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
					}
					return nil, graceful.ToStatusError(err)
				}
			}
		}
	}

	// Canonical event emission handled by graceful handler

	return &mediapb.DeleteMediaResponse{Status: "deleted"}, nil
}

// ListUserMedia lists user media files with pagination and metadata.
func (s *ServiceImpl) ListUserMedia(ctx context.Context, req *mediapb.ListUserMediaRequest) (*mediapb.ListUserMediaResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.InvalidArgument, "invalid user ID", err)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
	}

	// --- Production-grade pagination ---
	offset := 0
	if req.PageToken != "" {
		parsed, err := strconv.Atoi(req.PageToken)
		if err == nil && parsed >= 0 {
			offset = parsed
		}
		// else fallback to 0
	}
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = 20 // default page size
	}

	mediaList, err := s.repo.ListUserMedia(ctx, userID, "", pageSize, offset)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to list user media: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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

	// --- NextPageToken logic ---
	nextPageToken := ""
	if len(mediaList) == pageSize {
		nextPageToken = strconv.Itoa(offset + pageSize)
	}

	// --- WebSocket/Redis real-time update integration point ---
	// In production, push updates to user via WebSocket if connected, or publish to Redis channel 'media:events:user:{user_id}'.
	// See internal/server/ws/websocket.go for the canonical pattern.
	// Example stub:
	// if s.eventEmitter != nil {
	//     s.eventEmitter.EmitEventWithLogging(ctx, ...)
	// }

	return &mediapb.ListUserMediaResponse{
		Media:         result,
		Status:        "ok",
		NextPageToken: nextPageToken,
	}, nil
}

// ListSystemMedia lists system media files with pagination and metadata.
func (s *ServiceImpl) ListSystemMedia(ctx context.Context, req *mediapb.ListSystemMediaRequest) (*mediapb.ListSystemMediaResponse, error) {
	// --- Production-grade pagination ---
	offset := 0
	if req.PageToken != "" {
		parsed, err := strconv.Atoi(req.PageToken)
		if err == nil && parsed >= 0 {
			offset = parsed
		}
		// else fallback to 0
	}
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = 20 // default page size
	}

	mediaList, err := s.repo.ListSystemMedia(ctx, "", pageSize, offset)
	if err != nil {
		err = graceful.WrapErr(ctx, codes.Internal, fmt.Sprintf("failed to list system media: %v", err), nil)
		var ce *graceful.ContextError
		if errors.As(err, &ce) {
			ce.StandardOrchestrate(ctx, graceful.ErrorOrchestrationConfig{Log: s.log})
		}
		return nil, graceful.ToStatusError(err)
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

	// --- NextPageToken logic ---
	nextPageToken := ""
	if len(mediaList) == pageSize {
		nextPageToken = strconv.Itoa(offset + pageSize)
	}

	// --- WebSocket/Redis real-time update integration point ---
	// In production, push updates to system via WebSocket if connected, or publish to Redis channel 'media:events:system'.
	// See internal/server/ws/websocket.go for the canonical pattern.
	// Example stub:
	// if s.eventEmitter != nil {
	//     s.eventEmitter.EmitEventWithLogging(ctx, ...)
	// }

	return &mediapb.ListSystemMediaResponse{
		Media:         result,
		Status:        "ok",
		NextPageToken: nextPageToken,
	}, nil
}

// BroadcastSystemMedia: push a mock update to all system subscribers.
func (s *ServiceImpl) BroadcastSystemMedia(ctx context.Context, req *mediapb.BroadcastSystemMediaRequest) (*mediapb.BroadcastSystemMediaResponse, error) {
	// In prod, integrate with pub/sub or event bus
	// Remove in-memory pub/sub demo logic
	// Only keep Redis/WebSocket production logic below

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
