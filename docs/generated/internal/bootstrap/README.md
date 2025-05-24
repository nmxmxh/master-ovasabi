# Package bootstrap

## Types

### EventEmitter

EventEmitter interface (canonical platform interface).

### ServiceBootstrapper

ServiceBootstrapper centralizes registration of all services.

#### Methods

##### RegisterAll

RegisterAll registers all core services with the DI container and event bus using a struct-based
pattern. It no longer runs health checks; call RunHealthChecks after the gRPC server is started.

##### RunHealthChecks

RunHealthChecks runs all health checks for registered services. Call this after the gRPC server is
started and listening.

### ServiceRegistrationEntry

ServiceRegistrationEntry defines a struct for service registration metadata and logic.

## Functions

### StartAllEventSubscribers

StartAllEventSubscribers starts all event subscribers for all services.
