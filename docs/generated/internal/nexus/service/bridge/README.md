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

AuthorizeTransport enforces RBAC for transport actions.

### LogTransportEvent

LogTransportEvent logs transport events for audit purposes.

### RegisterAdapter

RegisterAdapter registers a new protocol adapter and updates the knowledge graph.

### VerifySenderIdentity

VerifySenderIdentity verifies the digital signature of a message sender.
