# Package referral

## Constants

### ReferralService_CreateReferral_FullMethodName

## Variables

### File_api_protos_referral_v0_referral_proto

### ReferralService_ServiceDesc

ReferralService_ServiceDesc is the grpc.ServiceDesc for ReferralService service. It's only intended
for direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### CreateReferralRequest

CreateReferralRequest contains referral creation parameters

#### Methods

##### Descriptor

Deprecated: Use CreateReferralRequest.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetDeviceHash

##### GetReferrerMasterId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateReferralResponse

CreateReferralResponse contains the created referral code

#### Methods

##### Descriptor

Deprecated: Use CreateReferralResponse.ProtoReflect.Descriptor instead.

##### GetReferral

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetReferralRequest

GetReferralRequest contains the referral code to retrieve

#### Methods

##### Descriptor

Deprecated: Use GetReferralRequest.ProtoReflect.Descriptor instead.

##### GetReferralCode

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetReferralResponse

GetReferralResponse contains the retrieved referral

#### Methods

##### Descriptor

Deprecated: Use GetReferralResponse.ProtoReflect.Descriptor instead.

##### GetReferral

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetReferralStatsRequest

GetReferralStatsRequest contains the user identifier

#### Methods

##### Descriptor

Deprecated: Use GetReferralStatsRequest.ProtoReflect.Descriptor instead.

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetReferralStatsResponse

GetReferralStatsResponse contains referral statistics

#### Methods

##### Descriptor

Deprecated: Use GetReferralStatsResponse.ProtoReflect.Descriptor instead.

##### GetActiveReferrals

##### GetReferrals

##### GetTotalReferrals

##### GetTotalRewards

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Referral

Referral contains information about a referral

#### Methods

##### Descriptor

Deprecated: Use Referral.ProtoReflect.Descriptor instead.

##### GetCampaignId

##### GetCreatedAt

##### GetDeviceHash

##### GetId

##### GetReferralCode

##### GetReferrerMasterId

##### GetSuccessful

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ReferralDetail

ReferralDetail contains information about a specific referral

#### Methods

##### Descriptor

Deprecated: Use ReferralDetail.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetIsActive

##### GetReferralCode

##### GetReferredUserId

##### GetRewardPoints

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ReferralServiceClient

ReferralServiceClient is the client API for ReferralService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### ReferralServiceServer

ReferralServiceServer is the server API for ReferralService service. All implementations must embed
UnimplementedReferralServiceServer for forward compatibility

### UnimplementedReferralServiceServer

UnimplementedReferralServiceServer must be embedded to have forward compatible implementations.

#### Methods

##### CreateReferral

##### GetReferral

##### GetReferralStats

### UnsafeReferralServiceServer

UnsafeReferralServiceServer may be embedded to opt out of forward compatibility for this service.
Use of this interface is not recommended, as added methods to ReferralServiceServer will result in
compilation errors.

## Functions

### RegisterReferralServiceServer
