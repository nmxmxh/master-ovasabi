# Package service

## Variables

### ServiceCacheConfigs

ServiceCacheConfigs is the central list of all service cache configs.

## Types

### CacheConfig

CacheConfig defines the configuration for a service cache.

### Provider

Provider manages service instances and their dependencies.

#### Methods

##### Admin

Admin returns the AdminService instance.

##### Analytics

Analytics returns the AnalyticsService instance.

##### Close

Close closes all resources.

##### Commerce

Commerce returns the CommerceService instance.

##### Container

Container returns the DI container.

##### Content

Content returns the ContentService instance.

##### ContentModeration

ContentModeration returns the ContentModerationService instance.

##### Localization

Localization returns the LocalizationService instance.

##### Nexus

Nexus returns the NexusServiceServer instance.

##### Notification

Notification returns the NotificationService instance.

##### RedisClient

RedisClient returns the underlying Redis client.

##### Referrals

Referrals returns the ReferralService instance.

##### Search

Search returns the SearchService instance.

##### Talent

Talent returns the TalentService instance.

##### User

User returns the UserService instance.

## Functions

### NewRedisProvider

NewRedisProvider initializes the Redis provider and registers all caches for all services in a
modular fashion. This function is used by the Provider to set up Redis-backed caching for DI and
orchestration.
