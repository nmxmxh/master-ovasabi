# Package media

## Variables

### ErrMediaNotFound

### MediaEventRegistry

## Types

### AccessibilityMetadata

### Broadcaster

Broadcaster struct.

#### Methods

##### Publish

##### Subscribe

##### Unsubscribe

### CaptionTrack

### ComplianceIssue

### ComplianceMetadata

### ComplianceStandard

### EventEmitter

EventEmitter defines the interface for emitting events (canonical platform interface).

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Metadata

### Model

### Repo

Repo implements Repository.

#### Methods

##### CreateMedia

CreateMedia creates a new media.

##### DeleteMedia

DeleteMedia deletes media by ID.

##### GetMedia

GetMedia retrieves media by ID.

##### ListSystemMedia

ListSystemMedia retrieves system media with pagination and optional master_id filter.

##### ListUserMedia

ListUserMedia retrieves media for a user with pagination and optional master_id filter.

##### UpdateMedia

UpdateMedia updates an existing media.

### Repository

Repository defines the interface for media operations.

### Service

Service defines the interface for asset operations.

### ServiceImpl

ServiceImpl implements the Service interface.

#### Methods

##### BroadcastAssetChunk

BroadcastAssetChunk allows publishing a live asset chunk to all subscribers (for live streaming).

##### BroadcastSystemMedia

BroadcastSystemMedia: push a mock update to all system subscribers.

##### CompleteMediaUpload

CompleteMediaUpload finalizes a heavy media upload.

##### DeleteMedia

DeleteMedia deletes a media file.

##### GetMedia

GetMedia retrieves a media file.

##### ListSystemMedia

ListSystemMedia lists system media files with pagination and metadata.

##### ListUserMedia

ListUserMedia lists user media files with pagination and metadata.

##### StartHeavyMediaUpload

StartHeavyMediaUpload initiates a chunked upload for large media.

##### StreamMediaChunk

StreamMediaChunk handles streaming chunks for heavy media uploads.

##### StreamMediaContent

StreamMediaContent streams the content of a media file.

##### UploadChunks

UploadChunks uploads media chunks concurrently with retry and timeout logic. It uses maxRetries,
uploadTimeout, chunkTimeout, and maxConcurrentUploadChunks constants.

##### UploadLightMedia

UploadLightMedia handles small media uploads (< 500KB).

### StorageType

### ThumbnailInfo

### TranslationTrack

### UploadMetadata

UploadMetadata stores upload session information.

## Functions

### NewMediaClient

NewMediaClient creates a new gRPC client connection and returns a MediaServiceClient and a cleanup
function.

### NewService

NewService constructs a new MediaServiceServer instance with event bus support.

### Register

Register registers the media service with the DI container and event bus support.

### StartEventSubscribers
