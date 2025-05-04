package asset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	assetpb "github.com/nmxmxh/master-ovasabi/api/protos/asset/v0"
	assetrepo "github.com/nmxmxh/master-ovasabi/internal/repository/asset"
	"github.com/nmxmxh/master-ovasabi/pkg/cdn/r2"
	"github.com/nmxmxh/master-ovasabi/pkg/redis"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

// AssetService defines the interface for asset operations
type AssetService interface {
	UploadLightAsset(ctx context.Context, req *assetpb.UploadLightAssetRequest) (*assetpb.Asset, error)
	StartHeavyAssetUpload(ctx context.Context, req *assetpb.StartHeavyAssetUploadRequest) (*assetpb.StartHeavyAssetUploadResponse, error)
	StreamAssetChunk(stream assetpb.AssetService_StreamAssetChunkServer) error
	CompleteAssetUpload(ctx context.Context, req *assetpb.CompleteAssetUploadRequest) (*assetpb.Asset, error)
	GetAsset(ctx context.Context, req *assetpb.GetAssetRequest) (*assetpb.Asset, error)
	StreamAssetContent(req *assetpb.GetAssetRequest, stream assetpb.AssetService_StreamAssetContentServer) error
	DeleteAsset(ctx context.Context, req *assetpb.DeleteAssetRequest) (*emptypb.Empty, error)
	ListUserAssets(ctx context.Context, req *assetpb.ListUserAssetsRequest) (*assetpb.ListUserAssetsResponse, error)
	ListSystemAssets(ctx context.Context, req *assetpb.ListSystemAssetsRequest) (*assetpb.ListSystemAssetsResponse, error)
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

// UploadLightAsset handles small asset uploads (< 500KB) and stores them in R2 CDN
func (s *ServiceImpl) UploadLightAsset(ctx context.Context, req *assetpb.UploadLightAssetRequest) (*assetpb.Asset, error) {
	// Validate request
	if !s.validateMimeType(req.MimeType) {
		return nil, status.Error(codes.InvalidArgument, "unsupported MIME type")
	}

	size := int64(len(req.Data))
	if err := s.validateUploadSize(size); err != nil {
		return nil, err
	}

	if size > lightThreshold {
		return nil, status.Error(codes.InvalidArgument, "asset too large for light upload, use heavy upload instead")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	// Calculate checksum for data integrity
	checksum := sha256.Sum256(req.Data)
	checksumHex := hex.EncodeToString(checksum[:])

	// In UploadLightAsset, use strict folder structure and store authenticity fields
	folderPrefix := fmt.Sprintf("nfts/%s/%s", req.UserId, "default") // Replace "default" with collectionID if available
	publicURL, authenticityHash, r2Key, err := r2.UploadFile(ctx, req.Data, req.Name, req.MimeType, folderPrefix)
	if err != nil {
		s.log.Error("failed to upload to R2 CDN", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to upload to CDN")
	}

	asset := &assetrepo.AssetModel{
		ID:               uuid.New(),
		UserID:           userID,
		Type:             assetrepo.StorageTypeLight,
		Name:             req.Name,
		MimeType:         req.MimeType,
		Size:             size,
		URL:              publicURL,
		Checksum:         checksumHex,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		AuthenticityHash: authenticityHash,
		R2Key:            r2Key,
		// Signature:   ... (optional, sign authenticityHash with server key)
	}

	if err := s.repo.CreateAsset(ctx, asset); err != nil {
		s.log.Error("failed to create light asset",
			zap.Error(err),
			zap.String("userId", userID.String()),
			zap.Int64("size", size),
		)
		return nil, status.Error(codes.Internal, "failed to create asset")
	}

	return s.assetToProto(asset), nil
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
func (s *ServiceImpl) StreamAssetChunk(stream assetpb.AssetService_StreamAssetChunkServer) error {
	ctx := stream.Context()
	var (
		currentAsset *assetrepo.AssetModel
		metadata     *UploadMetadata
		buffer       []byte
		hasher       = sha256.New()
		retryCount   = 0
	)

	// Set timeout for chunk processing
	chunkCtx, cancel := context.WithTimeout(ctx, chunkTimeout)
	defer cancel()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Error(codes.Internal, "failed to receive chunk")
		}

		// Initialize upload session
		if currentAsset == nil {
			uploadID, err := uuid.Parse(chunk.UploadId)
			if err != nil {
				return status.Error(codes.InvalidArgument, "invalid upload ID")
			}

			// Get upload metadata
			if err := s.cache.Get(ctx, "upload_metadata", uploadID.String(), &metadata); err != nil {
				return status.Error(codes.Internal, "upload session not found or expired")
			}

			asset, err := s.repo.GetAsset(ctx, uploadID)
			if err != nil {
				return status.Error(codes.Internal, "failed to get asset")
			}
			if asset == nil {
				return status.Error(codes.NotFound, "asset not found")
			}

			currentAsset = asset
			buffer = make([]byte, 0, metadata.Size)
		}

		// Validate chunk sequence
		if int(chunk.Sequence) >= metadata.ChunksTotal {
			return status.Error(codes.InvalidArgument, "invalid chunk sequence")
		}

		// Update chunk data
		buffer = append(buffer, chunk.Data...)
		hasher.Write(chunk.Data)
		metadata.ChunksReceived++
		metadata.LastUpdate = time.Now()

		// Update upload metadata in cache
		if err := s.cache.Set(chunkCtx, "upload_metadata", metadata.ID.String(), metadata, uploadTimeout); err != nil {
			s.log.Warn("failed to update upload metadata",
				zap.Error(err),
				zap.String("uploadId", metadata.ID.String()),
			)
		}

		// Process accumulated chunks
		if len(buffer) >= largeChunkSize || metadata.ChunksReceived == metadata.ChunksTotal {
			currentAsset.Checksum = hex.EncodeToString(hasher.Sum(nil))
			currentAsset.UpdatedAt = time.Now()

			if err := s.repo.UpdateAsset(chunkCtx, currentAsset); err != nil {
				retryCount++
				if retryCount >= maxRetries {
					return status.Errorf(codes.Internal, "failed to update asset after %d retries", maxRetries)
				}
				continue
			}

			buffer = buffer[:0]
			hasher.Reset()
			retryCount = 0
		}
	}

	// Verify upload completion
	if metadata.ChunksReceived != metadata.ChunksTotal {
		return status.Error(codes.FailedPrecondition, "incomplete upload: missing chunks")
	}

	return nil
}

// CompleteAssetUpload finalizes a heavy asset upload
func (s *ServiceImpl) CompleteAssetUpload(ctx context.Context, req *assetpb.CompleteAssetUploadRequest) (*assetpb.Asset, error) {
	uploadID, err := uuid.Parse(req.UploadId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid upload ID")
	}

	// Get upload metadata
	var metadata UploadMetadata
	if err := s.cache.Get(ctx, "upload_metadata", uploadID.String(), &metadata); err != nil {
		return nil, status.Error(codes.Internal, "upload session not found or expired")
	}

	asset, err := s.repo.GetAsset(ctx, uploadID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get asset")
	}
	if asset == nil {
		return nil, status.Error(codes.NotFound, "asset not found")
	}

	// Verify upload completion
	if metadata.ChunksReceived != metadata.ChunksTotal {
		return nil, status.Error(codes.FailedPrecondition, "incomplete upload: missing chunks")
	}

	// Clean up upload metadata
	if err := s.cache.Delete(ctx, "upload_metadata", uploadID.String()); err != nil {
		s.log.Warn("failed to clean up upload metadata",
			zap.Error(err),
			zap.String("uploadId", uploadID.String()),
		)
	}

	// Update asset metadata from metadata
	asset.AuthenticityHash = metadata.Checksum
	asset.R2Key = metadata.Checksum
	asset.UpdatedAt = time.Now()

	if err := s.repo.UpdateAsset(ctx, asset); err != nil {
		s.log.Error("failed to update asset", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update asset")
	}

	return s.assetToProto(asset), nil
}

// GetAsset retrieves an asset by ID
func (s *ServiceImpl) GetAsset(ctx context.Context, req *assetpb.GetAssetRequest) (*assetpb.Asset, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid asset ID")
	}

	// Try cache first
	var protoAsset assetpb.Asset
	if err := s.cache.Get(ctx, "asset", id.String(), &protoAsset); err == nil {
		return &protoAsset, nil
	}

	asset, err := s.repo.GetAsset(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get asset")
	}
	if asset == nil {
		return nil, status.Error(codes.NotFound, "asset not found")
	}

	result := s.assetToProto(asset)

	// Cache the result with TTL
	if err := s.cache.Set(ctx, "asset", id.String(), result, time.Hour); err != nil {
		s.log.Warn("failed to cache asset", zap.Error(err))
	}

	// In GetAsset, if asset is NFT/private, generate signed URL
	// Example:
	// if asset.IsNFT || asset.IsPrivate {
	//   signedURL, err := r2.GenerateSignedURL(asset.R2Key, 15*time.Minute)
	//   asset.URL = signedURL
	// }

	return result, nil
}

// StreamAssetContent streams the content of a stored asset from R2 in chunks via gRPC
func (s *ServiceImpl) StreamAssetContent(req *assetpb.GetAssetRequest, stream assetpb.AssetService_StreamAssetContentServer) error {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid asset ID")
	}

	asset, err := s.repo.GetAsset(stream.Context(), id)
	if err != nil {
		return status.Error(codes.Internal, "failed to get asset")
	}
	if asset == nil {
		return status.Error(codes.NotFound, "asset not found")
	}
	if asset.R2Key == "" {
		return status.Error(codes.NotFound, "asset R2 key not found")
	}

	// Stream from R2 using the asset's R2Key
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucket := os.Getenv("R2_BUCKET")
	endpoint := os.Getenv("R2_ENDPOINT")
	region := os.Getenv("R2_REGION")
	if region == "" {
		region = "auto"
	}

	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return status.Error(codes.Internal, "failed to create AWS session")
	}

	svc := s3.New(sess)
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(asset.R2Key),
	}
	obj, err := svc.GetObjectWithContext(stream.Context(), input)
	if err != nil {
		return status.Error(codes.Internal, "failed to fetch from R2")
	}
	defer obj.Body.Close()

	buf := make([]byte, 512*1024) // 512KB chunks
	sequence := uint32(0)
	for {
		n, err := obj.Body.Read(buf)
		if n > 0 {
			chunk := &assetpb.AssetChunk{
				UploadId: asset.ID.String(),
				Data:     buf[:n],
				Sequence: sequence,
			}
			if err := stream.Send(chunk); err != nil {
				return status.Error(codes.Internal, "failed to send chunk")
			}
			sequence++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Error(codes.Internal, "stream read error")
		}
	}
	return nil
}

// DeleteAsset deletes an asset by ID
func (s *ServiceImpl) DeleteAsset(ctx context.Context, req *assetpb.DeleteAssetRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid asset ID")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	asset, err := s.repo.GetAsset(ctx, id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get asset")
	}
	if asset == nil {
		return nil, status.Error(codes.NotFound, "asset not found")
	}

	if asset.UserID != userID && !asset.IsSystem {
		return nil, status.Error(codes.PermissionDenied, "unauthorized to delete asset")
	}

	if err := s.repo.DeleteAsset(ctx, id); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete asset")
	}

	// Clear cache
	if err := s.cache.Delete(ctx, "asset", id.String()); err != nil {
		s.log.Warn("failed to delete asset from cache", zap.Error(err))
	}

	return &emptypb.Empty{}, nil
}

// ListUserAssets lists assets for a user with pagination
func (s *ServiceImpl) ListUserAssets(ctx context.Context, req *assetpb.ListUserAssetsRequest) (*assetpb.ListUserAssetsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user ID")
	}

	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 50 // default page size
	}
	if pageSize > 100 { // max page size
		pageSize = 100
	}

	// Try cache first
	var response assetpb.ListUserAssetsResponse
	cacheKey := fmt.Sprintf("%s:%s", userID.String(), req.PageToken)
	if err := s.cache.Get(ctx, "user_assets", cacheKey, &response); err == nil {
		return &response, nil
	}

	offset := 0
	if req.PageToken != "" {
		if _, err := fmt.Sscanf(req.PageToken, "%d", &offset); err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid page token")
		}
	}

	assets, err := s.repo.ListUserAssets(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list assets")
	}

	protoAssets := make([]*assetpb.Asset, len(assets))
	for i, asset := range assets {
		protoAssets[i] = s.assetToProto(asset)
	}

	nextPageToken := ""
	if len(assets) == pageSize {
		nextPageToken = fmt.Sprintf("%d", offset+pageSize)
	}

	result := &assetpb.ListUserAssetsResponse{
		Assets:        protoAssets,
		NextPageToken: nextPageToken,
	}

	// Cache the result with TTL
	if err := s.cache.Set(ctx, "user_assets", cacheKey, result, time.Hour); err != nil {
		s.log.Warn("failed to cache user assets list", zap.Error(err))
	}

	return result, nil
}

// ListSystemAssets lists system assets with pagination
func (s *ServiceImpl) ListSystemAssets(ctx context.Context, req *assetpb.ListSystemAssetsRequest) (*assetpb.ListSystemAssetsResponse, error) {
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 50 // default page size
	}
	if pageSize > 100 { // max page size
		pageSize = 100
	}

	// Try cache first
	var response assetpb.ListSystemAssetsResponse
	cacheKey := fmt.Sprintf("size_%d:%s", pageSize, req.PageToken)
	if err := s.cache.Get(ctx, "system_assets", cacheKey, &response); err == nil {
		return &response, nil
	}

	offset := 0
	if req.PageToken != "" {
		if _, err := fmt.Sscanf(req.PageToken, "%d", &offset); err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid page token")
		}
	}

	assets, err := s.repo.ListSystemAssets(ctx, pageSize, offset)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list system assets")
	}

	protoAssets := make([]*assetpb.Asset, len(assets))
	for i, asset := range assets {
		protoAssets[i] = s.assetToProto(asset)
	}

	nextPageToken := ""
	if len(assets) == pageSize {
		nextPageToken = fmt.Sprintf("%d", offset+pageSize)
	}

	result := &assetpb.ListSystemAssetsResponse{
		Assets:        protoAssets,
		NextPageToken: nextPageToken,
	}

	// Cache the result with TTL
	if err := s.cache.Set(ctx, "system_assets", cacheKey, result, time.Hour); err != nil {
		s.log.Warn("failed to cache system assets list", zap.Error(err))
	}

	return result, nil
}

// Helper function to convert repository AssetModel to proto Asset
func (s *ServiceImpl) assetToProto(a *assetrepo.AssetModel) *assetpb.Asset {
	if a == nil {
		return nil
	}

	asset := &assetpb.Asset{
		Id:        a.ID.String(),
		UserId:    a.UserID.String(),
		Name:      a.Name,
		MimeType:  a.MimeType,
		Size:      a.Size,
		IsSystem:  a.IsSystem,
		CreatedAt: timestamppb.New(a.CreatedAt),
		UpdatedAt: timestamppb.New(a.UpdatedAt),
	}

	if a.DeletedAt != nil {
		asset.DeletedAt = timestamppb.New(*a.DeletedAt)
	}

	// Set type and data/url based on asset type
	if a.Type == assetrepo.StorageTypeLight {
		asset.Type = assetpb.AssetType_ASSET_TYPE_LIGHT
		asset.Url = a.URL
	} else {
		asset.Type = assetpb.AssetType_ASSET_TYPE_HEAVY
	}

	return asset
}
