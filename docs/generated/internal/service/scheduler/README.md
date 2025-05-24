# Package scheduler

## Variables

### SchedulerEventRegistry

## Types

### EventEmitter

EventEmitter defines the interface for emitting events (canonical platform interface).

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Metadata

SchedulerMetadata defines scheduler-specific metadata fields.

### Repository

#### Methods

##### CreateJob

##### DeleteJob

##### GetJob

##### ListJobRuns

##### ListJobs

##### RunJob

##### SubscribeToCDCEvents

SubscribeToCDCEvents subscribes to CDC events on the master table using PostgreSQL LISTEN/NOTIFY.
The handler receives the JSON payload as a string (with id and event_type).

##### UpdateJob

### RepositoryItf

### SchedulingInfo

SchedulingInfo defines the scheduling fields for jobs.

### Service

Service implements the Scheduler business logic with rich metadata handling and gRPC server
interface.

#### Methods

##### CreateJob

CreateJob implements the gRPC CreateJob endpoint.

##### DeleteJob

DeleteJob implements the gRPC DeleteJob endpoint.

##### GetJob

GetJob implements the gRPC GetJob endpoint.

##### ListJobRuns

ListJobRuns implements the gRPC ListJobRuns endpoint.

##### ListJobs

ListJobs implements the gRPC ListJobs endpoint.

##### RunJob

RunJob implements the gRPC RunJob endpoint.

##### UpdateJob

UpdateJob implements the gRPC UpdateJob endpoint.

## Functions

### EnrichSchedulerMetadata

EnrichSchedulerMetadata adds/updates scheduler-specific fields in commonpb.Metadata.

### Register

Register registers the scheduler service with the DI container and event bus support.

### StartEventSubscribers

### ValidateSchedulerMetadata

ValidateSchedulerMetadata validates required scheduler metadata fields.
