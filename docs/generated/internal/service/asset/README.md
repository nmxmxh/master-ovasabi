# Package asset

## Types

### AssetBroadcaster

#### Methods

##### Publish

##### Subscribe

##### Unsubscribe

### AssetService

AssetService defines the interface for asset operations

### ServiceImpl

ServiceImpl implements the AssetService interface

#### Methods

##### BroadcastAssetChunk

BroadcastAssetChunk allows publishing a live asset chunk to all subscribers (for live streaming)

##### CompleteAssetUpload

CompleteAssetUpload finalizes a heavy asset upload

##### DeleteAsset

DeleteAsset deletes an asset by ID

##### GetAsset

GetAsset retrieves an asset by ID

##### ListSystemAssets

ListSystemAssets lists system assets with pagination

##### ListUserAssets

ListUserAssets lists assets for a user with pagination

##### StartHeavyAssetUpload

StartHeavyAssetUpload initiates a chunked upload for large assets

##### StreamAssetChunk

StreamAssetChunk handles streaming chunks for heavy asset uploads

##### StreamAssetContent

StreamAssetContent streams the content of a stored asset from R2 in chunks via gRPC

##### UploadLightAsset

UploadLightAsset handles small asset uploads (< 500KB) and stores them in R2 CDN

### UploadMetadata

UploadMetadata stores upload session information
