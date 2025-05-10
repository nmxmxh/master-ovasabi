# Package asset

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

##### BroadcastSystemAsset

BroadcastSystemAsset broadcasts a system asset.

##### CompleteAssetUpload

CompleteAssetUpload finalizes a heavy asset upload.

##### DeleteAsset

DeleteAsset deletes an asset.

##### GetAsset

GetAsset retrieves an asset by ID.

##### ListSystemAssets

ListSystemAssets lists system assets with pagination.

##### ListUserAssets

ListUserAssets lists assets for a user with pagination.

##### StartHeavyAssetUpload

StartHeavyAssetUpload initiates a chunked upload for large assets.

##### StreamAssetChunk

StreamAssetChunk handles streaming chunks for heavy asset uploads.

##### StreamAssetContent

StreamAssetContent streams the content of a stored asset from R2 in chunks via gRPC.

##### SubscribeToSystemAssets

SubscribeToSystemAssets subscribes to system assets.

##### SubscribeToUserAssets

SubscribeToUserAssets subscribes to user assets.

##### UploadLightAsset

UploadLightAsset handles small asset uploads (< 500KB).

### UploadMetadata

UploadMetadata stores upload session information.
