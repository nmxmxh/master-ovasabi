# Package ws

## Variables

### SystemAggMu

Add at file scope:.

## Types

### CampaignWebSocketBus

### Client

#### Methods

##### Send

Send returns the send channel for this client (for external use).

### ClientMap

#### Methods

##### Delete

##### Load

##### Range

##### Store

### WebSocketBus

### WebSocketEvent

## Functions

### RegisterWebSocketHandlers

# WebSocket Event Bus: Unified, Event-Driven Pattern (2024+)

## Overview

The WebSocket server now implements a **unified, event-driven architecture** fully integrated with
the Nexus event bus. All WebSocket clients (browser, WASM, mobile, edge, etc.) are both emitters and
receivers of events. This enables real-time, bi-directional orchestration and a true "nervous
system" for the platform.

## Unified Event Envelope

All WebSocket messages (from client or server) use:

```json
{
  "type": "event_type",
  "payload": { ... },
  "metadata": { ... }
}
```

- `type`: Canonical event type (see Nexus event types)
- `payload`: Event-specific data
- `metadata`: Canonical metadata (see Robust Metadata Pattern)

## Client Participation

- **Emitters:** Any client can send events (e.g., search, actions, commands) via WebSocket.
- **Receivers:** Any client can receive events (e.g., search results, campaign updates,
  notifications).
- **Event routing:** The WebSocket server emits all incoming events to the Nexus event bus, and
  routes all relevant events from the bus to the appropriate clients.
- **Loose coupling:** All communication is event-driven and metadata-rich, enabling orchestration,
  automation, and extensibility.

## Example Flow

1. Client sends a `search` event via WebSocket.
2. WebSocket server emits `search.requested` to Nexus.
3. Search service processes and emits `search.completed`.
4. WebSocket server routes result to client.

## Extensibility

- New event types and workflows can be added without changing endpoints or protocol.
- All events are tracked, versioned, and auditable via metadata and the knowledge graph.

**This is now the canonical pattern for all real-time, event-driven communication in OVASABI.**
