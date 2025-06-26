# OVASABI Event Metadata Pattern

All event payloads in the OVASABI platform **must** use the shared `common.Metadata` struct, following these rules:

## 1. Use `common.Metadata` for All Events
- Every event emitted to Nexus must include a `common.Metadata` payload.
- This struct provides audit, trace, and compliance fields.

## 2. Service-Specific Fields
- Service-specific fields must be namespaced under `metadata.service_specific.{service}`.
- Example:
  ```json
  {
    "audit": { ... },
    "service_specific": {
      "user": { "role": "admin" },
      "media": { "stream_id": "abc123" }
    }
  }
  ```

## 3. Versioning and Compliance
- Always include version and audit fields where possible.
- Use the metadata pattern for all cross-service communication, orchestration, and event payloads.

## 4. Example EventRequest
```go
req := &nexusv1.EventRequest{
    EventType: "user.created",
    EntityId: userID,
    Metadata: &commonpb.Metadata{
        Audit: &structpb.Struct{Fields: ...},
        ServiceSpecific: &structpb.Struct{
            Fields: map[string]*structpb.Value{
                "user": structpb.NewStructValue(&structpb.Struct{Fields: ...}),
            },
        },
    },
}
```

## 5. Enforcement
- All new and existing services must follow this pattern for emitting and consuming events.
- Code reviews and tests should check for compliance.
