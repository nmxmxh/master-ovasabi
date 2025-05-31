# Package redis

## Constants

### NamespaceCache

Redis namespaces defines the top-level key prefixes for different types of data.

### ContextAuth

Redis contexts defines the second-level key prefixes for specific domains.

### TTLUserProfile

TTL constants defines the time-to-live durations for different types of data.

### PatternOriginSystem

## Types

### Cache

Cache provides Redis caching functionality.

#### Methods

##### Close

Close closes the Redis connection.

##### Delete

Delete removes a value from the cache.

##### DeletePattern

DeletePattern removes all keys matching a pattern.

##### Get

Get retrieves a value from the cache.

##### GetClient

GetClient returns the underlying Redis client.

##### Pipeline

Pipeline returns a new pipeline.

##### SAdd

SAdd adds members to a set.

##### SDiff

SDiff returns the difference between multiple sets.

##### SInter

SInter returns the intersection of multiple sets.

##### SIsMember

SIsMember checks if a member exists in a set.

##### SMembers

SMembers returns all members of a set.

##### SRem

SRem removes members from a set.

##### SUnion

SUnion returns the union of multiple sets.

##### Set

Set stores a value in the cache.

##### SetNX

SetNX sets a value if the key doesn't exist.

##### TxPipeline

TxPipeline returns a new transaction pipeline.

##### ZAdd

ZAdd adds members to a sorted set.

##### ZRange

ZRange returns a range of members from a sorted set.

##### ZRangeByScore

ZRangeByScore returns members from a sorted set by score.

##### ZRem

ZRem removes members from a sorted set.

### Client

Client wraps the Redis client with additional functionality.

#### Methods

##### Close

Close closes the Redis client connection.

##### IsAvailable

IsAvailable checks if Redis is available.

##### WithTimeout

WithTimeout wraps a context with a timeout.

### Config

Config holds Redis configuration.

### ExecutorOptions

ExecutorOptions defines configuration options for the pattern executor.

### KeyBuilder

KeyBuilder helps build Redis keys according to our naming convention.

#### Methods

##### Build

Build creates a Redis key following our naming convention.

##### BuildHash

BuildHash creates a Redis hash key.

##### BuildLock

BuildLock creates a Redis lock key.

##### BuildPattern

BuildPattern creates a Redis key pattern for searching.

##### BuildSet

BuildSet creates a Redis set key.

##### BuildTemp

BuildTemp creates a temporary Redis key.

##### BuildZSet

BuildZSet creates a Redis sorted set key.

##### GetContext

GetContext returns the context.

##### GetNamespace

GetNamespace returns the namespace.

##### Parse

Parse extracts components from a Redis key.

##### WithContext

WithContext creates a new key builder with a different context.

##### WithNamespace

WithNamespace creates a new key builder with a different namespace.

### OperationStep

OperationStep defines a single step in a pattern.

### Options

Options configures the Redis cache.

### PatternCategory

PatternCategory defines the category of a pattern.

### PatternExecutor

PatternExecutor executes stored patterns.

#### Methods

##### ExecutePattern

ExecutePattern executes a pattern with the given input.

### PatternOrigin

PatternOrigin defines the source of a pattern.

### PatternStore

PatternStore manages pattern storage in Redis.

#### Methods

##### DeletePattern

DeletePattern deletes a pattern from Redis.

##### GetPattern

GetPattern retrieves a pattern from Redis.

##### ListPatterns

ListPatterns lists patterns based on filters.

##### StorePattern

StorePattern stores a pattern in Redis.

##### UpdatePatternStats

UpdatePatternStats updates pattern usage statistics.

### Provider

Provider manages Redis cache instances.

#### Methods

##### Close

Close closes all Redis connections.

##### FlushAll

FlushAll flushes all Redis instances.

##### GetCache

GetCache returns a Redis cache instance.

##### GetPatternExecutor

GetPatternExecutor returns the pattern executor instance.

##### GetPatternStore

GetPatternStore returns the pattern store instance.

##### InitializePatternSupport

InitializePatternSupport initializes pattern store and executor.

##### Ping

Ping checks the connection to all Redis instances.

##### RegisterCache

RegisterCache registers a Redis cache configuration.

##### RegisterPatternCache

RegisterPatternCache registers the pattern cache configuration.

##### Stats

Stats returns statistics for all Redis instances.

### StoredPattern

StoredPattern represents a pattern stored in Redis.

## Functions

### EmitToDLQ

EmitToDLQ emits a failed event to the dead-letter queue (DLQ) Redis stream.

### GetOrSetWithProtection

GetOrSetWithProtection provides cache stampede protection using a sync.Map and singleflight pattern.
