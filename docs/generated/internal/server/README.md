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

## Types

### Server

#### Methods

##### Start

##### Stop

### ServiceRegistration

## Functions

### ContextInjectionMiddleware

HTTP middleware to inject request ID, trace ID, and feature flags into context

### ContextInjectionUnaryInterceptor

gRPC interceptor to inject request ID, trace ID, and feature flags into context

### RegisterAllServices

RegisterAllServices registers all gRPC services with the server.

### Run

Run starts the main server, including gRPC, health, and metrics endpoints.

### SecurityUnaryServerInterceptor

SecurityUnaryServerInterceptor enforces security and audit logging for all gRPC requests.

Best Practice Pathway:

1. Extract user/session info, method, and resource from context/request if available.
2. Prepare AuthorizeRequest with real data as soon as proto supports it.
3. Only call AuditEvent after the handler, and only if the request was authorized and handled.
4. Populate AuditEvent with as much context as possible: service, method, principal, resource,
   status, error, timestamp.
5. If authorization fails, do not call the handler or audit event.
6. If audit logging fails, log a warning but do not fail the request.
7. If guest_mode is detected, assign diminished responsibilities/permissions.
8. Minimize allocations and logging overhead in the hot path.
9. Add clear comments for future extensibility and best practices.

### StreamServerInterceptor

StreamServerInterceptor creates a new stream server interceptor that logs stream details.

### UnaryServerInterceptor

UnaryServerInterceptor creates a new unary server interceptor that logs request details.

### WaitForShutdown
