# Redis Practices

> **NOTE:** This document defines the required instructions, rules, and best practices for all Redis-related code, operations, and usage within this project.
>
> Before integrating or modifying Redis operations, contributors and tools (including AI assistants) must read and follow the rules in this file.

---

## General Principles

1. **Use Redis purposefully.**  
   Redis is best used for caching, pub/sub messaging, short-lived metadata, queues, and ephemeral storage.

2. **Do not store critical, persistent data in Redis.**  
   Redis is an in-memory store and may lose data unexpectedly if not configured for persistence.

3. **Design for TTL (time-to-live).**  
   Always set reasonable expiration times for non-permanent keys.

4. **Organize keyspace carefully.**  
   Use structured, consistent, and hierarchical key naming (see 'Key Naming Pattern' section).

5. **Minimize Redis memory usage.**  
   Only store essential, transient, or cacheable information.

6. **Use dependency injection for Redis clients.**  
   Always inject the Redis client into your services to maintain testability, lifecycle management, and loose coupling.

7. **Design for failure.**  
   Handle Redis downtime gracefully. Never assume Redis operations always succeed.

---

## Key Naming Pattern

- **Namespace:Context:Entity:Attribute**
  - Examples:
    - `cache:user:1234:profile`
    - `session:auth:token:abcd1234`
    - `queue:service:email_dispatch`

- **Naming Rules:**
  - Lowercase words.
  - Use colons (`:`) to separate logical parts.
  - Prefix keys by function (e.g., `cache:`, `queue:`, `session:`).
  - Keep key names short but descriptive.

- **Consistency is mandatory.**

---

## Usage Patterns

1. **Caching:**
   - Use Redis to cache frequent database reads.
   - Always set an expiration (TTL) when caching.
   - Invalidate or refresh caches on data updates.

2. **Sessions and Tokens:**
   - Store short-lived authentication sessions with expirations.
   - Never store long-lived user state solely in Redis.

3. **Queues and Pub/Sub:**
   - Use Redis lists for simple queues.
   - Use Redis pub/sub for broadcasting ephemeral messages.
   - If queue persistence is important, consider using reliable message queues (e.g., Kafka).

4. **Ephemeral Metadata:**
   - Use Redis for metadata needed during processing (e.g., job locks, temporary counters).

5. **Locks:**
   - Use simple locking with `SET resource_name my_random_value NX PX 30000`.
   - Always include expiration (`PX`) to avoid deadlocks.
   - For critical locks, use RedLock or a robust distributed locking library.

---

## Dependency Injection & Lifecycle

- **Inject Redis Client:**
  - Configure Redis clients at the application entry point.
  - Inject client instances into services via constructor or context.

- **Connection Management:**
  - Pool connections where possible.
  - Close Redis clients gracefully on server shutdown.

- **Health Checks:**
  - Add a `PING` check to monitor Redis availability in service health endpoints.

---

## Performance and Optimization

1. **Use pipelining for batch operations.**
2. **Avoid SCAN, KEYS, and FLUSHALL in production.**
   - They are expensive and can block Redis.
3. **Use appropriate data structures:**
   - Strings, lists, sets, sorted sets, hashes.
   - Choose structures suited to your access patterns.
4. **Set memory limits and eviction policies.**
   - Configure `maxmemory` and `maxmemory-policy`.
5. **Monitor hit/miss ratios.**
   - Adjust caching strategies based on access patterns.

---

## Security and Resilience

1. **Do not expose Redis to the open internet.**
   - Use private networks, firewalls, and authentication.
2. **Enable Redis AUTH and ACLs.**
3. **Encrypt traffic if using Redis over networks.**
4. **Backups and persistence:**
   - Enable RDB or AOF if persistence is necessary.
   - Regularly snapshot if needed.
5. **Graceful degradation:**
   - Design applications to continue operating if Redis is unavailable (e.g., fallback mechanisms).

---

## Project-Specific Redis Usage

### Key Prefixes

- `session:auth:` - Authentication tokens and sessions
- `cache:user:` - User profile and data caching
- `cache:referral:` - Referral code caching
- `queue:notification:` - Notification dispatch queue
- `lock:broadcast:` - Broadcast operation locks
- `rate:auth:` - Authentication rate limiting

### TTL Policies

- Authentication tokens: 24 hours
- User profile cache: 1 hour
- Referral code cache: 6 hours
- Rate limiting keys: 1 minute
- Lock keys: 30 seconds maximum

### Service-Specific Usage

1. **Auth Service:**
   - Token caching
   - Rate limiting
   - Session management

2. **User Service:**
   - Profile caching
   - Referral code caching

3. **Notification Service:**
   - Notification queuing
   - Delivery status tracking

4. **Broadcast Service:**
   - Real-time message distribution
   - Subscription management

---

## Implementation Examples

See the following files for implementation details:

- `pkg/redis/client.go` - Redis client initialization and configuration
- `pkg/redis/cache.go` - Caching utilities and helpers
- `pkg/redis/lock.go` - Distributed locking implementation
- `pkg/redis/queue.go` - Queue management utilities
- `pkg/redis/health.go` - Health check implementation

---

**Remember:**  
Every Redis integration must be reviewed for compliance with these practices.  
When in doubt, consult a senior engineer or architect before proceeding.
