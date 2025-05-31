# Package errors

## Variables

### ErrUserNotFound

### ErrInterfaceMustBePointer

DI container errors.

## Functions

### LogWithError

LogWithError logs the error with context and returns a wrapped error. Use this for standardized
error logging across services.

### New

New creates a new error with the given message.

### Wrap

Wrap wraps an error with additional context.
