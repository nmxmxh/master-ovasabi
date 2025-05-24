# Package schedulerpb

## Constants

### SchedulerService_CreateJob_FullMethodName

## Variables

### TriggerType_name

Enum value maps for TriggerType.

### JobType_name

Enum value maps for JobType.

### JobStatus_name

Enum value maps for JobStatus.

### File_scheduler_v1_scheduler_proto

### SchedulerService_ServiceDesc

SchedulerService_ServiceDesc is the grpc.ServiceDesc for SchedulerService service. It's only
intended for direct use with grpc.RegisterService, and not to be introspected or modified (even as a
copy)

## Types

### CDCTrigger

CDCTrigger defines a change data capture trigger for event-based jobs.

#### Methods

##### Descriptor

Deprecated: Use CDCTrigger.ProtoReflect.Descriptor instead.

##### GetEventType

##### GetFilter

##### GetTable

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateJobRequest

#### Methods

##### Descriptor

Deprecated: Use CreateJobRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetJob

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateJobResponse

#### Methods

##### Descriptor

Deprecated: Use CreateJobResponse.ProtoReflect.Descriptor instead.

##### GetJob

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteJobRequest

#### Methods

##### Descriptor

Deprecated: Use DeleteJobRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetJobId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteJobResponse

#### Methods

##### Descriptor

Deprecated: Use DeleteJobResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetJobRequest

#### Methods

##### Descriptor

Deprecated: Use GetJobRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetJobId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetJobResponse

#### Methods

##### Descriptor

Deprecated: Use GetJobResponse.ProtoReflect.Descriptor instead.

##### GetJob

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Job

Job represents a scheduled or event-driven job.

#### Methods

##### Descriptor

Deprecated: Use Job.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetCdcTrigger

##### GetCreatedAt

##### GetHistory

##### GetId

##### GetJobType

##### GetLabels

##### GetLastRunId

##### GetMasterUuid

##### GetMetadata

##### GetName

##### GetNextRunTime

##### GetOwner

##### GetPayload

##### GetRetryPolicy

##### GetSchedule

##### GetStatus

##### GetTriggerType

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### JobHistory

JobHistory tracks past runs for a job (optional, for future use).

#### Methods

##### Descriptor

Deprecated: Use JobHistory.ProtoReflect.Descriptor instead.

##### GetRuns

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### JobRun

#### Methods

##### Descriptor

Deprecated: Use JobRun.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetError

##### GetFinishedAt

##### GetId

##### GetJobId

##### GetMasterUuid

##### GetMetadata

##### GetResult

##### GetStartedAt

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### JobStatus

JobStatus provides canonical job status values.

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use JobStatus.Descriptor instead.

##### Number

##### String

##### Type

### JobType

JobType classifies the type of job for better filtering and handling.

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use JobType.Descriptor instead.

##### Number

##### String

##### Type

### ListJobRunsRequest

#### Methods

##### Descriptor

Deprecated: Use ListJobRunsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetJobId

##### GetPage

##### GetPageSize

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListJobRunsResponse

#### Methods

##### Descriptor

Deprecated: Use ListJobRunsResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetRuns

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListJobsRequest

#### Methods

##### Descriptor

Deprecated: Use ListJobsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPage

##### GetPageSize

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListJobsResponse

#### Methods

##### Descriptor

Deprecated: Use ListJobsResponse.ProtoReflect.Descriptor instead.

##### GetJobs

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RetryPolicy

RetryPolicy defines how failed jobs are retried.

#### Methods

##### Descriptor

Deprecated: Use RetryPolicy.ProtoReflect.Descriptor instead.

##### GetBackoffSeconds

##### GetExponential

##### GetMaxAttempts

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RunJobRequest

#### Methods

##### Descriptor

Deprecated: Use RunJobRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetJobId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RunJobResponse

#### Methods

##### Descriptor

Deprecated: Use RunJobResponse.ProtoReflect.Descriptor instead.

##### GetRun

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SchedulerServiceClient

SchedulerServiceClient is the client API for SchedulerService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### SchedulerServiceServer

SchedulerServiceServer is the server API for SchedulerService service. All implementations must
embed UnimplementedSchedulerServiceServer for forward compatibility.

### TriggerType

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use TriggerType.Descriptor instead.

##### Number

##### String

##### Type

### UnimplementedSchedulerServiceServer

UnimplementedSchedulerServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### CreateJob

##### DeleteJob

##### GetJob

##### ListJobRuns

##### ListJobs

##### RunJob

##### UpdateJob

### UnsafeSchedulerServiceServer

UnsafeSchedulerServiceServer may be embedded to opt out of forward compatibility for this service.
Use of this interface is not recommended, as added methods to SchedulerServiceServer will result in
compilation errors.

### UpdateJobRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateJobRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetJob

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateJobResponse

#### Methods

##### Descriptor

Deprecated: Use UpdateJobResponse.ProtoReflect.Descriptor instead.

##### GetJob

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterSchedulerServiceServer
