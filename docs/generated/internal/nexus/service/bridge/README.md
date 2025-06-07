# Package bridge

## Types

### Adapter

Adapter defines the interface for all protocol adapters (MQTT, AMQP, WebSocket, etc.)

### AdapterConfig

### AdapterFactory

### AdapterPool

#### Methods

##### Get

##### Put

### Container

Container provides the bridge container orchestration and health endpoints.

#### Methods

##### Start

Start runs the bridge container, serving health and metrics endpoints and handling graceful
shutdown.

### ErrorEvent

### Event

### EventBus

### HealthStatus

### Message

### MessageHandler

### MetadataMatcher

#### Methods

##### Matches

Matches checks if the metadata matches the rule's matcher.

### Metrics

### MetricsCollector

### Router

#### Methods

##### Route

Route routes a message to the correct adapter based on metadata and routing rules.

##### RouteAsync

RouteAsync routes a message asynchronously and returns a channel for the error result.

### RoutingRule

#### Methods

##### Matches

Matches checks if the rule's matcher matches the given metadata.

### Service

Service provides the Nexus bridge orchestration and protocol adapter logic.

#### Methods

##### HandleInboundMessage

For adapters to push inbound messages to the event bus.

## Functions

### AuthorizeTransport

AuthorizeTransport checks RBAC/authorization using canonical metadata.

### LogTransportEvent

LogTransportEvent logs transport events for audit purposes.

### RegisterAdapter

RegisterAdapter registers a new protocol adapter and updates the knowledge graph.

### VerifySenderIdentity

VerifySenderIdentity checks the sender's identity using canonical metadata.

# WebSocket Endpoint: /ws

## Overview

The /ws endpoint provides a unified, real-time event stream for frontend, WASM, and edge clients. It
is fully integrated with the Nexus event bus and supports per-client event type filters and payload
format negotiation.

## Connecting

- **URL:** `ws://<host>:8090/ws`
- **Query Parameters:**
  - `filters`: Comma-separated list of event types to subscribe to (e.g.,
    `filters=search,messaging,content`)
  - `format`: Payload format, either `json` (default) or `protobuf`

**Example:**

```
ws://localhost:8090/ws?filters=search,messaging&format=json
```

## Negotiation

- On connect, the server parses the `filters` and `format` query parameters.
- Only events matching the specified types are sent to the client.
- Payloads are encoded as JSON or protobuf, as requested.

## Supported Event Types

- `search`
- `messaging`
- `content`
- `talent`
- `product`
- `campaign`

## Message Envelope

All messages use the unified event envelope:

```
{
  "type": "search" | "messaging" | "content" | "talent" | "product" | "campaign",
  "payload": { ... },
  "metadata": { ... }
}
```

- `type`: Event type
- `payload`: Event-specific data (see service docs)
- `metadata`: Canonical metadata (see Robust Metadata Pattern)

## Emitting Events

Clients can emit events by sending a message in the same envelope format. The server will emit the
event to the Nexus event bus, and all subscribed services/clients will receive it if they match the
filters.

## Receiving Events

Clients receive events matching their filters, encoded in their preferred format.

## WASM/Frontend Integration

- Use a WebSocket client to connect to `/ws` with the desired filters and format.
- Parse incoming messages according to the negotiated format.
- To emit an event, send a message in the unified envelope format.

## References

- See `docs/amadeus/amadeus_context.md` for the canonical metadata pattern and event envelope.
- See `internal/nexus/service/bridge/adapters/websocket.go` for implementation details.
