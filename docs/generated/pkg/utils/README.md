# Package utils

## Constants

### DefaultTimeout

DefaultTimeout is the default timeout for operations.

## Variables

### BufferPool

## Functions

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

### GetBuffer

GetBuffer retrieves a buffer from the pool.

### GetByteSlice

GetByteSlice retrieves a byte slice from the pool.

### GetValue

GetValue retrieves a value from the context with type safety.

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
