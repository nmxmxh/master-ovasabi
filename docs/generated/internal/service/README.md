# Package service

## Variables

### ServiceCacheConfigs

ServiceCacheConfigs is the central list of all service cache configs.

## Types

### CacheConfig

CacheConfig defines the configuration for a service cache.

### Provider

#### Methods

##### EmitEchoEvent

EmitEchoEvent emits a canonical 'echo' event to Nexus for testing and onboarding.

##### EmitEvent

EmitEvent emits an event to the Nexus event bus.

##### EmitEventWithLogging

EmitEventWithLogging emits an event to Nexus and logs the outcome, orchestrating errors with
graceful.

##### StartEchoLoop

StartEchoLoop starts a background goroutine that emits an echo event every 15 seconds.

##### SubscribeEvents

SubscribeEvents subscribes to events from the Nexus event bus.

## Functions

### NewRedisProvider

NewRedisProvider initializes the Redis provider and registers all caches for all services in a
modular fashion. This function is used by the Provider to set up Redis-backed caching for DI and
orchestration.
