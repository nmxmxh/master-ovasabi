# Package service

## Variables

### ServiceCacheConfigs

ServiceCacheConfigs is the central list of all service cache configs.

## Types

### CacheConfig

CacheConfig defines the configuration for a service cache.

### Provider

#### Methods

##### EmitEvent

EmitEvent emits an event to the Nexus event bus.

##### SubscribeEvents

SubscribeEvents subscribes to events from the Nexus event bus.

## Functions

### NewRedisProvider

NewRedisProvider initializes the Redis provider and registers all caches for all services in a
modular fashion. This function is used by the Provider to set up Redis-backed caching for DI and
orchestration.
