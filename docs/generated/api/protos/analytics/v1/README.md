# Package analyticspb

## Constants

### AnalyticsService_TrackEvent_FullMethodName

## Variables

### AnalyticsService_ServiceDesc

AnalyticsService_ServiceDesc is the grpc.ServiceDesc for AnalyticsService service. It's only
intended for direct use with grpc.RegisterService, and not to be introspected or modified (even as a
copy)

### File_analytics_v1_analytics_proto

## Types

### AnalyticsEvent

#### Methods

##### Descriptor

Deprecated: Use AnalyticsEvent.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetEventId

##### GetMetadata

##### GetTimestamp

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AnalyticsServiceClient

AnalyticsServiceClient is the client API for AnalyticsService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### AnalyticsServiceServer

AnalyticsServiceServer is the server API for AnalyticsService service. All implementations must
embed UnimplementedAnalyticsServiceServer for forward compatibility.

### BatchTrackEventsRequest

#### Methods

##### Descriptor

Deprecated: Use BatchTrackEventsRequest.ProtoReflect.Descriptor instead.

##### GetEvents

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BatchTrackEventsResponse

#### Methods

##### Descriptor

Deprecated: Use BatchTrackEventsResponse.ProtoReflect.Descriptor instead.

##### GetFailureCount

##### GetSuccessCount

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CaptureEventRequest

#### Methods

##### Descriptor

Deprecated: Use CaptureEventRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetContext

##### GetEventType

##### GetGdprObscure

##### GetGroups

##### GetProperties

##### GetUserEmail

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CaptureEventResponse

#### Methods

##### Descriptor

Deprecated: Use CaptureEventResponse.ProtoReflect.Descriptor instead.

##### GetEventId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### EnrichEventMetadataRequest

#### Methods

##### Descriptor

Deprecated: Use EnrichEventMetadataRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetEventId

##### GetNewFields

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### EnrichEventMetadataResponse

#### Methods

##### Descriptor

Deprecated: Use EnrichEventMetadataResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Event

#### Methods

##### Descriptor

Deprecated: Use Event.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetEntityId

##### GetEntityType

##### GetEventType

##### GetId

##### GetMasterId

##### GetMasterUuid

##### GetMetadata

##### GetProperties

##### GetTimestamp

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetProductEventsRequest

#### Methods

##### Descriptor

Deprecated: Use GetProductEventsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPage

##### GetPageSize

##### GetProductId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetProductEventsResponse

#### Methods

##### Descriptor

Deprecated: Use GetProductEventsResponse.ProtoReflect.Descriptor instead.

##### GetEvents

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetReportRequest

#### Methods

##### Descriptor

Deprecated: Use GetReportRequest.ProtoReflect.Descriptor instead.

##### GetParameters

##### GetReportId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetReportResponse

#### Methods

##### Descriptor

Deprecated: Use GetReportResponse.ProtoReflect.Descriptor instead.

##### GetReport

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserEventsRequest

#### Methods

##### Descriptor

Deprecated: Use GetUserEventsRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserEventsResponse

#### Methods

##### Descriptor

Deprecated: Use GetUserEventsResponse.ProtoReflect.Descriptor instead.

##### GetEvents

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListEventsRequest

#### Methods

##### Descriptor

Deprecated: Use ListEventsRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListEventsResponse

#### Methods

##### Descriptor

Deprecated: Use ListEventsResponse.ProtoReflect.Descriptor instead.

##### GetEvents

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListReportsRequest

#### Methods

##### Descriptor

Deprecated: Use ListReportsRequest.ProtoReflect.Descriptor instead.

##### GetPage

##### GetPageSize

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListReportsResponse

#### Methods

##### Descriptor

Deprecated: Use ListReportsResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetReports

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Report

#### Methods

##### Descriptor

Deprecated: Use Report.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetData

##### GetDescription

##### GetId

##### GetName

##### GetParameters

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TrackEventRequest

#### Methods

##### Descriptor

Deprecated: Use TrackEventRequest.ProtoReflect.Descriptor instead.

##### GetEvent

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TrackEventResponse

#### Methods

##### Descriptor

Deprecated: Use TrackEventResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedAnalyticsServiceServer

UnimplementedAnalyticsServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### BatchTrackEvents

##### CaptureEvent

##### EnrichEventMetadata

##### GetProductEvents

##### GetReport

##### GetUserEvents

##### ListEvents

##### ListReports

##### TrackEvent

### UnsafeAnalyticsServiceServer

UnsafeAnalyticsServiceServer may be embedded to opt out of forward compatibility for this service.
Use of this interface is not recommended, as added methods to AnalyticsServiceServer will result in
compilation errors.

## Functions

### RegisterAnalyticsServiceServer
