# OVASABI/Amadeus Campaign Event Envelope Spec

## Overview

This document defines the canonical JSON envelope for all campaign-driven events in the OVASABI/Amadeus platform. All user/system actions, UI events, media actions, uploads, and state changes are sent from the frontend to the campaign orchestrator (and back) using this envelope. This ensures unified, auditable, and extensible orchestration across the system.

---

## Envelope Structure

```json
{
  "type": "campaign_event",           // Always "campaign_event"
  "campaign_id": "string",            // Campaign or flow ID
  "user": {
    "user_id": "string",              // Authenticated user or "guest"
    "session_id": "string",           // Session or device ID
    "roles": ["string"]               // User roles (optional)
  },
  "event": "string",                 // Event name/type (e.g., "content.create", "media.play", "file.upload")
  "payload": { ... },                  // Event-specific data (arbitrary JSON)
  "metadata": { ... },                 // Extensible metadata (versioning, context, etc.)
  "timestamp": "ISO8601 string",      // Event creation time (UTC)
  "request_id": "string"              // Optional: for tracing/correlation
}
```

---

## Field Documentation

- **type**: Always set to `"campaign_event"` for routing and validation.
- **campaign_id**: The unique ID of the campaign, experiment, or flow this event belongs to.
- **user**:
  - **user_id**: The user's unique ID, or "guest" if unauthenticated.
  - **session_id**: The session or device identifier (for tracking, analytics, and guest flows).
  - **roles**: (Optional) List of user roles (e.g., `["admin", "participant"]`).
- **event**: The event name/type, namespaced by domain (e.g., `"content.create"`, `"media.play"`, `"file.upload"`).
- **payload**: Arbitrary JSON object with event-specific data (e.g., content fields, media info, file metadata).
- **metadata**: Extensible object for versioning, context, feature flags, etc. (see platform metadata pattern).
- **timestamp**: ISO8601 UTC timestamp of event creation (set by frontend or orchestrator).
- **request_id**: (Optional) Unique ID for tracing/correlation (set by frontend or orchestrator).

---

## Example: Content Creation Event

```json
{
  "type": "campaign_event",
  "campaign_id": "spring_sale_2025",
  "user": {
    "user_id": "user_123",
    "session_id": "sess_abc456",
    "roles": ["participant"]
  },
  "event": "content.create",
  "payload": {
    "title": "My First Post",
    "body": "Hello, world!",
    "tags": ["intro", "welcome"]
  },
  "metadata": {
    "versioning": {
      "system_version": "2.0.0",
      "service_version": "1.2.0",
      "environment": "beta"
    },
    "feature_flags": ["rich_text"]
  },
  "timestamp": "2025-06-04T12:00:00Z",
  "request_id": "req_789xyz"
}
```

---

## Example: Media Play Event

```json
{
  "type": "campaign_event",
  "campaign_id": "music_festival_2025",
  "user": {
    "user_id": "guest",
    "session_id": "sess_guest_001"
  },
  "event": "media.play",
  "payload": {
    "media_id": "track_42",
    "position": 0
  },
  "metadata": {
    "versioning": {
      "system_version": "2.0.0",
      "service_version": "1.2.0",
      "environment": "prod"
    }
  },
  "timestamp": "2025-06-04T12:01:00Z"
}
```

---

## Usage Notes

- All frontend-to-backend and backend-to-frontend flows use this envelope.
- The orchestrator (campaign service) validates, enriches, and routes events to Nexus and services.
- All responses/events from backend to frontend should use the same envelope (with result or event data in `payload`).
- This envelope supports all flows: UI actions, media, uploads, analytics, etc.

---

## Versioning & Extensibility

- The `metadata` field should always include a `versioning` object as per platform standards.
- Additional fields can be added as needed for new features, campaigns, or flows.

---

## References

- [docs/amadeus/amadeus_context.md](amadeus_context.md)
- [docs/services/metadata.md](../services/metadata.md)
- [docs/services/versioning.md](../services/versioning.md)
