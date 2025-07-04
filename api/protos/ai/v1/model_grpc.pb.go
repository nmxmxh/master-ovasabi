// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: ai/v1/model.proto

package aipb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	AIService_ProcessContent_FullMethodName     = "/ai.v1.AIService/ProcessContent"
	AIService_GenerateEmbeddings_FullMethodName = "/ai.v1.AIService/GenerateEmbeddings"
	AIService_SubmitModelUpdate_FullMethodName  = "/ai.v1.AIService/SubmitModelUpdate"
	AIService_GetCurrentModel_FullMethodName    = "/ai.v1.AIService/GetCurrentModel"
	AIService_HandleClientEvent_FullMethodName  = "/ai.v1.AIService/HandleClientEvent"
)

// AIServiceClient is the client API for AIService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// --- Service Definitions ---
type AIServiceClient interface {
	ProcessContent(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[EnrichmentRequest, EnrichmentResponse], error)
	GenerateEmbeddings(ctx context.Context, in *EnrichmentRequest, opts ...grpc.CallOption) (*EnrichmentResponse_Vector, error)
	SubmitModelUpdate(ctx context.Context, in *ModelUpdate, opts ...grpc.CallOption) (*ModelUpdateAck, error)
	GetCurrentModel(ctx context.Context, in *ModelRequest, opts ...grpc.CallOption) (*Model, error)
	// New client feedback endpoint
	HandleClientEvent(ctx context.Context, in *ClientEvent, opts ...grpc.CallOption) (*ClientEventAck, error)
}

type aIServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAIServiceClient(cc grpc.ClientConnInterface) AIServiceClient {
	return &aIServiceClient{cc}
}

func (c *aIServiceClient) ProcessContent(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[EnrichmentRequest, EnrichmentResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &AIService_ServiceDesc.Streams[0], AIService_ProcessContent_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[EnrichmentRequest, EnrichmentResponse]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type AIService_ProcessContentClient = grpc.ClientStreamingClient[EnrichmentRequest, EnrichmentResponse]

func (c *aIServiceClient) GenerateEmbeddings(ctx context.Context, in *EnrichmentRequest, opts ...grpc.CallOption) (*EnrichmentResponse_Vector, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(EnrichmentResponse_Vector)
	err := c.cc.Invoke(ctx, AIService_GenerateEmbeddings_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *aIServiceClient) SubmitModelUpdate(ctx context.Context, in *ModelUpdate, opts ...grpc.CallOption) (*ModelUpdateAck, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ModelUpdateAck)
	err := c.cc.Invoke(ctx, AIService_SubmitModelUpdate_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *aIServiceClient) GetCurrentModel(ctx context.Context, in *ModelRequest, opts ...grpc.CallOption) (*Model, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Model)
	err := c.cc.Invoke(ctx, AIService_GetCurrentModel_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *aIServiceClient) HandleClientEvent(ctx context.Context, in *ClientEvent, opts ...grpc.CallOption) (*ClientEventAck, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ClientEventAck)
	err := c.cc.Invoke(ctx, AIService_HandleClientEvent_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AIServiceServer is the server API for AIService service.
// All implementations must embed UnimplementedAIServiceServer
// for forward compatibility.
//
// --- Service Definitions ---
type AIServiceServer interface {
	ProcessContent(grpc.ClientStreamingServer[EnrichmentRequest, EnrichmentResponse]) error
	GenerateEmbeddings(context.Context, *EnrichmentRequest) (*EnrichmentResponse_Vector, error)
	SubmitModelUpdate(context.Context, *ModelUpdate) (*ModelUpdateAck, error)
	GetCurrentModel(context.Context, *ModelRequest) (*Model, error)
	// New client feedback endpoint
	HandleClientEvent(context.Context, *ClientEvent) (*ClientEventAck, error)
	mustEmbedUnimplementedAIServiceServer()
}

// UnimplementedAIServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedAIServiceServer struct{}

func (UnimplementedAIServiceServer) ProcessContent(grpc.ClientStreamingServer[EnrichmentRequest, EnrichmentResponse]) error {
	return status.Errorf(codes.Unimplemented, "method ProcessContent not implemented")
}
func (UnimplementedAIServiceServer) GenerateEmbeddings(context.Context, *EnrichmentRequest) (*EnrichmentResponse_Vector, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenerateEmbeddings not implemented")
}
func (UnimplementedAIServiceServer) SubmitModelUpdate(context.Context, *ModelUpdate) (*ModelUpdateAck, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubmitModelUpdate not implemented")
}
func (UnimplementedAIServiceServer) GetCurrentModel(context.Context, *ModelRequest) (*Model, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCurrentModel not implemented")
}
func (UnimplementedAIServiceServer) HandleClientEvent(context.Context, *ClientEvent) (*ClientEventAck, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HandleClientEvent not implemented")
}
func (UnimplementedAIServiceServer) mustEmbedUnimplementedAIServiceServer() {}
func (UnimplementedAIServiceServer) testEmbeddedByValue()                   {}

// UnsafeAIServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AIServiceServer will
// result in compilation errors.
type UnsafeAIServiceServer interface {
	mustEmbedUnimplementedAIServiceServer()
}

func RegisterAIServiceServer(s grpc.ServiceRegistrar, srv AIServiceServer) {
	// If the following call pancis, it indicates UnimplementedAIServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&AIService_ServiceDesc, srv)
}

func _AIService_ProcessContent_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(AIServiceServer).ProcessContent(&grpc.GenericServerStream[EnrichmentRequest, EnrichmentResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type AIService_ProcessContentServer = grpc.ClientStreamingServer[EnrichmentRequest, EnrichmentResponse]

func _AIService_GenerateEmbeddings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EnrichmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AIServiceServer).GenerateEmbeddings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: AIService_GenerateEmbeddings_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AIServiceServer).GenerateEmbeddings(ctx, req.(*EnrichmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AIService_SubmitModelUpdate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ModelUpdate)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AIServiceServer).SubmitModelUpdate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: AIService_SubmitModelUpdate_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AIServiceServer).SubmitModelUpdate(ctx, req.(*ModelUpdate))
	}
	return interceptor(ctx, in, info, handler)
}

func _AIService_GetCurrentModel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ModelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AIServiceServer).GetCurrentModel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: AIService_GetCurrentModel_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AIServiceServer).GetCurrentModel(ctx, req.(*ModelRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AIService_HandleClientEvent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClientEvent)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AIServiceServer).HandleClientEvent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: AIService_HandleClientEvent_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AIServiceServer).HandleClientEvent(ctx, req.(*ClientEvent))
	}
	return interceptor(ctx, in, info, handler)
}

// AIService_ServiceDesc is the grpc.ServiceDesc for AIService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AIService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ai.v1.AIService",
	HandlerType: (*AIServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GenerateEmbeddings",
			Handler:    _AIService_GenerateEmbeddings_Handler,
		},
		{
			MethodName: "SubmitModelUpdate",
			Handler:    _AIService_SubmitModelUpdate_Handler,
		},
		{
			MethodName: "GetCurrentModel",
			Handler:    _AIService_GetCurrentModel_Handler,
		},
		{
			MethodName: "HandleClientEvent",
			Handler:    _AIService_HandleClientEvent_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "ProcessContent",
			Handler:       _AIService_ProcessContent_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "ai/v1/model.proto",
}
