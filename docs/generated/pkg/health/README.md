# Package health

## Types

### Check

Check interface defines the health check contract.

### Checker

Checker manages health checks.

#### Methods

##### Add

Add adds a new health check.

##### Check

Check performs all health checks.

##### Register

Register adds a new health check.

##### Run

Run performs all health checks.

### Client

Client represents a health check client.

#### Methods

##### Check

Check performs a health check against the remote service.

### DatabaseCheck

DatabaseCheck checks database connectivity.

#### Methods

##### Check

##### Name

### GRPCClient

GRPCClient provides methods to check gRPC service health.

#### Methods

##### Close

Close closes the client connection.

##### WaitForReady

WaitForReady waits for the service to be ready with a timeout.

### HTTPCheck

HTTPCheck checks HTTP service connectivity.

#### Methods

##### Check

##### Name

### RedisCheck

RedisCheck checks Redis connectivity.

#### Methods

##### Check

##### Name

### Status

Status represents the health status.
