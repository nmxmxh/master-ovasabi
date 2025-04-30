# Package server

Package server provides gRPC server implementation with monitoring, logging, and tracing
capabilities.

## Functions

### RegisterServices

RegisterServices registers all gRPC services with the server.

### StreamServerInterceptor

StreamServerInterceptor creates a new stream server interceptor that logs stream details.

### UnaryServerInterceptor

UnaryServerInterceptor creates a new unary server interceptor that logs request details.
