# Package server

Package server provides gRPC server implementation with monitoring, logging, and tracing
capabilities.

## Security Enforcement via gRPC Interceptor

- All unary gRPC requests are intercepted by SecurityUnaryServerInterceptor.
- The interceptor resolves SecurityService from the DI container for each request.
- It calls Authorize (with an empty request for now) before allowing the request to proceed.

  - If not authorized, the request is denied with PermissionDenied.

- After the handler executes, RecordAuditEvent is called for audit logging.
- This ensures all services are monitored and enforced by SecurityService at the gRPC layer.
- When the proto is updated with more fields, the interceptor can extract and populate them from the
  request/context.

This approach centralizes security, reduces boilerplate in each service, and ensures consistent
enforcement and auditability across the platform.

## Variables

### ErrServiceDegraded

Common errors.

## Types

### CampaignWebSocketBus

### KGHooks

KGHooks manages real-time knowledge graph updates.

#### Methods

##### Start

Start begins processing knowledge graph updates.

##### Stop

Stop gracefully shuts down the hooks.

### KGService

KGService manages the knowledge graph service.

#### Methods

##### IsDegraded

IsDegraded returns whether the service is in degraded mode.

##### PublishUpdate

PublishUpdate sends an update to the knowledge graph.

##### RecoverFromDegradedMode

RecoverFromDegradedMode attempts to recover the service from degraded mode.

##### RegisterService

RegisterService registers a new service with the knowledge graph.

##### Start

Start initializes the knowledge graph service.

##### Stop

Stop gracefully shuts down the service.

##### UpdateRelation

UpdateRelation updates a relation in the knowledge graph.

##### UpdateSchema

UpdateSchema updates the schema for a service.

### KGUpdate

KGUpdate represents a knowledge graph update event.

### KGUpdateType

KGUpdateType represents the type of knowledge graph update.

### Server

#### Methods

##### Start

##### Stop

### WebSocketBus

### WebSocketEvent

## Functions

### RegisterAllServices

RegisterAllServices registers all gRPC services with the server.

### RegisterMediaUploadHandlers

RegisterMediaUploadHandlers registers all media upload endpoints to the mux.

### RegisterWebSocketHandlers

### Run

Run starts the main server, including gRPC, health, and metrics endpoints.

### SecurityUnaryServerInterceptor

SecurityUnaryServerInterceptor creates a new unary server interceptor that logs request details and
checks authorization.

### StartHTTPServer

StartHTTPServer sets up and starts the HTTP server in a goroutine.

### StreamServerInterceptor

StreamServerInterceptor creates a new stream server interceptor that logs stream details.

### UnaryServerInterceptor

UnaryServerInterceptor creates a new unary server interceptor that logs request details.

### WaitForShutdown
