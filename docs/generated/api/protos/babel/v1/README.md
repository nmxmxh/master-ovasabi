# Package babel

## Constants

### BabelService_GetLocationContext_FullMethodName

## Variables

### BabelService_ServiceDesc

BabelService_ServiceDesc is the grpc.ServiceDesc for BabelService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

### File_api_protos_babel_v1_babel_proto

## Types

### BabelServiceClient

BabelServiceClient is the client API for BabelService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### BabelServiceServer

BabelServiceServer is the server API for BabelService service. All implementations must embed
UnimplementedBabelServiceServer for forward compatibility.

### GetLocationContextRequest

#### Methods

##### Descriptor

Deprecated: Use GetLocationContextRequest.ProtoReflect.Descriptor instead.

##### GetCity

##### GetCountryCode

##### GetRegion

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetLocationContextResponse

#### Methods

##### Descriptor

Deprecated: Use GetLocationContextResponse.ProtoReflect.Descriptor instead.

##### GetRule

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### PricingRule

#### Methods

##### Descriptor

Deprecated: Use PricingRule.ProtoReflect.Descriptor instead.

##### GetAffluenceTier

##### GetBasePrice

##### GetCity

##### GetCountryCode

##### GetCreatedAt

##### GetCurrencyCode

##### GetDemandLevel

##### GetEffectiveFrom

##### GetEffectiveTo

##### GetId

##### GetKgEntityId

##### GetMultiplier

##### GetNotes

##### GetRegion

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TranslateRequest

#### Methods

##### Descriptor

Deprecated: Use TranslateRequest.ProtoReflect.Descriptor instead.

##### GetKey

##### GetLocale

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TranslateResponse

#### Methods

##### Descriptor

Deprecated: Use TranslateResponse.ProtoReflect.Descriptor instead.

##### GetValue

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedBabelServiceServer

UnimplementedBabelServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### GetLocationContext

##### Translate

### UnsafeBabelServiceServer

UnsafeBabelServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to BabelServiceServer will result in
compilation errors.

## Functions

### RegisterBabelServiceServer
