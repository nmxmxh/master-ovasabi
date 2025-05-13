# Package talentpb

## Constants

### TalentService_CreateTalentProfile_FullMethodName

## Variables

### File_talent_v1_talent_proto

### TalentService_ServiceDesc

TalentService_ServiceDesc is the grpc.ServiceDesc for TalentService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### BookTalentRequest

#### Methods

##### Descriptor

Deprecated: Use BookTalentRequest.ProtoReflect.Descriptor instead.

##### GetEndTime

##### GetMetadata

##### GetNotes

##### GetStartTime

##### GetTalentId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BookTalentResponse

#### Methods

##### Descriptor

Deprecated: Use BookTalentResponse.ProtoReflect.Descriptor instead.

##### GetBooking

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Booking

#### Methods

##### Descriptor

Deprecated: Use Booking.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetEndTime

##### GetId

##### GetMetadata

##### GetNotes

##### GetStartTime

##### GetStatus

##### GetTalentId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateTalentProfileRequest

#### Methods

##### Descriptor

Deprecated: Use CreateTalentProfileRequest.ProtoReflect.Descriptor instead.

##### GetProfile

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateTalentProfileResponse

#### Methods

##### Descriptor

Deprecated: Use CreateTalentProfileResponse.ProtoReflect.Descriptor instead.

##### GetProfile

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteTalentProfileRequest

#### Methods

##### Descriptor

Deprecated: Use DeleteTalentProfileRequest.ProtoReflect.Descriptor instead.

##### GetProfileId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteTalentProfileResponse

#### Methods

##### Descriptor

Deprecated: Use DeleteTalentProfileResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Education

#### Methods

##### Descriptor

Deprecated: Use Education.ProtoReflect.Descriptor instead.

##### GetDegree

##### GetEndDate

##### GetFieldOfStudy

##### GetInstitution

##### GetMetadata

##### GetStartDate

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Experience

#### Methods

##### Descriptor

Deprecated: Use Experience.ProtoReflect.Descriptor instead.

##### GetCompany

##### GetDescription

##### GetEndDate

##### GetMetadata

##### GetStartDate

##### GetTitle

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetTalentProfileRequest

#### Methods

##### Descriptor

Deprecated: Use GetTalentProfileRequest.ProtoReflect.Descriptor instead.

##### GetProfileId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetTalentProfileResponse

#### Methods

##### Descriptor

Deprecated: Use GetTalentProfileResponse.ProtoReflect.Descriptor instead.

##### GetProfile

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListBookingsRequest

#### Methods

##### Descriptor

Deprecated: Use ListBookingsRequest.ProtoReflect.Descriptor instead.

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListBookingsResponse

#### Methods

##### Descriptor

Deprecated: Use ListBookingsResponse.ProtoReflect.Descriptor instead.

##### GetBookings

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListTalentProfilesRequest

#### Methods

##### Descriptor

Deprecated: Use ListTalentProfilesRequest.ProtoReflect.Descriptor instead.

##### GetLocation

##### GetPage

##### GetPageSize

##### GetSkills

##### GetTags

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListTalentProfilesResponse

#### Methods

##### Descriptor

Deprecated: Use ListTalentProfilesResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetProfiles

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchTalentProfilesRequest

#### Methods

##### Descriptor

Deprecated: Use SearchTalentProfilesRequest.ProtoReflect.Descriptor instead.

##### GetLocation

##### GetPage

##### GetPageSize

##### GetQuery

##### GetSkills

##### GetTags

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SearchTalentProfilesResponse

#### Methods

##### Descriptor

Deprecated: Use SearchTalentProfilesResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetProfiles

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TalentProfile

#### Methods

##### Descriptor

Deprecated: Use TalentProfile.ProtoReflect.Descriptor instead.

##### GetAvatarUrl

##### GetBio

##### GetCreatedAt

##### GetDisplayName

##### GetEducations

##### GetExperiences

##### GetId

##### GetLocation

##### GetMasterId

##### GetMetadata

##### GetSkills

##### GetTags

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### TalentServiceClient

TalentServiceClient is the client API for TalentService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### TalentServiceServer

TalentServiceServer is the server API for TalentService service. All implementations must embed
UnimplementedTalentServiceServer for forward compatibility.

### UnimplementedTalentServiceServer

UnimplementedTalentServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### BookTalent

##### CreateTalentProfile

##### DeleteTalentProfile

##### GetTalentProfile

##### ListBookings

##### ListTalentProfiles

##### SearchTalentProfiles

##### UpdateTalentProfile

### UnsafeTalentServiceServer

UnsafeTalentServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to TalentServiceServer will result in
compilation errors.

### UpdateTalentProfileRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateTalentProfileRequest.ProtoReflect.Descriptor instead.

##### GetProfile

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateTalentProfileResponse

#### Methods

##### Descriptor

Deprecated: Use UpdateTalentProfileResponse.ProtoReflect.Descriptor instead.

##### GetProfile

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterTalentServiceServer
