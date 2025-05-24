# Package security

## Constants

### SecurityService_Authenticate_FullMethodName

## Variables

### File_security_v1_security_proto

### SecurityService_ServiceDesc

SecurityService_ServiceDesc is the grpc.ServiceDesc for SecurityService service. It's only intended
for direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### AuditEventRequest

#### Methods

##### Descriptor

Deprecated: Use AuditEventRequest.ProtoReflect.Descriptor instead.

##### GetAction

##### GetEventType

##### GetMetadata

##### GetPrincipalId

##### GetResource

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuditEventResponse

#### Methods

##### Descriptor

Deprecated: Use AuditEventResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetMetadata

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthenticateRequest

#### Methods

##### Descriptor

Deprecated: Use AuthenticateRequest.ProtoReflect.Descriptor instead.

##### GetCredential

##### GetMetadata

##### GetPrincipalId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthenticateResponse

#### Methods

##### Descriptor

Deprecated: Use AuthenticateResponse.ProtoReflect.Descriptor instead.

##### GetExpiresAt

##### GetMetadata

##### GetSessionToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthorizeRequest

#### Methods

##### Descriptor

Deprecated: Use AuthorizeRequest.ProtoReflect.Descriptor instead.

##### GetAction

##### GetMetadata

##### GetPrincipalId

##### GetResource

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuthorizeResponse

#### Methods

##### Descriptor

Deprecated: Use AuthorizeResponse.ProtoReflect.Descriptor instead.

##### GetAllowed

##### GetMetadata

##### GetReason

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DetectThreatsRequest

Threat detection and audit event messages

#### Methods

##### Descriptor

Deprecated: Use DetectThreatsRequest.ProtoReflect.Descriptor instead.

##### GetContextType

##### GetMetadata

##### GetPrincipalId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DetectThreatsResponse

#### Methods

##### Descriptor

Deprecated: Use DetectThreatsResponse.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetThreats

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetPolicyRequest

#### Methods

##### Descriptor

Deprecated: Use GetPolicyRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPolicyId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetPolicyResponse

#### Methods

##### Descriptor

Deprecated: Use GetPolicyResponse.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPolicy

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### IssueSecretRequest

#### Methods

##### Descriptor

Deprecated: Use IssueSecretRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPrincipalId

##### GetSecretType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### IssueSecretResponse

#### Methods

##### Descriptor

Deprecated: Use IssueSecretResponse.ProtoReflect.Descriptor instead.

##### GetExpiresAt

##### GetMetadata

##### GetSecret

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### QueryEventsRequest

#### Methods

##### Descriptor

Deprecated: Use QueryEventsRequest.ProtoReflect.Descriptor instead.

##### GetEventType

##### GetFrom

##### GetMetadata

##### GetPrincipalId

##### GetTo

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### QueryEventsResponse

#### Methods

##### Descriptor

Deprecated: Use QueryEventsResponse.ProtoReflect.Descriptor instead.

##### GetEvents

##### GetMetadata

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SecurityEvent

#### Methods

##### Descriptor

Deprecated: Use SecurityEvent.ProtoReflect.Descriptor instead.

##### GetAction

##### GetDetails

##### GetEventType

##### GetId

##### GetPrincipalId

##### GetResource

##### GetTimestamp

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SecurityServiceClient

SecurityServiceClient is the client API for SecurityService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

SecurityService: Central authority for platform security (not user CRUD/session mgmt)

### SecurityServiceServer

SecurityServiceServer is the server API for SecurityService service. All implementations must embed
UnimplementedSecurityServiceServer for forward compatibility.

SecurityService: Central authority for platform security (not user CRUD/session mgmt)

### SetPolicyRequest

#### Methods

##### Descriptor

Deprecated: Use SetPolicyRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPolicy

##### GetPolicyId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SetPolicyResponse

#### Methods

##### Descriptor

Deprecated: Use SetPolicyResponse.ProtoReflect.Descriptor instead.

##### GetError

##### GetMetadata

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ThreatSignal

#### Methods

##### Descriptor

Deprecated: Use ThreatSignal.ProtoReflect.Descriptor instead.

##### GetDescription

##### GetMetadata

##### GetScore

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedSecurityServiceServer

UnimplementedSecurityServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### AuditEvent

##### Authenticate

##### Authorize

##### DetectThreats

##### GetPolicy

##### IssueSecret

##### QueryEvents

##### SetPolicy

##### ValidateCredential

### UnsafeSecurityServiceServer

UnsafeSecurityServiceServer may be embedded to opt out of forward compatibility for this service.
Use of this interface is not recommended, as added methods to SecurityServiceServer will result in
compilation errors.

### ValidateCredentialRequest

#### Methods

##### Descriptor

Deprecated: Use ValidateCredentialRequest.ProtoReflect.Descriptor instead.

##### GetCredential

##### GetMetadata

##### GetType

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ValidateCredentialResponse

#### Methods

##### Descriptor

Deprecated: Use ValidateCredentialResponse.ProtoReflect.Descriptor instead.

##### GetExpiresAt

##### GetMetadata

##### GetPrincipalId

##### GetValid

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterSecurityServiceServer
