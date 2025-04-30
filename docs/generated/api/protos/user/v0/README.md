# Package userv0

## Constants

### UserService_CreateUser_FullMethodName

## Variables

### UserStatus_name

Enum value maps for UserStatus.

### File_api_protos_user_v0_user_proto

### UserService_ServiceDesc

UserService_ServiceDesc is the grpc.ServiceDesc for UserService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### CreateReferralRequest

#### Methods

##### Descriptor

Deprecated: Use CreateReferralRequest.ProtoReflect.Descriptor instead.

##### GetCampaignSlug

##### GetDeviceHash

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateReferralResponse

#### Methods

##### Descriptor

Deprecated: Use CreateReferralResponse.ProtoReflect.Descriptor instead.

##### GetReferralCode

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateUserRequest

CreateUserRequest represents the request to create a new user

#### Methods

##### Descriptor

Deprecated: Use CreateUserRequest.ProtoReflect.Descriptor instead.

##### GetEmail

##### GetMetadata

##### GetPassword

##### GetProfile

##### GetRoles

##### GetUsername

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateUserResponse

CreateUserResponse represents the response from creating a user

#### Methods

##### Descriptor

Deprecated: Use CreateUserResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteUserRequest

DeleteUserRequest represents the request to delete a user

#### Methods

##### Descriptor

Deprecated: Use DeleteUserRequest.ProtoReflect.Descriptor instead.

##### GetHardDelete

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteUserResponse

DeleteUserResponse represents the response from deleting a user

#### Methods

##### Descriptor

Deprecated: Use DeleteUserResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserByUsernameRequest

GetUserByUsernameRequest represents the request to get user information by username

#### Methods

##### Descriptor

Deprecated: Use GetUserByUsernameRequest.ProtoReflect.Descriptor instead.

##### GetUsername

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserByUsernameResponse

GetUserByUsernameResponse represents the response containing user information by username

#### Methods

##### Descriptor

Deprecated: Use GetUserByUsernameResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserRequest

GetUserRequest represents the request to get user information

#### Methods

##### Descriptor

Deprecated: Use GetUserRequest.ProtoReflect.Descriptor instead.

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserResponse

GetUserResponse represents the response containing user information

#### Methods

##### Descriptor

Deprecated: Use GetUserResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUsersRequest

ListUsersRequest represents the request to list users

#### Methods

##### Descriptor

Deprecated: Use ListUsersRequest.ProtoReflect.Descriptor instead.

##### GetFilters

##### GetPage

##### GetPageSize

##### GetSortBy

##### GetSortDesc

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUsersResponse

ListUsersResponse represents the response containing a list of users

#### Methods

##### Descriptor

Deprecated: Use ListUsersResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### GetUsers

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RegisterInterestRequest

#### Methods

##### Descriptor

Deprecated: Use RegisterInterestRequest.ProtoReflect.Descriptor instead.

##### GetDeviceHash

##### GetEmail

##### GetLocation

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RegisterInterestResponse

#### Methods

##### Descriptor

Deprecated: Use RegisterInterestResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedUserServiceServer

UnimplementedUserServiceServer must be embedded to have forward compatible implementations.

#### Methods

##### CreateReferral

##### CreateUser

##### DeleteUser

##### GetUser

##### GetUserByUsername

##### ListUsers

##### RegisterInterest

##### UpdatePassword

##### UpdateProfile

##### UpdateUser

### UnsafeUserServiceServer

UnsafeUserServiceServer may be embedded to opt out of forward compatibility for this service. Use of
this interface is not recommended, as added methods to UserServiceServer will result in compilation
errors.

### UpdatePasswordRequest

UpdatePasswordRequest represents the request to update a user's password

#### Methods

##### Descriptor

Deprecated: Use UpdatePasswordRequest.ProtoReflect.Descriptor instead.

##### GetCurrentPassword

##### GetNewPassword

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdatePasswordResponse

UpdatePasswordResponse represents the response from updating a password

#### Methods

##### Descriptor

Deprecated: Use UpdatePasswordResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateProfileRequest

UpdateProfileRequest represents the request to update a user's profile

#### Methods

##### Descriptor

Deprecated: Use UpdateProfileRequest.ProtoReflect.Descriptor instead.

##### GetFieldsToUpdate

##### GetProfile

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateProfileResponse

UpdateProfileResponse represents the response from updating a profile

#### Methods

##### Descriptor

Deprecated: Use UpdateProfileResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateUserRequest

UpdateUserRequest represents the request to update user information

#### Methods

##### Descriptor

Deprecated: Use UpdateUserRequest.ProtoReflect.Descriptor instead.

##### GetFieldsToUpdate

##### GetUser

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateUserResponse

UpdateUserResponse represents the response from updating user information

#### Methods

##### Descriptor

Deprecated: Use UpdateUserResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### User

User represents a user in the system

#### Methods

##### Descriptor

Deprecated: Use User.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetDeviceHash

##### GetEmail

##### GetId

##### GetLocation

##### GetMetadata

##### GetPasswordHash

##### GetReferralCode

##### GetReferredBy

##### GetUpdatedAt

##### GetUsername

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UserProfile

UserProfile contains additional user information

#### Methods

##### Descriptor

Deprecated: Use UserProfile.ProtoReflect.Descriptor instead.

##### GetAvatarUrl

##### GetBio

##### GetCustomFields

##### GetFirstName

##### GetLanguage

##### GetLastName

##### GetPhoneNumber

##### GetTimezone

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UserServiceClient

UserServiceClient is the client API for UserService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

### UserServiceServer

UserServiceServer is the server API for UserService service. All implementations must embed
UnimplementedUserServiceServer for forward compatibility

### UserStatus

UserStatus represents the user's account status

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use UserStatus.Descriptor instead.

##### Number

##### String

##### Type

## Functions

### RegisterUserServiceServer
