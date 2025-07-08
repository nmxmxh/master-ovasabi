# Communication & Event Naming Standards

## Overview

This document defines the canonical standards for event, channel, and key naming across all communication layers in the OVASABI platform. These standards ensure consistency, observability, and maintainability for all services, including their integration with the `graceful` orchestration framework.

---

## 1. Canonical Event Type Format

All events, channels, and keys MUST use the following format:

```text
{service}:{action}:v{version}:{state}
```

- **service**: The normalized service name (e.g., `search`, `user`, `media`).
- **action**: The snake_case method or action name (e.g., `search`, `suggest`, `create_user`).
- **version**: The API/service version (e.g., `v1`).
- **state**: One of a controlled set of states (see below).

### Example Event Types

- `search:search:v1:requested`
- `search:search:v1:success`
- `search:suggest:v1:failed`
- `user:create_user:v1:completed`

---

## 2. Controlled State Vocabulary

All event types and keys must use one of the following states:

- `requested`
- `started`
- `success`
- `failed`
- `completed`

---

## 3. Channel and Key Usage

- **Event Bus (Nexus/Redis/NATS):**
  - Use the canonical event type as the channel/topic name.
- **Redis Keys:**
  - Use `{event_type}:{id}` for data keys (e.g., `search:search:v1:success:12345`).
  - Use the event type alone for pub/sub channels.
- **WebSocket:**
  - Use the event type for all ingress/egress event messages.
- **gRPC:**
  - Map gRPC methods to `{service}:{action}:v{version}`; state is inferred from response or error.

---

## 4. Source of Truth & Code Generation

- All valid event types and key patterns are generated from service registration/proto definitions.
- Go/TypeScript constants and JSON/Markdown docs are auto-generated for use in code and documentation.
- All event emission, subscription, and key usage must use these generated constants.

---

## 5. Validation & Linting

- At build/startup, validate that all emitted/subscribed event types and Redis keys are in the registry.
- CI linter checks for naming convention violations.

---

## 6. Observability & Tracing

- All logs must include the full event type/key.
- Correlation IDs must be included in all event payloads for end-to-end tracing.

---

## 7. Backward Compatibility

- Legacy event/key names must be aliased or migrated to the canonical format.

---

## 8. Security & Access Control

- Use prefixes/namespaces to prevent cross-service collisions.
- Restrict which services can publish/subscribe to which event types/channels.

---

## 9. Integration with `graceful`

- All orchestration, error handling, and state transitions in `graceful` must use canonical event types and key patterns.
- Graceful workflows should emit and listen for events using the `{service}:{action}:v{version}:{state}` format.

---

## 10. Example: Search Service

- **gRPC:** `SearchService.Search` → `search:search:v1:requested`
- **Event Bus:** Publishes `search:search:v1:success` on completion
- **Redis:** Stores results at `search:search:v1:success:{search_id}`
- **WebSocket:** Sends `search:search:v1:completed` to client

---

## 11. Auto-Generated Reference

- See `events/constants.go` and `events/event_types.json` for the full list of valid event types and key patterns.

---

## 12. Enforcement

- All new code and services must comply with these standards.
- PRs will be rejected if they introduce non-canonical event types or key patterns.

---

## 13. Event Routing: System, Campaign, and User Scope

### WebSocket Gateway & Nexus Event Routing

The ws-gateway and Nexus distinguish between system-wide, campaign-specific, and user-specific events based on the presence of `campaign_id` and `user_id` fields in the event payload. This enables generic, extensible routing for all canonical event types.

| Event Scope        | `campaign_id` | `user_id` | Gateway Broadcast Function | Recipients         |
|--------------------|--------------|-----------|---------------------------|--------------------|
| System-wide        | ❌           | ❌        | broadcastSystem           | All clients        |
| Campaign-specific  | ✅           | ❌        | broadcastCampaign         | All in campaign    |
| User-specific      | (any)        | ✅        | broadcastUser             | Only that user     |

- **System events:** Omit both `campaign_id` and `user_id` in the payload. All connected clients receive the event.
- **Campaign events:** Include `campaign_id` (string), omit `user_id`. Only clients in the specified campaign receive the event.
- **User events:** Include `user_id` (string). Only the specified user receives the event, regardless of campaign.

> **Note:** The canonical event type (`{service}:{action}:v{version}:{state}`) does not determine routing. Routing is based solely on the presence of these fields in the payload, allowing all event types to be handled generically.

### Implementation Guidance

- **Nexus:** When emitting events, set the correct fields in the payload for the intended audience.
- **ws-gateway:** Inspects the payload and calls the appropriate broadcast function. No per-event-type logic is required.
- **Extensibility:** New services/actions do not require gateway changes; only the payload structure matters for routing.

---

*This document is integral to the operation of all services and the graceful orchestration framework. All contributors must read and follow these standards.*
