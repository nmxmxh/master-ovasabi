# Package v0

## Constants

### AuthService_Register_FullMethodName

## Variables

### AuthService_ServiceDesc

AuthService_ServiceDesc is the grpc.ServiceDesc for AuthService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

### File_api_protos_auth_v0_auth_proto

## Types

### AuthServiceClient

AuthServiceClient is the client API for AuthService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### AuthServiceServer

AuthServiceServer is the server API for AuthService service. All implementations must embed
UnimplementedAuthServiceServer for forward compatibility

### GetUserAuthRequest

GetUserAuthRequest contains the user identifier for GetUserAuth

#### Methods

##### Descriptor

Deprecated: Use GetUserAuthRequest.ProtoReflect.Descriptor instead.

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserAuthResponse

GetUserAuthResponse contains user authentication information

#### Methods

##### Descriptor

Deprecated: Use GetUserAuthResponse.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetCreatedAt

##### GetMasterId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserRequest

GetUserRequest contains the user identifier

#### Methods

##### Descriptor

Deprecated: Use GetUserRequest.ProtoReflect.Descriptor instead.

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserResponse

GetUserResponse contains user information

#### Methods

##### Descriptor

Deprecated: Use GetUserResponse.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetEmail

##### GetName

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### LoginRequest

LoginRequest contains user login credentials

#### Methods

##### Descriptor

Deprecated: Use LoginRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMetadata

##### GetPassword

##### GetUsername

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### LoginResponse

LoginResponse contains authentication tokens

#### Methods

##### Descriptor

Deprecated: Use LoginResponse.ProtoReflect.Descriptor instead.

##### GetMessage

##### GetSuccess

##### GetToken

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RegisterRequest

RegisterRequest contains user registration information

#### Methods

##### Descriptor

Deprecated: Use RegisterRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetMasterId

##### GetMetadata

##### GetPassword

##### GetUsername

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RegisterResponse

RegisterResponse contains the result of registration

#### Methods

##### Descriptor

Deprecated: Use RegisterResponse.ProtoReflect.Descriptor instead.

##### GetMessage

##### GetSuccess

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedAuthServiceServer

UnimplementedAuthServiceServer must be embedded to have forward compatible implementations.

#### Methods

##### GetUser

##### GetUserAuth

##### Login

##### Register

##### ValidateToken

### UnsafeAuthServiceServer

UnsafeAuthServiceServer may be embedded to opt out of forward compatibility for this service. Use of
this interface is not recommended, as added methods to AuthServiceServer will result in compilation
errors.

### ValidateTokenRequest

ValidateTokenRequest contains the token to validate

#### Methods

##### Descriptor

Deprecated: Use ValidateTokenRequest.ProtoReflect.Descriptor instead.

##### GetToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ValidateTokenResponse

ValidateTokenResponse contains the validation result

#### Methods

##### Descriptor

Deprecated: Use ValidateTokenResponse.ProtoReflect.Descriptor instead.

##### GetRoles

##### GetUserId

##### GetValid

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterAuthServiceServer
