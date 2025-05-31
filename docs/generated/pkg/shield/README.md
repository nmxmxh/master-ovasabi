# Package shield

## Variables

### ErrUnauthenticated

Custom error types for clear error handling.

## Types

### Option

Option type for functional options pattern.

## Functions

### AuthInterceptor

AuthInterceptor returns a gRPC unary server interceptor that checks permissions using
shield.CheckPermission.

### AuthorizationMiddleware

AuthorizationMiddleware returns an HTTP middleware that checks permissions using
shield.CheckPermission.

### BuildRequestMetadata

BuildRequestMetadata constructs a \*commonpb.Metadata from HTTP request, userID, and guest status.

### CheckPermission

CheckPermission performs authorization checks using the platform's AuthorizeRequest pattern.
