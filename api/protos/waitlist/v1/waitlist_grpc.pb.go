// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: waitlist/v1/waitlist.proto

package waitlistpb

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
	WaitlistService_CreateWaitlistEntry_FullMethodName       = "/waitlist.v1.WaitlistService/CreateWaitlistEntry"
	WaitlistService_GetWaitlistEntry_FullMethodName          = "/waitlist.v1.WaitlistService/GetWaitlistEntry"
	WaitlistService_UpdateWaitlistEntry_FullMethodName       = "/waitlist.v1.WaitlistService/UpdateWaitlistEntry"
	WaitlistService_ListWaitlistEntries_FullMethodName       = "/waitlist.v1.WaitlistService/ListWaitlistEntries"
	WaitlistService_InviteUser_FullMethodName                = "/waitlist.v1.WaitlistService/InviteUser"
	WaitlistService_CheckUsernameAvailability_FullMethodName = "/waitlist.v1.WaitlistService/CheckUsernameAvailability"
	WaitlistService_ValidateReferralUsername_FullMethodName  = "/waitlist.v1.WaitlistService/ValidateReferralUsername"
	WaitlistService_GetLeaderboard_FullMethodName            = "/waitlist.v1.WaitlistService/GetLeaderboard"
	WaitlistService_GetReferralsByUser_FullMethodName        = "/waitlist.v1.WaitlistService/GetReferralsByUser"
	WaitlistService_GetLocationStats_FullMethodName          = "/waitlist.v1.WaitlistService/GetLocationStats"
	WaitlistService_GetWaitlistStats_FullMethodName          = "/waitlist.v1.WaitlistService/GetWaitlistStats"
	WaitlistService_GetWaitlistPosition_FullMethodName       = "/waitlist.v1.WaitlistService/GetWaitlistPosition"
)

// WaitlistServiceClient is the client API for WaitlistService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Waitlist service definition
type WaitlistServiceClient interface {
	// Create a new waitlist entry
	CreateWaitlistEntry(ctx context.Context, in *CreateWaitlistEntryRequest, opts ...grpc.CallOption) (*CreateWaitlistEntryResponse, error)
	// Get waitlist entry by ID, UUID, or email
	GetWaitlistEntry(ctx context.Context, in *GetWaitlistEntryRequest, opts ...grpc.CallOption) (*GetWaitlistEntryResponse, error)
	// Update an existing waitlist entry
	UpdateWaitlistEntry(ctx context.Context, in *UpdateWaitlistEntryRequest, opts ...grpc.CallOption) (*UpdateWaitlistEntryResponse, error)
	// List waitlist entries with pagination and filters
	ListWaitlistEntries(ctx context.Context, in *ListWaitlistEntriesRequest, opts ...grpc.CallOption) (*ListWaitlistEntriesResponse, error)
	// Invite a user (update status to invited)
	InviteUser(ctx context.Context, in *InviteUserRequest, opts ...grpc.CallOption) (*InviteUserResponse, error)
	// Check if username is available
	CheckUsernameAvailability(ctx context.Context, in *CheckUsernameAvailabilityRequest, opts ...grpc.CallOption) (*CheckUsernameAvailabilityResponse, error)
	// Validate referral username
	ValidateReferralUsername(ctx context.Context, in *ValidateReferralUsernameRequest, opts ...grpc.CallOption) (*ValidateReferralUsernameResponse, error)
	// Get referral leaderboard
	GetLeaderboard(ctx context.Context, in *GetLeaderboardRequest, opts ...grpc.CallOption) (*GetLeaderboardResponse, error)
	// Get referrals made by a user
	GetReferralsByUser(ctx context.Context, in *GetReferralsByUserRequest, opts ...grpc.CallOption) (*GetReferralsByUserResponse, error)
	// Get location-based statistics
	GetLocationStats(ctx context.Context, in *GetLocationStatsRequest, opts ...grpc.CallOption) (*GetLocationStatsResponse, error)
	// Get waitlist statistics
	GetWaitlistStats(ctx context.Context, in *GetWaitlistStatsRequest, opts ...grpc.CallOption) (*GetWaitlistStatsResponse, error)
	// Get user's waitlist position
	GetWaitlistPosition(ctx context.Context, in *GetWaitlistPositionRequest, opts ...grpc.CallOption) (*GetWaitlistPositionResponse, error)
}

type waitlistServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewWaitlistServiceClient(cc grpc.ClientConnInterface) WaitlistServiceClient {
	return &waitlistServiceClient{cc}
}

func (c *waitlistServiceClient) CreateWaitlistEntry(ctx context.Context, in *CreateWaitlistEntryRequest, opts ...grpc.CallOption) (*CreateWaitlistEntryResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CreateWaitlistEntryResponse)
	err := c.cc.Invoke(ctx, WaitlistService_CreateWaitlistEntry_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *waitlistServiceClient) GetWaitlistEntry(ctx context.Context, in *GetWaitlistEntryRequest, opts ...grpc.CallOption) (*GetWaitlistEntryResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetWaitlistEntryResponse)
	err := c.cc.Invoke(ctx, WaitlistService_GetWaitlistEntry_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *waitlistServiceClient) UpdateWaitlistEntry(ctx context.Context, in *UpdateWaitlistEntryRequest, opts ...grpc.CallOption) (*UpdateWaitlistEntryResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdateWaitlistEntryResponse)
	err := c.cc.Invoke(ctx, WaitlistService_UpdateWaitlistEntry_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *waitlistServiceClient) ListWaitlistEntries(ctx context.Context, in *ListWaitlistEntriesRequest, opts ...grpc.CallOption) (*ListWaitlistEntriesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ListWaitlistEntriesResponse)
	err := c.cc.Invoke(ctx, WaitlistService_ListWaitlistEntries_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *waitlistServiceClient) InviteUser(ctx context.Context, in *InviteUserRequest, opts ...grpc.CallOption) (*InviteUserResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(InviteUserResponse)
	err := c.cc.Invoke(ctx, WaitlistService_InviteUser_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *waitlistServiceClient) CheckUsernameAvailability(ctx context.Context, in *CheckUsernameAvailabilityRequest, opts ...grpc.CallOption) (*CheckUsernameAvailabilityResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CheckUsernameAvailabilityResponse)
	err := c.cc.Invoke(ctx, WaitlistService_CheckUsernameAvailability_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *waitlistServiceClient) ValidateReferralUsername(ctx context.Context, in *ValidateReferralUsernameRequest, opts ...grpc.CallOption) (*ValidateReferralUsernameResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ValidateReferralUsernameResponse)
	err := c.cc.Invoke(ctx, WaitlistService_ValidateReferralUsername_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *waitlistServiceClient) GetLeaderboard(ctx context.Context, in *GetLeaderboardRequest, opts ...grpc.CallOption) (*GetLeaderboardResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetLeaderboardResponse)
	err := c.cc.Invoke(ctx, WaitlistService_GetLeaderboard_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *waitlistServiceClient) GetReferralsByUser(ctx context.Context, in *GetReferralsByUserRequest, opts ...grpc.CallOption) (*GetReferralsByUserResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetReferralsByUserResponse)
	err := c.cc.Invoke(ctx, WaitlistService_GetReferralsByUser_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *waitlistServiceClient) GetLocationStats(ctx context.Context, in *GetLocationStatsRequest, opts ...grpc.CallOption) (*GetLocationStatsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetLocationStatsResponse)
	err := c.cc.Invoke(ctx, WaitlistService_GetLocationStats_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *waitlistServiceClient) GetWaitlistStats(ctx context.Context, in *GetWaitlistStatsRequest, opts ...grpc.CallOption) (*GetWaitlistStatsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetWaitlistStatsResponse)
	err := c.cc.Invoke(ctx, WaitlistService_GetWaitlistStats_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *waitlistServiceClient) GetWaitlistPosition(ctx context.Context, in *GetWaitlistPositionRequest, opts ...grpc.CallOption) (*GetWaitlistPositionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetWaitlistPositionResponse)
	err := c.cc.Invoke(ctx, WaitlistService_GetWaitlistPosition_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WaitlistServiceServer is the server API for WaitlistService service.
// All implementations must embed UnimplementedWaitlistServiceServer
// for forward compatibility.
//
// Waitlist service definition
type WaitlistServiceServer interface {
	// Create a new waitlist entry
	CreateWaitlistEntry(context.Context, *CreateWaitlistEntryRequest) (*CreateWaitlistEntryResponse, error)
	// Get waitlist entry by ID, UUID, or email
	GetWaitlistEntry(context.Context, *GetWaitlistEntryRequest) (*GetWaitlistEntryResponse, error)
	// Update an existing waitlist entry
	UpdateWaitlistEntry(context.Context, *UpdateWaitlistEntryRequest) (*UpdateWaitlistEntryResponse, error)
	// List waitlist entries with pagination and filters
	ListWaitlistEntries(context.Context, *ListWaitlistEntriesRequest) (*ListWaitlistEntriesResponse, error)
	// Invite a user (update status to invited)
	InviteUser(context.Context, *InviteUserRequest) (*InviteUserResponse, error)
	// Check if username is available
	CheckUsernameAvailability(context.Context, *CheckUsernameAvailabilityRequest) (*CheckUsernameAvailabilityResponse, error)
	// Validate referral username
	ValidateReferralUsername(context.Context, *ValidateReferralUsernameRequest) (*ValidateReferralUsernameResponse, error)
	// Get referral leaderboard
	GetLeaderboard(context.Context, *GetLeaderboardRequest) (*GetLeaderboardResponse, error)
	// Get referrals made by a user
	GetReferralsByUser(context.Context, *GetReferralsByUserRequest) (*GetReferralsByUserResponse, error)
	// Get location-based statistics
	GetLocationStats(context.Context, *GetLocationStatsRequest) (*GetLocationStatsResponse, error)
	// Get waitlist statistics
	GetWaitlistStats(context.Context, *GetWaitlistStatsRequest) (*GetWaitlistStatsResponse, error)
	// Get user's waitlist position
	GetWaitlistPosition(context.Context, *GetWaitlistPositionRequest) (*GetWaitlistPositionResponse, error)
	mustEmbedUnimplementedWaitlistServiceServer()
}

// UnimplementedWaitlistServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedWaitlistServiceServer struct{}

func (UnimplementedWaitlistServiceServer) CreateWaitlistEntry(context.Context, *CreateWaitlistEntryRequest) (*CreateWaitlistEntryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateWaitlistEntry not implemented")
}
func (UnimplementedWaitlistServiceServer) GetWaitlistEntry(context.Context, *GetWaitlistEntryRequest) (*GetWaitlistEntryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetWaitlistEntry not implemented")
}
func (UnimplementedWaitlistServiceServer) UpdateWaitlistEntry(context.Context, *UpdateWaitlistEntryRequest) (*UpdateWaitlistEntryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateWaitlistEntry not implemented")
}
func (UnimplementedWaitlistServiceServer) ListWaitlistEntries(context.Context, *ListWaitlistEntriesRequest) (*ListWaitlistEntriesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListWaitlistEntries not implemented")
}
func (UnimplementedWaitlistServiceServer) InviteUser(context.Context, *InviteUserRequest) (*InviteUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InviteUser not implemented")
}
func (UnimplementedWaitlistServiceServer) CheckUsernameAvailability(context.Context, *CheckUsernameAvailabilityRequest) (*CheckUsernameAvailabilityResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckUsernameAvailability not implemented")
}
func (UnimplementedWaitlistServiceServer) ValidateReferralUsername(context.Context, *ValidateReferralUsernameRequest) (*ValidateReferralUsernameResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ValidateReferralUsername not implemented")
}
func (UnimplementedWaitlistServiceServer) GetLeaderboard(context.Context, *GetLeaderboardRequest) (*GetLeaderboardResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLeaderboard not implemented")
}
func (UnimplementedWaitlistServiceServer) GetReferralsByUser(context.Context, *GetReferralsByUserRequest) (*GetReferralsByUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetReferralsByUser not implemented")
}
func (UnimplementedWaitlistServiceServer) GetLocationStats(context.Context, *GetLocationStatsRequest) (*GetLocationStatsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetLocationStats not implemented")
}
func (UnimplementedWaitlistServiceServer) GetWaitlistStats(context.Context, *GetWaitlistStatsRequest) (*GetWaitlistStatsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetWaitlistStats not implemented")
}
func (UnimplementedWaitlistServiceServer) GetWaitlistPosition(context.Context, *GetWaitlistPositionRequest) (*GetWaitlistPositionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetWaitlistPosition not implemented")
}
func (UnimplementedWaitlistServiceServer) mustEmbedUnimplementedWaitlistServiceServer() {}
func (UnimplementedWaitlistServiceServer) testEmbeddedByValue()                         {}

// UnsafeWaitlistServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WaitlistServiceServer will
// result in compilation errors.
type UnsafeWaitlistServiceServer interface {
	mustEmbedUnimplementedWaitlistServiceServer()
}

func RegisterWaitlistServiceServer(s grpc.ServiceRegistrar, srv WaitlistServiceServer) {
	// If the following call pancis, it indicates UnimplementedWaitlistServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&WaitlistService_ServiceDesc, srv)
}

func _WaitlistService_CreateWaitlistEntry_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateWaitlistEntryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).CreateWaitlistEntry(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_CreateWaitlistEntry_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).CreateWaitlistEntry(ctx, req.(*CreateWaitlistEntryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WaitlistService_GetWaitlistEntry_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetWaitlistEntryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).GetWaitlistEntry(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_GetWaitlistEntry_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).GetWaitlistEntry(ctx, req.(*GetWaitlistEntryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WaitlistService_UpdateWaitlistEntry_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateWaitlistEntryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).UpdateWaitlistEntry(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_UpdateWaitlistEntry_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).UpdateWaitlistEntry(ctx, req.(*UpdateWaitlistEntryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WaitlistService_ListWaitlistEntries_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListWaitlistEntriesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).ListWaitlistEntries(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_ListWaitlistEntries_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).ListWaitlistEntries(ctx, req.(*ListWaitlistEntriesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WaitlistService_InviteUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InviteUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).InviteUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_InviteUser_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).InviteUser(ctx, req.(*InviteUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WaitlistService_CheckUsernameAvailability_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckUsernameAvailabilityRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).CheckUsernameAvailability(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_CheckUsernameAvailability_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).CheckUsernameAvailability(ctx, req.(*CheckUsernameAvailabilityRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WaitlistService_ValidateReferralUsername_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ValidateReferralUsernameRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).ValidateReferralUsername(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_ValidateReferralUsername_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).ValidateReferralUsername(ctx, req.(*ValidateReferralUsernameRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WaitlistService_GetLeaderboard_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetLeaderboardRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).GetLeaderboard(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_GetLeaderboard_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).GetLeaderboard(ctx, req.(*GetLeaderboardRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WaitlistService_GetReferralsByUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetReferralsByUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).GetReferralsByUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_GetReferralsByUser_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).GetReferralsByUser(ctx, req.(*GetReferralsByUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WaitlistService_GetLocationStats_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetLocationStatsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).GetLocationStats(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_GetLocationStats_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).GetLocationStats(ctx, req.(*GetLocationStatsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WaitlistService_GetWaitlistStats_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetWaitlistStatsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).GetWaitlistStats(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_GetWaitlistStats_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).GetWaitlistStats(ctx, req.(*GetWaitlistStatsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _WaitlistService_GetWaitlistPosition_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetWaitlistPositionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WaitlistServiceServer).GetWaitlistPosition(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WaitlistService_GetWaitlistPosition_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WaitlistServiceServer).GetWaitlistPosition(ctx, req.(*GetWaitlistPositionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// WaitlistService_ServiceDesc is the grpc.ServiceDesc for WaitlistService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var WaitlistService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "waitlist.v1.WaitlistService",
	HandlerType: (*WaitlistServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateWaitlistEntry",
			Handler:    _WaitlistService_CreateWaitlistEntry_Handler,
		},
		{
			MethodName: "GetWaitlistEntry",
			Handler:    _WaitlistService_GetWaitlistEntry_Handler,
		},
		{
			MethodName: "UpdateWaitlistEntry",
			Handler:    _WaitlistService_UpdateWaitlistEntry_Handler,
		},
		{
			MethodName: "ListWaitlistEntries",
			Handler:    _WaitlistService_ListWaitlistEntries_Handler,
		},
		{
			MethodName: "InviteUser",
			Handler:    _WaitlistService_InviteUser_Handler,
		},
		{
			MethodName: "CheckUsernameAvailability",
			Handler:    _WaitlistService_CheckUsernameAvailability_Handler,
		},
		{
			MethodName: "ValidateReferralUsername",
			Handler:    _WaitlistService_ValidateReferralUsername_Handler,
		},
		{
			MethodName: "GetLeaderboard",
			Handler:    _WaitlistService_GetLeaderboard_Handler,
		},
		{
			MethodName: "GetReferralsByUser",
			Handler:    _WaitlistService_GetReferralsByUser_Handler,
		},
		{
			MethodName: "GetLocationStats",
			Handler:    _WaitlistService_GetLocationStats_Handler,
		},
		{
			MethodName: "GetWaitlistStats",
			Handler:    _WaitlistService_GetWaitlistStats_Handler,
		},
		{
			MethodName: "GetWaitlistPosition",
			Handler:    _WaitlistService_GetWaitlistPosition_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "waitlist/v1/waitlist.proto",
}
