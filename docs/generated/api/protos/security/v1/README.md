# Package securitypb

## Constants

### SecurityService_Authenticate_FullMethodName

## Variables

### FactorType_name

Enum value maps for FactorType.

### File_security_v1_security_proto

### SecurityService_ServiceDesc

SecurityService_ServiceDesc is the grpc.ServiceDesc for SecurityService service. It's only intended
for direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### AuditEvent

#### Methods

##### Descriptor

Deprecated: Use AuditEvent.ProtoReflect.Descriptor instead.

##### GetAction

##### GetContext

##### GetEventId

##### GetEventType

##### GetMetadata

##### GetPrincipal

##### GetResource

##### GetTimestamp

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuditLogEntry

#### Methods

##### Descriptor

Deprecated: Use AuditLogEntry.ProtoReflect.Descriptor instead.

##### GetEnrichedMetadata

##### GetEvent

##### GetRelatedIncidents

##### GetRisk

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuditLogRequest

#### Methods

##### Descriptor

Deprecated: Use AuditLogRequest.ProtoReflect.Descriptor instead.

##### GetEndTime

##### GetEventType

##### GetPageSize

##### GetPageToken

##### GetPrincipal

##### GetResource

##### GetStartTime

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthenticateRequest

#### Methods

##### Descriptor

Deprecated: Use AuthenticateRequest.ProtoReflect.Descriptor instead.

##### GetContext

##### GetFactors

##### GetIdentity

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthenticateResponse

#### Methods

##### Descriptor

Deprecated: Use AuthenticateResponse.ProtoReflect.Descriptor instead.

##### GetExpiration

##### GetMetadata

##### GetPermissions

##### GetSecurityScore

##### GetSessionToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthenticationFactor

#### Methods

##### Descriptor

Deprecated: Use AuthenticationFactor.ProtoReflect.Descriptor instead.

##### GetCredential

##### GetMetadata

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthenticationRequest

#### Methods

##### Descriptor

Deprecated: Use AuthenticationRequest.ProtoReflect.Descriptor instead.

##### GetContext

##### GetFactors

##### GetIdentity

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthenticationResponse

#### Methods

##### Descriptor

Deprecated: Use AuthenticationResponse.ProtoReflect.Descriptor instead.

##### GetExpiration

##### GetMetadata

##### GetPermissions

##### GetSecurityScore

##### GetSessionToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthorizationRequest

#### Methods

##### Descriptor

Deprecated: Use AuthorizationRequest.ProtoReflect.Descriptor instead.

##### GetAction

##### GetContext

##### GetResource

##### GetSessionToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthorizationResponse

#### Methods

##### Descriptor

Deprecated: Use AuthorizationResponse.ProtoReflect.Descriptor instead.

##### GetApplicablePolicies

##### GetAuthorized

##### GetMetadata

##### GetReason

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthorizeRequest

#### Methods

##### Descriptor

Deprecated: Use AuthorizeRequest.ProtoReflect.Descriptor instead.

##### GetAction

##### GetContext

##### GetResource

##### GetSessionToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthorizeResponse

#### Methods

##### Descriptor

Deprecated: Use AuthorizeResponse.ProtoReflect.Descriptor instead.

##### GetApplicablePolicies

##### GetAuthorized

##### GetMetadata

##### GetReason

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DetectThreatsRequest

#### Methods

##### Descriptor

Deprecated: Use DetectThreatsRequest.ProtoReflect.Descriptor instead.

##### GetAction

##### GetAttributes

##### GetContext

##### GetResourceId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DetectThreatsResponse

#### Methods

##### Descriptor

Deprecated: Use DetectThreatsResponse.ProtoReflect.Descriptor instead.

##### GetDetectedThreats

##### GetMetadata

##### GetMitigations

##### GetThreatScore

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Distribution

#### Methods

##### Descriptor

Deprecated: Use Distribution.ProtoReflect.Descriptor instead.

##### GetMean

##### GetMedian

##### GetP95

##### GetP99

##### GetPercentiles

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### FactorType

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use FactorType.Descriptor instead.

##### Number

##### String

##### Type

### GetAuditLogRequest

#### Methods

##### Descriptor

Deprecated: Use GetAuditLogRequest.ProtoReflect.Descriptor instead.

##### GetEndTime

##### GetEventType

##### GetPageSize

##### GetPageToken

##### GetPrincipal

##### GetResource

##### GetStartTime

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetAuditLogResponse

#### Methods

##### Descriptor

Deprecated: Use GetAuditLogResponse.ProtoReflect.Descriptor instead.

##### GetEntry

##### GetMetadata

##### GetNextPageToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetSecurityMetricsRequest

#### Methods

##### Descriptor

Deprecated: Use GetSecurityMetricsRequest.ProtoReflect.Descriptor instead.

##### GetEndTime

##### GetMetricTypes

##### GetStartTime

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetSecurityMetricsResponse

#### Methods

##### Descriptor

Deprecated: Use GetSecurityMetricsResponse.ProtoReflect.Descriptor instead.

##### GetIncidents

##### GetMetadata

##### GetMetrics

##### GetOverallScore

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GraphEdge

#### Methods

##### Descriptor

Deprecated: Use GraphEdge.ProtoReflect.Descriptor instead.

##### GetProperties

##### GetRelationship

##### GetSource

##### GetTarget

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GraphVertex

#### Methods

##### Descriptor

Deprecated: Use GraphVertex.ProtoReflect.Descriptor instead.

##### GetId

##### GetProperties

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### IncidentReport

#### Methods

##### Descriptor

Deprecated: Use IncidentReport.ProtoReflect.Descriptor instead.

##### GetContext

##### GetDescription

##### GetIncidentId

##### GetMetadata

##### GetSeverity

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### IncidentResponse

#### Methods

##### Descriptor

Deprecated: Use IncidentResponse.ProtoReflect.Descriptor instead.

##### GetActionsTaken

##### GetResolutionTime

##### GetStatus

##### GetTrackingId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MetricValue

#### Methods

##### Descriptor

Deprecated: Use MetricValue.ProtoReflect.Descriptor instead.

##### GetDistribution

##### GetLabels

##### GetNumericValue

##### GetStringValue

##### GetValue

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MetricValue_Distribution

### MetricValue_NumericValue

### MetricValue_StringValue

### PatternRegistrationResponse

#### Methods

##### Descriptor

Deprecated: Use PatternRegistrationResponse.ProtoReflect.Descriptor instead.

##### GetPatternId

##### GetStatus

##### GetValidationMessages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### PatternValidationRequest

#### Methods

##### Descriptor

Deprecated: Use PatternValidationRequest.ProtoReflect.Descriptor instead.

##### GetContext

##### GetPatternId

##### GetValidationParams

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### PatternValidationResponse

#### Methods

##### Descriptor

Deprecated: Use PatternValidationResponse.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetRiskAssessment

##### GetValid

##### GetValidationErrors

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RecordAuditEventRequest

#### Methods

##### Descriptor

Deprecated: Use RecordAuditEventRequest.ProtoReflect.Descriptor instead.

##### GetEvent

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RecordAuditEventResponse

#### Methods

##### Descriptor

Deprecated: Use RecordAuditEventResponse.ProtoReflect.Descriptor instead.

##### GetEventId

##### GetMetadata

##### GetStatus

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RegisterSecurityPatternRequest

#### Methods

##### Descriptor

Deprecated: Use RegisterSecurityPatternRequest.ProtoReflect.Descriptor instead.

##### GetPattern

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RegisterSecurityPatternResponse

#### Methods

##### Descriptor

Deprecated: Use RegisterSecurityPatternResponse.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPatternId

##### GetStatus

##### GetValidationMessages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ReportIncidentRequest

#### Methods

##### Descriptor

Deprecated: Use ReportIncidentRequest.ProtoReflect.Descriptor instead.

##### GetContext

##### GetDescription

##### GetMetadata

##### GetSeverity

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ReportIncidentResponse

#### Methods

##### Descriptor

Deprecated: Use ReportIncidentResponse.ProtoReflect.Descriptor instead.

##### GetActionsTaken

##### GetMetadata

##### GetResolutionTime

##### GetStatus

##### GetTrackingId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RiskAssessment

#### Methods

##### Descriptor

Deprecated: Use RiskAssessment.ProtoReflect.Descriptor instead.

##### GetFactorWeights

##### GetMitigations

##### GetRiskScore

##### GetSeverity

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SecurityContext

#### Methods

##### Descriptor

Deprecated: Use SecurityContext.ProtoReflect.Descriptor instead.

##### GetAttributes

##### GetClientIp

##### GetRequestId

##### GetTimestamp

##### GetUserAgent

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SecurityIncident

#### Methods

##### Descriptor

Deprecated: Use SecurityIncident.ProtoReflect.Descriptor instead.

##### GetContext

##### GetDescription

##### GetDetectionTime

##### GetIncidentId

##### GetRisk

##### GetSeverity

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SecurityMetricsRequest

#### Methods

##### Descriptor

Deprecated: Use SecurityMetricsRequest.ProtoReflect.Descriptor instead.

##### GetEndTime

##### GetMetricTypes

##### GetStartTime

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SecurityMetricsResponse

#### Methods

##### Descriptor

Deprecated: Use SecurityMetricsResponse.ProtoReflect.Descriptor instead.

##### GetIncidents

##### GetMetadata

##### GetMetrics

##### GetOverallScore

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SecurityPattern

#### Methods

##### Descriptor

Deprecated: Use SecurityPattern.ProtoReflect.Descriptor instead.

##### GetConstraints

##### GetDescription

##### GetEdges

##### GetMetadata

##### GetName

##### GetPatternId

##### GetRisk

##### GetVertices

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SecurityScore

#### Methods

##### Descriptor

Deprecated: Use SecurityScore.ProtoReflect.Descriptor instead.

##### GetAuthenticationScore

##### GetFactorScores

##### GetPrivacyScore

##### GetThreatScore

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SecurityServiceClient

SecurityServiceClient is the client API for SecurityService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### SecurityServiceServer

SecurityServiceServer is the server API for SecurityService service. All implementations must embed
UnimplementedSecurityServiceServer for forward compatibility.

### SecurityService_GetAuditLogClient

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### SecurityService_GetAuditLogServer

This type alias is provided for backwards compatibility with existing code that references the prior
non-generic stream type by name.

### ThreatDetectionRequest

#### Methods

##### Descriptor

Deprecated: Use ThreatDetectionRequest.ProtoReflect.Descriptor instead.

##### GetAction

##### GetAttributes

##### GetContext

##### GetResourceId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ThreatDetectionResponse

#### Methods

##### Descriptor

Deprecated: Use ThreatDetectionResponse.ProtoReflect.Descriptor instead.

##### GetDetectedThreats

##### GetMetadata

##### GetMitigations

##### GetThreatScore

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TokenValidationRequest

#### Methods

##### Descriptor

Deprecated: Use TokenValidationRequest.ProtoReflect.Descriptor instead.

##### GetContext

##### GetToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TokenValidationResponse

#### Methods

##### Descriptor

Deprecated: Use TokenValidationResponse.ProtoReflect.Descriptor instead.

##### GetExpiration

##### GetMetadata

##### GetReason

##### GetSecurityScore

##### GetValid

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedSecurityServiceServer

UnimplementedSecurityServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### Authenticate

##### Authorize

##### DetectThreats

##### GetAuditLog

##### GetSecurityMetrics

##### RecordAuditEvent

##### RegisterSecurityPattern

##### ReportIncident

##### ValidatePattern

##### ValidateToken

### UnsafeSecurityServiceServer

UnsafeSecurityServiceServer may be embedded to opt out of forward compatibility for this service.
Use of this interface is not recommended, as added methods to SecurityServiceServer will result in
compilation errors.

### ValidatePatternRequest

#### Methods

##### Descriptor

Deprecated: Use ValidatePatternRequest.ProtoReflect.Descriptor instead.

##### GetContext

##### GetPatternId

##### GetValidationParams

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ValidatePatternResponse

#### Methods

##### Descriptor

Deprecated: Use ValidatePatternResponse.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetRiskAssessment

##### GetValid

##### GetValidationErrors

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ValidateTokenRequest

#### Methods

##### Descriptor

Deprecated: Use ValidateTokenRequest.ProtoReflect.Descriptor instead.

##### GetContext

##### GetToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ValidateTokenResponse

#### Methods

##### Descriptor

Deprecated: Use ValidateTokenResponse.ProtoReflect.Descriptor instead.

##### GetExpiration

##### GetMetadata

##### GetReason

##### GetSecurityScore

##### GetValid

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterSecurityServiceServer
