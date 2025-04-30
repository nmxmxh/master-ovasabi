# Package protos

## Constants

### EchoService_Echo_FullMethodName

## Variables

### EchoService_ServiceDesc

EchoService_ServiceDesc is the grpc.ServiceDesc for EchoService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

### File_api_protos_echo_proto

## Types

### EchoRequest

EchoRequest contains the message to echo

#### Methods

##### Descriptor

Deprecated: Use EchoRequest.ProtoReflect.Descriptor instead.

##### GetMessage

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### EchoResponse

EchoResponse contains the echoed message

#### Methods

##### Descriptor

Deprecated: Use EchoResponse.ProtoReflect.Descriptor instead.

##### GetMessage

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### EchoServiceClient

EchoServiceClient is the client API for EchoService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### EchoServiceServer

EchoServiceServer is the server API for EchoService service. All implementations must embed
UnimplementedEchoServiceServer for forward compatibility

### UnimplementedEchoServiceServer

UnimplementedEchoServiceServer must be embedded to have forward compatible implementations.

#### Methods

##### Echo

### UnsafeEchoServiceServer

UnsafeEchoServiceServer may be embedded to opt out of forward compatibility for this service. Use of
this interface is not recommended, as added methods to EchoServiceServer will result in compilation
errors.

## Functions

### RegisterEchoServiceServer
