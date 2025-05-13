# Communication Standards for OVASABI Platform

> **Reference:** See [Amadeus Context](../amadeus/amadeus_context.md) for system-wide architecture,
> metadata, and service integration patterns.

---

## Overview

This document defines the **central communication standards** for all application-based interactions
on the OVASABI platform. It establishes:

- A unified, dynamic REST API pattern for all updates and commands
- A real-time WebSocket event system for feedback and streaming
- Metadata-driven extensibility for campaign, user, and system context
- Integration guidance for streaming and media uploading services

These standards ensure all services—current and future—are interoperable, extensible, and compliant
with the Amadeus knowledge graph and orchestration system.

---

## 1. Central Communication Pattern

### REST API (Dynamic, Metadata-Driven)

- **Single endpoint:** `/api/{campaign}/update` for all user/system actions
- **Dynamic fields:** `data` and `metadata` objects allow flexible, campaign- and user-specific
  payloads
- **Action routing:** The `action` field determines backend logic (e.g., `submit_quote`, `register`,
  `upload_media`, `start_stream`)
- **Extensible:** New actions and data fields can be added without new endpoints

### WebSocket Event System

- **Endpoint:** `/ws/{campaign}/{user_id}`
- **Event types:** `update_ack`, `campaign_event`, `stats_update`, `media_event`, `stream_event`,
  `error`, `notification`, etc.
- **Real-time feedback:** All updates, notifications, and streaming events are pushed to clients
- **Correlation:** REST actions and WebSocket events are linked by user/session/campaign context

### Metadata-Driven Orchestration

- **All payloads include `metadata`:** Used for scheduling, features, custom rules, audit, tags, and
  service-specific extensions
- **Nexus orchestration:** Backend routes and composes logic based on metadata, campaign, and user
  context
- **Knowledge graph integration:** All communication is tracked and enriched in Amadeus

---

## 2. OpenAPI Specification (REST)

See [OpenAPI Spec Example](#openapi-spec-example) below for a canonical definition. All REST
endpoints must:

- Accept `user_id`, `action`, `data`, and `metadata`
- Respond with dynamic fields based on action/campaign
- Use JWT or session authentication

---

## 3. WebSocket Event Schema

- All clients connect to `/ws/{campaign}/{user_id}`
- Events are JSON objects with a `type` field and dynamic payload
- Event types include:
  - `update_ack`: Acknowledgement of REST action
  - `campaign_event`: Campaign-specific events (e.g., referral milestones)
  - `stats_update`: Real-time stats (active users, leaderboards)
  - `media_event`: Media upload/progress/completion notifications
  - `stream_event`: Streaming status, errors, or data
  - `error`: Error messages
  - `notification`: General notifications

---

## 4. Streaming and Media Uploading Services

### Streaming Services

- **Initiation:**
  - Client sends a REST `POST /api/{campaign}/update` with `action: start_stream` and relevant
    `data`/`metadata`
  - Backend processes, allocates resources, and responds with stream info (e.g., stream ID,
    endpoints)
  - WebSocket event (`stream_event`) notifies client of stream status, errors, or data
- **Real-Time Data:**
  - All real-time stream updates (e.g., viewer count, stream health, chat) are pushed via WebSocket
    as `stream_event`
  - Clients can send control commands (e.g., stop, pause) via REST or a dedicated WebSocket message
    (if bi-directional)
- **Extensibility:**
  - New stream types or features are added by extending the `action` and `data` fields, and
    documenting new `stream_event` payloads

### Media Uploading Services

- **Initiation:**
  - Client sends a REST `POST /api/{campaign}/update` with `action: upload_media` and file metadata
    in `data`/`metadata`
  - Backend responds with upload URL or instructions (e.g., pre-signed S3 URL)
- **Upload Progress:**
  - Client uploads media directly (e.g., to object storage)
  - Backend pushes `media_event` via WebSocket for progress, completion, or errors
- **Completion:**
  - On upload completion, backend processes media (e.g., transcoding, validation)
  - Client receives `media_event` updates (e.g., `processing`, `ready`, `failed`)
- **Extensibility:**
  - Support for new media types, processing steps, or notifications is added by extending the
    `action`, `data`, and `media_event` schema

---

## 5. Example OpenAPI Spec

```
openapi: 3.0.0
info:
  title: OVASABI Central Communication API
  version: 1.0.0
paths:
  /api/{campaign}/update:
    post:
      summary: Submit an update or command for a campaign
      parameters:
        - in: path
          name: campaign
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                user_id:
                  type: string
                action:
                  type: string
                data:
                  type: object
                metadata:
                  type: object
              required:
                - user_id
                - action
                - data
      responses:
        '200':
          description: Dynamic response
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                  result:
                    type: object
                  message:
                    type: string
```

---

## 6. Example WebSocket Event Types

- **update_ack**
  ```json
  {
    "type": "update_ack",
    "action": "upload_media",
    "status": "success",
    "result": { "media_id": "m_123" }
  }
  ```
- **media_event**
  ```json
  { "type": "media_event", "media_id": "m_123", "status": "processing", "progress": 42 }
  { "type": "media_event", "media_id": "m_123", "status": "ready", "url": "https://..." }
  ```
- **stream_event**
  ```json
  { "type": "stream_event", "stream_id": "s_456", "status": "started", "viewers": 10 }
  { "type": "stream_event", "stream_id": "s_456", "status": "ended" }
  ```

---

## 7. Extensibility & Best Practices

- **All new features** (streaming, media, campaigns, etc.) use the same REST/WebSocket pattern
- **Document new actions and event types** in this standard and in the Amadeus context
- **Use metadata** for all extensible fields and orchestration
- **Integrate with Amadeus/Nexus** for registration, orchestration, and knowledge graph enrichment
- **Secure all endpoints** (JWT/session for REST, token/cookie for WebSocket)
- **Monitor and log** all communication for observability and compliance

---

## 8. References

- [Amadeus Context](../amadeus/amadeus_context.md)
- [Service Implementation Pattern](../services/implementation_pattern.md)
- [OpenAPI Specification](https://swagger.io/specification/)
- [AsyncAPI (for WebSocket/Event Docs)](https://www.asyncapi.com/)
