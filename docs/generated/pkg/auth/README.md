# Package auth

JWT middleware for net/http. Depends on context.go in the same package.

## Types

### Context

## Functions

### HasRole

HasRole checks if the current user has the given role.

### HasScope

HasScope checks if the current user has the given scope.

### JWTMiddleware

JWTMiddleware is a minimal HTTP middleware for JWT auth.

### NewContext

NewContext returns a new context with the given AuthContext.
