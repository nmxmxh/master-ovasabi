# 3D Asset Storage & Delivery Practices

> **NOTE:** This document defines the required instructions, rules, and best practices for all 3D
> asset storage, retrieval, and delivery in this project. Contributors and tools (including AI
> assistants) must read and follow these rules before integrating or modifying 3D asset workflows.

---

## General Principles

1. **Choose storage based on asset size and usage.**
   - Small, frequently accessed, or transactional assets: store as `BYTEA` in Postgres.
   - Large, high-res, or static assets: store in external object storage (e.g., S3, Supabase, CDN)
     and reference by URL.
2. **Never store large binary assets in JSONB.**
   - Use `BYTEA` for binary, `JSONB` for metadata or scene graphs only.
3. **Optimize for delivery and caching.**
   - Use CDNs for large/external assets.
   - Use in-memory or browser cache for small/inlined assets.
4. **Support both inline and external assets.**
   - Design schema and APIs to handle both types seamlessly.
5. **Store metadata for all assets.**
   - Always record size, type, MIME, and creation time.

---

## Storage Patterns

| Use Case                                  | Storage      | Why                                      |
| ----------------------------------------- | ------------ | ---------------------------------------- |
| Small meshes, icons, modular geometry     | Postgres     | Fast, transactional, low overhead        |
| User-generated, frequently updated assets | Postgres     | Transactional, easy to version           |
| Large scenes, photoreal models, textures  | External CDN | Scalable, cacheable, reduces DB bloat    |
| Scene graphs, compositions                | JSONB        | Efficient querying/filtering, not binary |

---

## Schema Example

```sql
CREATE TABLE assets (
  id SERIAL PRIMARY KEY,
  name TEXT,
  type TEXT CHECK (type IN ('inline', 'external')),
  byte_data BYTEA,          -- only if type = 'inline'
  url TEXT,                 -- only if type = 'external'
  size_bytes INTEGER,
  mime_type TEXT,
  created_at TIMESTAMPTZ DEFAULT now()
);
```

- Add a generated column for easy filtering:

  ```sql
  ALTER TABLE assets ADD COLUMN is_lightweight BOOLEAN GENERATED ALWAYS AS (size_bytes < 500000) STORED;
  ```

---

## Process

1. **On Upload:**
   - If asset size < 500KB, store as `BYTEA` in Postgres.
   - If asset size ≥ 500KB, upload to external storage, store URL in DB.
   - Always record metadata (size, MIME, etc).
2. **On Retrieval:**
   - For `inline` assets, serve as binary with correct MIME type.
   - For `external` assets, return the URL for direct loading.
3. **On Frontend (React/Three.js):**
   - Use `GLTFLoader` or appropriate loader.
   - For `inline` assets, use `URL.createObjectURL` to create a blob URL.
   - For `external` assets, load directly from the URL (CDN/S3).
4. **On Deletion:**
   - Remove DB record.
   - If external, also delete from object storage.

---

## High-Performance File Uploader in Go

Inspired by
[Building a High-Performance File Uploader in Go](https://medium.com/@souravchoudhary0306/building-a-high-performance-file-uploader-in-go-e812076d598c):

- Use Go's `http` package with streaming to handle large files efficiently.
- Avoid loading the entire file into memory; use `io.Copy` or buffered reads/writes.
- Validate file size, type, and content before accepting uploads.
- For large files, upload directly to object storage (S3, etc.) using multipart upload.
- For small files, read into memory and store as `BYTEA` in Postgres.
- Use goroutines and channels for concurrent uploads if needed.

**Example Go Handler Skeleton:**

```go
func uploadAssetHandler(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form
    err := r.ParseMultipartForm(10 << 20) // 10MB max memory
    if err != nil {
        // handle error (see error handling section)
        return
    }
    file, handler, err := r.FormFile("asset")
    if err != nil {
        // handle error
        return
    }
    defer file.Close()

    // Validate file type/size
    if handler.Size > maxAllowedSize {
        // handle error
        return
    }

    // Decide storage method
    if handler.Size < 500_000 {
        // Read into []byte and store in Postgres
    } else {
        // Stream to S3 or external storage
    }
}
```

---

## Advanced Error Handling for Asset Uploads

Inspired by
[Advanced Error Handling in Go](https://medium.com/@UsamahJ/advanced-error-handling-in-go-9ab6aeca08ee):

- Use custom error types for different error scenarios (validation, storage, network, etc.).
- Wrap errors with context using `fmt.Errorf("context: %w", err)`.
- Log errors with enough context for debugging, but return user-friendly messages to clients.
- Use error codes or types to distinguish between client and server errors.
- For API responses, return structured error objects (JSON) with code, message, and details.

**Example Error Handling Pattern:**

```go
type AssetUploadError struct {
    Code    string
    Message string
    Err     error
}

func (e *AssetUploadError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func wrapError(code, msg string, err error) *AssetUploadError {
    return &AssetUploadError{Code: code, Message: msg, Err: err}
}

// Usage in handler
if err != nil {
    log.Errorf("upload failed: %v", err)
    apiError := wrapError("UPLOAD_FAILED", "Failed to upload asset", err)
    http.Error(w, apiError.Error(), http.StatusInternalServerError)
    return
}
```

---

## Security & Performance

1. **Do not expose direct DB access for asset retrieval.**  
   Always use an API endpoint to serve or proxy assets.
2. **Set appropriate CORS and cache headers** for asset endpoints.
3. **Monitor asset size and usage.**  
   Regularly review for oversized or unused assets.
4. **Version assets if needed** for cache busting and rollback.

---

## Additional Recommendations

1. **Prefer external storage for anything user-uploaded or >1MB.**
2. **Use signed URLs or access control for private assets.**
3. **Document asset types and loader requirements in API docs.**
4. **Automate cleanup of orphaned assets in both DB and storage.**
5. **Test asset delivery performance from multiple regions.**

---

## Example: React/Three.js Loader Logic

```js
const loadAsset = asset => {
  const loader = new GLTFLoader();
  const source =
    asset.type === 'inline'
      ? URL.createObjectURL(new Blob([asset.byte_data], { type: asset.mime_type }))
      : asset.url;
  loader.load(source, gltf => scene.add(gltf.scene));
};
```

---

## Summary Table

| Principle    | Practice                                      |
| ------------ | --------------------------------------------- |
| Small assets | Store as BYTEA in Postgres                    |
| Large assets | Store in external storage, reference by URL   |
| Metadata     | Always record size, type, MIME, created_at    |
| Security     | Serve via API, not direct DB access           |
| Performance  | Use CDN, cache headers, and efficient loaders |

---

**Every 3D asset workflow must be reviewed for compliance with these practices.  
If in doubt, consult a senior engineer or architect before proceeding.**
