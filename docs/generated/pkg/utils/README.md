# Package utils

## Constants

### ContextRolesKey

ContextRolesKey is the key for the authenticated user roles in the context.

### ContextUserIDKey

ContextUserIDKey is the key for the authenticated user ID in the context.

### DefaultTimeout

DefaultTimeout is the default timeout for operations.

## Variables

### BufferPool

## Functions

### BatchProcess

BatchProcess runs a function on each item in batches with parallelism.

### ContextWithCustomTimeout

ContextWithCustomTimeout creates a context with a custom timeout.

### ContextWithDeadline

ContextWithDeadline creates a context with a deadline.

### ContextWithTimeout

ContextWithTimeout creates a context with the default timeout.

### GenerateTestEmail

GenerateTestEmail generates a unique test email address.

### GenerateTestName

GenerateTestName generates a test user name.

### GenerateTestPassword

GenerateTestPassword generates a test password.

### GetAuthenticatedUserID

GetAuthenticatedUserID retrieves the authenticated user ID from the context.

### GetAuthenticatedUserRoles

GetAuthenticatedUserRoles retrieves the authenticated user roles from the context.

### GetBuffer

GetBuffer retrieves a buffer from the pool.

### GetByteSlice

GetByteSlice retrieves a byte slice from the pool.

### GetContextFields

GetContextFields extracts common fields from context for logging and error context.

### GetValue

GetValue retrieves a value from the context with type safety.

### IsAdmin

IsAdmin checks if the given roles include the "admin" role.

### IsServiceAdmin

IsServiceAdmin checks if the given roles include the global admin or a service-specific admin role.

### MergeContexts

MergeContexts creates a new context that is canceled when any of the input contexts are canceled.

### NewUUID

NewUUID generates a new UUIDv7 (time-based).

### NewUUIDOrDefault

NewUUIDOrDefault generates a new UUIDv7 (time-based) or returns a default if generation fails.

### ParseUUID

ParseUUID parses a UUID string into a UUID object.

### PutBuffer

PutBuffer returns a buffer to the pool.

### PutByteSlice

PutByteSlice returns a byte slice to the pool.

### StreamItems

StreamItems streams items from a channel to a callback, with cancellation support.

### ToBigInt

ToBigInt safely converts an int to a \*big.Int.

### ToBigInt64

ToBigInt64 safely converts an int64 to a \*big.Int.

### ToInt32

ToInt32 safely converts an int to int32, clamping to the int32 range.

### ValidateUUID

ValidateUUID checks if a string is a valid UUID.

### WithValue

WithValue adds a value to the context with type safety.
