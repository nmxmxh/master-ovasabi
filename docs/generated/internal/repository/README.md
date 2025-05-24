# Package repository

## Constants

### TTLSearchPattern

### ContextMaster

For redis.ContextMaster, if not defined in pkg/redis, define here:.

## Variables

### ErrMasterNotFound

## Types

### BaseRepository

BaseRepository provides common database functionality.

#### Methods

##### BeginTx

BeginTx starts a new transaction.

##### CommitTx

CommitTx commits a transaction.

##### GenerateMasterName

GenerateMasterName creates a standardized name for master records.

##### GetContext

GetContext returns the context, possibly with transaction.

##### GetDB

GetDB returns the database connection.

##### RollbackTx

RollbackTx rolls back a transaction.

##### WithTx

WithTx returns a new repository with transaction.

### CacheInvalidationPattern

CacheInvalidationPattern represents a pattern for cache invalidation.

### CachedMasterRepository

CachedMasterRepository wraps a MasterRepository with caching.

#### Methods

##### Create

Create creates a master record with cache invalidation.

##### CreateMasterRecord

Add CreateMasterRecord to implement MasterRepository interface.

##### Delete

Delete deletes a master record with cache invalidation and locking.

##### Get

##### GetByUUID

##### List

##### QuickSearch

QuickSearch performs a fast search with caching.

##### SearchByPattern

SearchByPattern searches for master records matching a pattern with caching.

##### SearchByPatternAcrossTypes

SearchByPatternAcrossTypes searches across all types with caching.

##### Update

Update updates a master record with cache invalidation and locking.

##### WithLock

WithLock executes a function while holding a distributed lock.

### DBTX

DBTX represents a database connection that can execute queries or a transaction.

### DefaultMasterRepository

(interface is defined in types.go).

#### Methods

##### BatchExecute

##### BatchExecuteWithArgs

##### Create

Create creates a new master record.

##### CreateMasterRecord

Add CreateMasterRecord to implement MasterRepository interface.

##### Delete

Delete removes a master record.

##### ExecuteInTx

##### Get

Get retrieves a master record by ID.

##### GetByUUID

GetByUUID retrieves a master record by UUID.

##### List

List retrieves a paginated list of master records.

##### QuickSearch

QuickSearch performs a fast search with default parameters.

##### SearchByPattern

SearchByPattern searches for master records matching a pattern.

##### SearchByPatternAcrossTypes

SearchByPatternAcrossTypes searches for master records matching a pattern across all entity types.

##### Update

Update updates a master record with optimistic locking.

##### UpdateWithTransaction

UpdateWithTransaction updates a master record within a transaction.

##### WithLock

WithLock executes a function while holding a distributed lock.

### EntityType

EntityType represents the type of entity in the master table.

### Master

Master represents the core entity in the master table.

### MasterRepository

MasterRepository defines the interface for master entity operations, including caching and search.

### Provider

Provider manages repository instances.

#### Methods

##### GetMasterRepository

GetMasterRepository returns the master repository instance.

### Repository

Repository defines the common interface for all repositories.

### SearchResult

SearchResult represents a master record with similarity score.

### Statement

Statement represents a SQL statement with its arguments.

### TxFn

TxFn represents a function that will be executed within a transaction.

## Functions

### FromJSONB

FromJSONB unmarshals JSONB ([]byte) from Postgres to a map.

### ToJSONB

ToJSONB marshals a map to JSONB ([]byte) for Postgres.

### WithTransaction

WithTransaction executes the given function within a transaction.
