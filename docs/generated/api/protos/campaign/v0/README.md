# Package campaign

## Constants

### CampaignService_CreateCampaign_FullMethodName

## Variables

### CampaignService_ServiceDesc

CampaignService_ServiceDesc is the grpc.ServiceDesc for CampaignService service. It's only intended
for direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

### File_api_protos_campaign_v0_campaign_proto

## Types

### Campaign

#### Methods

##### Descriptor

Deprecated: Use Campaign.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetDescription

##### GetEndDate

##### GetId

##### GetMetadata

##### GetRankingFormula

##### GetSlug

##### GetStartDate

##### GetTitle

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CampaignServiceClient

CampaignServiceClient is the client API for CampaignService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### CampaignServiceServer

CampaignServiceServer is the server API for CampaignService service. All implementations must embed
UnimplementedCampaignServiceServer for forward compatibility

### CreateCampaignRequest

#### Methods

##### Descriptor

Deprecated: Use CreateCampaignRequest.ProtoReflect.Descriptor instead.

##### GetDescription

##### GetEndDate

##### GetMetadata

##### GetRankingFormula

##### GetSlug

##### GetStartDate

##### GetTitle

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateCampaignResponse

#### Methods

##### Descriptor

Deprecated: Use CreateCampaignResponse.ProtoReflect.Descriptor instead.

##### GetCampaign

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteCampaignRequest

#### Methods

##### Descriptor

Deprecated: Use DeleteCampaignRequest.ProtoReflect.Descriptor instead.

##### GetId

##### GetSlug

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteCampaignResponse

#### Methods

##### Descriptor

Deprecated: Use DeleteCampaignResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetCampaignRequest

#### Methods

##### Descriptor

Deprecated: Use GetCampaignRequest.ProtoReflect.Descriptor instead.

##### GetSlug

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetCampaignResponse

#### Methods

##### Descriptor

Deprecated: Use GetCampaignResponse.ProtoReflect.Descriptor instead.

##### GetCampaign

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListCampaignsRequest

#### Methods

##### Descriptor

Deprecated: Use ListCampaignsRequest.ProtoReflect.Descriptor instead.

##### GetPage

##### GetPageSize

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListCampaignsResponse

#### Methods

##### Descriptor

Deprecated: Use ListCampaignsResponse.ProtoReflect.Descriptor instead.

##### GetCampaigns

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedCampaignServiceServer

UnimplementedCampaignServiceServer must be embedded to have forward compatible implementations.

#### Methods

##### CreateCampaign

##### DeleteCampaign

##### GetCampaign

##### ListCampaigns

##### UpdateCampaign

### UnsafeCampaignServiceServer

UnsafeCampaignServiceServer may be embedded to opt out of forward compatibility for this service.
Use of this interface is not recommended, as added methods to CampaignServiceServer will result in
compilation errors.

### UpdateCampaignRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateCampaignRequest.ProtoReflect.Descriptor instead.

##### GetCampaign

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateCampaignResponse

#### Methods

##### Descriptor

Deprecated: Use UpdateCampaignResponse.ProtoReflect.Descriptor instead.

##### GetCampaign

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterCampaignServiceServer
