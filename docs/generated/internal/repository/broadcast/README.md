# Package repository

## Variables

### ErrBroadcastNotFound

## Types

### Broadcast

Broadcast represents a broadcast message in the service_broadcast table

### BroadcastRepository

BroadcastRepository handles operations on the service_broadcast table

#### Methods

##### Create

Create inserts a new broadcast record

##### Delete

Delete removes a broadcast and its master record

##### GetByID

GetByID retrieves a broadcast by ID

##### List

List retrieves a paginated list of broadcasts

##### ListPending

ListPending retrieves a list of pending broadcasts

##### Update

Update updates a broadcast record
