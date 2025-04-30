# Package asset

## Types

### AssetService

AssetService defines the interface for asset operations

### ServiceImpl

ServiceImpl implements the AssetService interface

#### Methods

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

StreamAssetContent streams the content of an asset

##### UploadLightAsset

UploadLightAsset handles small asset uploads (< 500KB) stored directly in DB

### UploadMetadata

UploadMetadata stores upload session information
