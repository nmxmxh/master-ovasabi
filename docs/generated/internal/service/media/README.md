# Package media

## Types

### Broadcaster

Broadcaster struct.

#### Methods

##### Publish

##### Subscribe

##### Unsubscribe

### Service

Service defines the interface for asset operations.

### ServiceImpl

ServiceImpl implements the Service interface.

#### Methods

##### BroadcastAssetChunk

BroadcastAssetChunk allows publishing a live asset chunk to all subscribers (for live streaming).

##### BroadcastSystemMedia

BroadcastSystemMedia broadcasts a system media file.

##### CompleteMediaUpload

CompleteMediaUpload finalizes a heavy media upload.

##### DeleteMedia

DeleteMedia deletes a media file.

##### GetMedia

GetMedia retrieves a media file.

##### ListSystemMedia

ListSystemMedia lists system media files.

##### ListUserMedia

ListUserMedia lists user media files.

##### StartHeavyMediaUpload

StartHeavyMediaUpload initiates a chunked upload for large media.

##### StreamMediaChunk

StreamMediaChunk handles streaming chunks for heavy media uploads.

##### StreamMediaContent

StreamMediaContent streams the content of a media file.

##### SubscribeToSystemMedia

SubscribeToSystemMedia subscribes to system media updates.

##### SubscribeToUserMedia

SubscribeToUserMedia subscribes to user media updates.

##### UploadLightMedia

UploadLightMedia handles small media uploads (< 500KB).

### UploadMetadata

UploadMetadata stores upload session information.
