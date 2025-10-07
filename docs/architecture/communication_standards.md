# OVASABI Platform Communication Standards

> **Version:** 2025-10-06
>
> **Audience:** All Engineers and Architects
>
> **Reference:** See [Amadeus Context](../amadeus/amadeus_context.md) for the system-wide orchestration and knowledge graph patterns that these standards support.

---

## 1. Overview

This document defines the **single, unified communication standard** for all service and client interactions on the OVASABI platform. It establishes a hybrid architectural pattern that combines the directness of **Remote Procedure Calls (RPC)** with the ubiquity and web-friendliness of **REST and WebSockets**.

The core principles are:
- **A Unified Command Endpoint:** All state-changing operations are treated as RPC-style "actions," sent as JSON payloads to a single, dynamic HTTP endpoint.
- **A Real-Time Event Bus:** A WebSocket-based system provides real-time feedback, streaming, and notifications using a strictly-defined, canonical event structure.
- **Metadata-Driven Orchestration:** All communication is enriched with metadata, enabling dynamic routing, feature flagging, and deep integration with the Amadeus knowledge graph.

These standards are the foundation of our "API Factory" model, ensuring that all services—current and future—are interoperable, observable, scalable, and provide a consistent developer experience.

---

## 2. Architectural Philosophy: RPC-Style Commands over Web Protocols

To understand our approach, it's useful to compare it with traditional API paradigms.

| Paradigm | Style | Pros | Cons |
| :--- | :--- | :--- | :--- |
| **REST** | **Resource-Oriented (Nouns)** | Standard, stateless, great for public-facing CRUD APIs. | Can be rigid and verbose for complex, action-based workflows. Leads to endpoint sprawl. |
| **gRPC (RPC)** | **Action-Oriented (Verbs)** | High-performance, strongly-typed, ideal for internal service-to-service communication. | Less web-native; gRPC-Web has limitations and doesn't support true bidirectional streaming. |
| **Our Hybrid** | **Action-Oriented over HTTP/WS** | Combines RPC's directness with the web's native protocols. Single endpoint, flexible, real-time. | Requires a smart gateway and disciplined adherence to the standard. |

Our system adopts an **RPC-style philosophy** but implements it over standard web protocols. Instead of creating dozens of resource-specific REST endpoints (e.g., `POST /quotes`, `POST /users`), we treat every operation as a procedure call.

This provides the conceptual clarity of RPC (`commerce.createQuote({...})`) while avoiding the client-side complexity of gRPC-Web. It is the key to bridging our high-performance gRPC backend with any web or mobile client, seamlessly.

---

## 3. The Unified Communication Pattern

### 3.1. Command Endpoint (HTTP)

All write operations, commands, and actions MUST be sent to a single, unified endpoint. This endpoint acts as the primary ingress for client-initiated actions.

- **Endpoint:** `POST /api/v1/execute`
- **Request Envelope:** All requests MUST use the following JSON structure:
  ```json
  {
    "action": "service.method",
    "payload": { ... },
    "metadata": { ... }
  }
  ```
  - **`action`**: A string identifying the target service and method (e.g., `commerce.createQuote`). This is the "Remote Procedure Call."
  - **`payload`**: An object containing the specific data for the action.
  - **`metadata`**: An object containing contextual information for routing, orchestration, and analytics (see Section 6).

### 3.2. Real-Time Event Bus (WebSocket)

The WebSocket connection is the primary channel for real-time, server-to-client communication.

- **Endpoint:** `/ws/v1`
- **Unified Event Envelope:** All messages (from server to client) MUST use the canonical event structure:
  ```json
  {
    "event": "service:action:v1:state",
    "payload": { ... },
    "metadata": { ... }
  }
  ```
  - **`event`**: The canonical event type (see Section 4).
  - **`payload`**: The data associated with the event.
  - **`metadata`**: Contextual information, including a `correlation_id` linking the event to the initial command.

---

## 4. Canonical Event Naming Standard

To ensure consistency, observability, and predictability, all events, channels, and cache keys MUST use the following standardized format.

**Format:** `{service}:{action}:v{version}:{state}`

- **`service`**: Normalized, lowercase service name (e.g., `commerce`, `user`).
- **`action`**: Snake_case action/method name (e.g., `create_quote`, `get_profile`).
- **`version`**: API/service version (e.g., `v1`).
- **`state`**: A controlled term describing the event's position in the action lifecycle.

### 4.1. Controlled State Vocabulary

All event `state` fields MUST use one of the following values:

| State | Description |
| :--- | :--- |
| `requested` | An action has been accepted by the system and is queued for processing. |
| `started` | A worker has begun processing the action. |
| `success` | The action completed successfully. The payload contains the result. |
| `failed` | The action failed. The payload contains error details. |
| `completed` | A terminal state, often used for workflows to signal the entire process is finished. |

This strict naming convention allows for powerful, automated tooling, monitoring, and creates a predictable, observable lifecycle for every action in the system.

---

## 5. Event Routing: System, Campaign, and User Scope

The WebSocket gateway and Nexus route events based on the **presence of `campaign_id` and `user_id` fields in the event payload**, not the event name itself. This enables generic, extensible routing for all canonical event types.

| Event Scope | `campaign_id` Present? | `user_id` Present? | Gateway Broadcast | Recipients |
| :--- | :--- | :--- | :--- | :--- |
| **System-wide** | ❌ | ❌ | `broadcastSystem` | All connected clients |
| **Campaign-specific** | ✅ | ❌ | `broadcastCampaign` | All clients in that campaign |
| **User-specific** | (any) | ✅ | `broadcastUser` | Only that specific user |

This design decouples event creation from routing logic. New services and event types require no changes to the gateway, ensuring scalability.

---

## 6. Metadata-Driven Orchestration

The `metadata` object is present in every command and event and is critical for orchestration.

- **All payloads MUST include `metadata`**.
- It is used for: scheduling, feature flags, custom business rules, audit trails, A/B testing, and service-specific extensions.
- The Nexus and Amadeus systems use this metadata to route, compose logic, and enrich the platform's knowledge graph.

---

## 7. End-to-End Example Flow: Creating a Quote

1.  **Client Sends Command:** The client sends a command to the unified HTTP endpoint.
    ```http
    POST /api/v1/execute
    Content-Type: application/json

    {
      "action": "commerce.createQuote",
      "payload": {
        "amount": 100,
        "currency": "USD",
        "product_id": "prod_123"
      },
      "metadata": {
        "user_id": "user_abc",
        "campaign_id": "camp_xyz",
        "correlation_id": "uuid-1"
      }
    }
    ```

2.  **API Gateway & Nexus:**
    - The gateway accepts the request and immediately returns a `202 Accepted`.
    - It publishes a `commerce:create_quote:v1:requested` event to the internal message bus.

3.  **Commerce Service:**
    - Subscribed to `*:*:v1:requested` events, it picks up the job.
    - It begins processing, and may optionally emit a `commerce:create_quote:v1:started` event.
    - Upon successful processing, it emits the `commerce:create_quote:v1:success` event with the result.

4.  **WebSocket Gateway:**
    - It is subscribed to all `*:*:v1:success` and `*:*:v1:failed` events.
    - It receives the `success` event and inspects its `metadata`.
    - Seeing `user_id: "user_abc"`, it uses the `broadcastUser` function to send the event only to that user's WebSocket connection.

5.  **Client Receives Real-Time Event:** The client's UI receives the event and updates reactively.
    ```json
    {
      "event": "commerce:create_quote:v1:success",
      "payload": {
        "quote_id": "quote_789",
        "expires_at": "2025-10-06T22:00:00Z"
      },
      "metadata": {
        "user_id": "user_abc",
        "campaign_id": "camp_xyz",
        "correlation_id": "uuid-1"
      }
    }
    ```

---

## 8. Enforcement & Best Practices

- **Source of Truth:** All valid event types are generated from service registration and proto definitions. Use these generated constants.
- **Validation:** CI pipelines and service startup routines MUST validate that all subscribed and emitted event types are registered.
- **Observability:** All logs MUST include the full event type and `correlation_id`.
- **Security:** Use prefixes and namespaces to prevent collisions. Access control should restrict which services can publish/subscribe to which event channels.
- **Compliance:** All new code and services MUST comply with these standards.

