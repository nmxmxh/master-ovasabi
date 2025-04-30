# Asset Service

The Asset service manages all 3D assets, their metadata, storage, and delivery in the OVASABI
platform.

## Architecture

```mermaid
graph TD
    A[gRPC API] -->|Implements| B[Service Layer]
    B -->|Uses| C[Repository Layer]
    C -->|Writes| D[Postgres]
    C -->|Caches| E[Redis]
    C -->|Stores| F[Object Storage]
    G[Master Repository] -->|References| D
    H[Event Log] -->|Records| D
    I[Auth Service] -->|Validates| B
    J[User Service] -->|References| B
```

## Features

1. **Asset Management**

   - Asset upload and storage
   - Metadata management
   - Version control
   - Asset retrieval

2. **Storage Optimization**

   - Smart storage selection
   - Size-based routing
   - Format optimization
   - Compression management

3. **Delivery System**

   - CDN integration
   - Caching strategy
   - Progressive loading
   - Streaming support

4. **Asset Processing**
   - Format conversion
   - Optimization pipeline
   - Thumbnail generation
   - Quality control

## API Reference

### Proto Definition

```protobuf
service AssetService {
    rpc UploadAsset(UploadAssetRequest) returns (AssetResponse);
    rpc GetAsset(GetAssetRequest) returns (AssetResponse);
    rpc ListAssets(ListAssetsRequest) returns (ListAssetsResponse);
    rpc UpdateAsset(UpdateAssetRequest) returns (AssetResponse);
    rpc DeleteAsset(DeleteAssetRequest) returns (DeleteAssetResponse);
    rpc OptimizeAsset(OptimizeAssetRequest) returns (AssetResponse);
    rpc GenerateThumbnail(GenerateThumbnailRequest) returns (ThumbnailResponse);
}
```

### Methods

#### UploadAsset

Uploads a new 3D asset with smart storage selection.

```go
func (s *service) UploadAsset(ctx context.Context, req *pb.UploadAssetRequest) (*pb.AssetResponse, error)
```

#### OptimizeAsset

Optimizes an existing asset for better performance.

```go
func (s *service) OptimizeAsset(ctx context.Context, req *pb.OptimizeAssetRequest) (*pb.AssetResponse, error)
```

## Data Model

### Asset Model

```go
type AssetModel struct {
    ID           uuid.UUID
    UserID       uuid.UUID
    Name         string
    Type         string // inline, external
    ByteData     []byte           // only if type = inline
    URL          string           // only if type = external
    SizeBytes    int64
    MimeType     string
    Format       string           // gltf, obj, fbx, etc.
    Metadata     map[string]interface{} // JSONB in DB
    Versions     []AssetVersion
    ThumbnailURL string
    Status       string // processing, ready, failed
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type AssetVersion struct {
    Version     int
    URL         string
    SizeBytes   int64
    Format      string
    CreatedAt   time.Time
    IsOptimized bool
}
```

### Database Schema

```sql
CREATE TABLE assets (
    id UUID PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id),
    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    type TEXT CHECK (type IN ('inline', 'external')),
    byte_data BYTEA,
    url TEXT,
    size_bytes BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    format TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    thumbnail_url TEXT,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE asset_versions (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id),
    version INT NOT NULL,
    url TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    format TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_optimized BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_assets_user_id ON assets(user_id);
CREATE INDEX idx_assets_type ON assets(type);
CREATE INDEX idx_assets_format ON assets(format);
CREATE INDEX idx_assets_status ON assets(status);
CREATE INDEX idx_assets_metadata ON assets USING gin(metadata);
CREATE INDEX idx_asset_versions_asset_id ON asset_versions(asset_id);
```

## Knowledge Graph

### Capabilities

```go
type AssetCapabilities struct {
    AssetManagement struct {
        Upload       bool `json:"upload"`
        Storage      bool `json:"storage"`
        Retrieval    bool `json:"retrieval"`
        Versioning   bool `json:"versioning"`
    } `json:"asset_management"`

    StorageOptimization struct {
        SmartRouting  bool `json:"smart_routing"`
        Compression   bool `json:"compression"`
        Deduplication bool `json:"deduplication"`
        FormatOpt     bool `json:"format_opt"`
    } `json:"storage_optimization"`

    DeliverySystem struct {
        CDN          bool `json:"cdn"`
        Caching      bool `json:"caching"`
        Progressive  bool `json:"progressive"`
        Streaming    bool `json:"streaming"`
    } `json:"delivery_system"`

    Processing struct {
        Conversion   bool `json:"conversion"`
        Optimization bool `json:"optimization"`
        Thumbnails   bool `json:"thumbnails"`
        QualityCheck bool `json:"quality_check"`
    } `json:"processing"`
}
```

### Growth Patterns

```mermaid
graph TD
    A[Basic Storage] -->|Add Optimization| B[Smart Storage]
    B -->|Add Analytics| C[Intelligent Storage]
    C -->|Add ML| D[Predictive Storage]

    E[Basic Delivery] -->|Add CDN| F[Fast Delivery]
    F -->|Add Progressive| G[Smart Delivery]
    G -->|Add Analytics| H[Optimized Delivery]

    I[Basic Processing] -->|Add Pipeline| J[Smart Processing]
    J -->|Add Quality| K[Quality Processing]
    K -->|Add AI| L[AI Processing]

    M[Basic Formats] -->|Add Conversion| N[Multi Format]
    N -->|Add Optimization| O[Optimized Format]
    O -->|Add Auto| P[Auto Format]
```

### Evolution Tracking

```go
type AssetEvolution struct {
    StorageCapabilities    []string `json:"storage_capabilities"`
    DeliveryCapabilities   []string `json:"delivery_capabilities"`
    ProcessingCapabilities []string `json:"processing_capabilities"`
    FormatSupport         []string `json:"format_support"`
    Version              string    `json:"version"`
    LastUpdated          time.Time `json:"last_updated"`
}
```

## Caching Strategy

### Key Structure

- Asset Metadata: `cache:asset:metadata:{asset_id}`
- Asset Content: `cache:asset:content:{asset_id}`
- Asset Versions: `cache:asset:versions:{asset_id}`
- User Assets: `cache:asset:user:{user_id}`
- Thumbnails: `cache:asset:thumbnail:{asset_id}`

### TTL Values

- Metadata: 1 hour
- Content: 24 hours
- Versions: 6 hours
- User Assets: 30 minutes
- Thumbnails: 24 hours

## Error Handling

### Error Types

1. **UploadError**

   - Invalid file type
   - Size limit exceeded
   - Storage failure
   - Processing error

2. **RetrievalError**

   - Asset not found
   - Access denied
   - Storage error
   - Format error

3. **ProcessingError**
   - Conversion failed
   - Optimization failed
   - Quality check failed
   - Version conflict

## Monitoring

### Metrics

1. **Storage Metrics**

   - Total storage used
   - Storage by type
   - Upload success rate
   - Processing time

2. **Performance Metrics**
   - Download speed
   - Cache hit ratio
   - CDN performance
   - Error rates

## Security

1. **Upload Security**

   - File validation
   - Size limits
   - Format verification
   - Malware scanning

2. **Access Control**
   - Role-based access
   - User ownership
   - Download limits
   - IP verification

## Future Improvements

1. **Phase 1 - Enhanced Processing**

   - Advanced format support
   - Batch processing
   - Custom optimization
   - Quality profiles

2. **Phase 2 - Smart Delivery**

   - Dynamic CDN routing
   - Smart caching
   - Predictive loading
   - Bandwidth optimization

3. **Phase 3 - AI Features**
   - Auto-optimization
   - Content analysis
   - Quality prediction
   - Format suggestion

## Dependencies

- Auth Service (v1)
- User Service (v1)
- Redis Cache
- Postgres Database
- Object Storage (S3/Supabase)
- CDN Provider

## Configuration

```yaml
asset:
  database:
    pool_size: 20
    max_idle: 10
    max_lifetime: 1h
  cache:
    ttl:
      metadata: 1h
      content: 24h
      versions: 6h
      user_assets: 30m
      thumbnails: 24h
  storage:
    max_inline_size: 500000 # 500KB
    allowed_formats: ['gltf', 'glb', 'obj', 'fbx']
    max_file_size: 100000000 # 100MB
  cdn:
    enabled: true
    provider: 'cloudfront'
    domain: 'assets.ovasabi.com'
  processing:
    thumbnail_sizes: ['128x128', '256x256', '512x512']
    optimization_quality: 85
    max_versions: 5
```

## Deployment

### Resource Requirements

- CPU: 8 cores
- Memory: 16GB
- Storage: 500GB
- Cache: 8GB Redis
- Object Storage: 1TB

### Environment Variables

```bash
ASSET_DB_URL=postgres://user:pass@host:5432/db
ASSET_REDIS_URL=redis://host:6379
ASSET_LOG_LEVEL=info
ASSET_API_PORT=50055
ASSET_STORAGE_KEY=your-storage-key
ASSET_CDN_KEY=your-cdn-key
```
