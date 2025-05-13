# Package userv1

## Constants

### UserService_CreateUser_FullMethodName

## Variables

### UserStatus_name

Enum value maps for UserStatus.

### File_user_v1_user_proto

### UserService_ServiceDesc

UserService_ServiceDesc is the grpc.ServiceDesc for UserService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### AssignRoleRequest

--- RBAC & Permissions ---

#### Methods

##### Descriptor

Deprecated: Use AssignRoleRequest.ProtoReflect.Descriptor instead.

##### GetRole

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AssignRoleResponse

#### Methods

##### Descriptor

Deprecated: Use AssignRoleResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AuditLog

#### Methods

##### Descriptor

Deprecated: Use AuditLog.ProtoReflect.Descriptor instead.

##### GetAction

##### GetId

##### GetMasterId

##### GetMetadata

##### GetOccurredAt

##### GetPayload

##### GetResource

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

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

### CreateSessionRequest

--- Session Management ---

#### Methods

##### Descriptor

Deprecated: Use CreateSessionRequest.ProtoReflect.Descriptor instead.

##### GetDeviceInfo

##### GetPassword

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateSessionResponse

#### Methods

##### Descriptor

Deprecated: Use CreateSessionResponse.ProtoReflect.Descriptor instead.

##### GetSession

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateUserRequest

--- User Management Requests/Responses (existing, updated for UUID) ---

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

#### Methods

##### Descriptor

Deprecated: Use CreateUserResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteUserRequest

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

#### Methods

##### Descriptor

Deprecated: Use DeleteUserResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetSessionRequest

#### Methods

##### Descriptor

Deprecated: Use GetSessionRequest.ProtoReflect.Descriptor instead.

##### GetSessionId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetSessionResponse

#### Methods

##### Descriptor

Deprecated: Use GetSessionResponse.ProtoReflect.Descriptor instead.

##### GetSession

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserByEmailRequest

#### Methods

##### Descriptor

Deprecated: Use GetUserByEmailRequest.ProtoReflect.Descriptor instead.

##### GetEmail

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserByEmailResponse

#### Methods

##### Descriptor

Deprecated: Use GetUserByEmailResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserByUsernameRequest

#### Methods

##### Descriptor

Deprecated: Use GetUserByUsernameRequest.ProtoReflect.Descriptor instead.

##### GetUsername

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserByUsernameResponse

#### Methods

##### Descriptor

Deprecated: Use GetUserByUsernameResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserRequest

#### Methods

##### Descriptor

Deprecated: Use GetUserRequest.ProtoReflect.Descriptor instead.

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetUserResponse

#### Methods

##### Descriptor

Deprecated: Use GetUserResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### InitiateMFARequest

#### Methods

##### Descriptor

Deprecated: Use InitiateMFARequest.ProtoReflect.Descriptor instead.

##### GetMfaType

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### InitiateMFAResponse

#### Methods

##### Descriptor

Deprecated: Use InitiateMFAResponse.ProtoReflect.Descriptor instead.

##### GetChallengeId

##### GetInitiated

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### InitiateSSORequest

--- SSO, MFA, SCIM Extensibility (placeholders) ---

#### Methods

##### Descriptor

Deprecated: Use InitiateSSORequest.ProtoReflect.Descriptor instead.

##### GetProvider

##### GetRedirectUri

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### InitiateSSOResponse

#### Methods

##### Descriptor

Deprecated: Use InitiateSSOResponse.ProtoReflect.Descriptor instead.

##### GetSsoUrl

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListAuditLogsRequest

#### Methods

##### Descriptor

Deprecated: Use ListAuditLogsRequest.ProtoReflect.Descriptor instead.

##### GetPage

##### GetPageSize

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListAuditLogsResponse

#### Methods

##### Descriptor

Deprecated: Use ListAuditLogsResponse.ProtoReflect.Descriptor instead.

##### GetLogs

##### GetTotalCount

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListPermissionsRequest

#### Methods

##### Descriptor

Deprecated: Use ListPermissionsRequest.ProtoReflect.Descriptor instead.

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListPermissionsResponse

#### Methods

##### Descriptor

Deprecated: Use ListPermissionsResponse.ProtoReflect.Descriptor instead.

##### GetPermissions

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListRolesRequest

#### Methods

##### Descriptor

Deprecated: Use ListRolesRequest.ProtoReflect.Descriptor instead.

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListRolesResponse

#### Methods

##### Descriptor

Deprecated: Use ListRolesResponse.ProtoReflect.Descriptor instead.

##### GetRoles

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListSessionsRequest

#### Methods

##### Descriptor

Deprecated: Use ListSessionsRequest.ProtoReflect.Descriptor instead.

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListSessionsResponse

#### Methods

##### Descriptor

Deprecated: Use ListSessionsResponse.ProtoReflect.Descriptor instead.

##### GetSessions

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUserEventsRequest

--- Audit/Event Log ---

#### Methods

##### Descriptor

Deprecated: Use ListUserEventsRequest.ProtoReflect.Descriptor instead.

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUserEventsResponse

#### Methods

##### Descriptor

Deprecated: Use ListUserEventsResponse.ProtoReflect.Descriptor instead.

##### GetEvents

##### GetTotalCount

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUsersRequest

#### Methods

##### Descriptor

Deprecated: Use ListUsersRequest.ProtoReflect.Descriptor instead.

##### GetFilters

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetSearchQuery

##### GetSortBy

##### GetSortDesc

##### GetTags

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUsersResponse

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

--- Legacy/Platform-specific ---

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

### RemoveRoleRequest

#### Methods

##### Descriptor

Deprecated: Use RemoveRoleRequest.ProtoReflect.Descriptor instead.

##### GetRole

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RemoveRoleResponse

#### Methods

##### Descriptor

Deprecated: Use RemoveRoleResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RevokeSessionRequest

#### Methods

##### Descriptor

Deprecated: Use RevokeSessionRequest.ProtoReflect.Descriptor instead.

##### GetSessionId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RevokeSessionResponse

#### Methods

##### Descriptor

Deprecated: Use RevokeSessionResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Session

#### Methods

##### Descriptor

Deprecated: Use Session.ProtoReflect.Descriptor instead.

##### GetAccessToken

##### GetCreatedAt

##### GetDeviceInfo

##### GetExpiresAt

##### GetId

##### GetIpAddress

##### GetMetadata

##### GetRefreshToken

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SyncSCIMRequest

#### Methods

##### Descriptor

Deprecated: Use SyncSCIMRequest.ProtoReflect.Descriptor instead.

##### GetScimPayload

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SyncSCIMResponse

#### Methods

##### Descriptor

Deprecated: Use SyncSCIMResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedUserServiceServer

UnimplementedUserServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### AssignRole

##### CreateReferral

##### CreateSession

##### CreateUser

##### DeleteUser

##### GetSession

##### GetUser

##### GetUserByEmail

##### GetUserByUsername

##### InitiateMFA

##### InitiateSSO

##### ListAuditLogs

##### ListPermissions

##### ListRoles

##### ListSessions

##### ListUserEvents

##### ListUsers

##### RegisterInterest

##### RemoveRole

##### RevokeSession

##### SyncSCIM

##### UpdatePassword

##### UpdateProfile

##### UpdateUser

### UnsafeUserServiceServer

UnsafeUserServiceServer may be embedded to opt out of forward compatibility for this service. Use of
this interface is not recommended, as added methods to UserServiceServer will result in compilation
errors.

### UpdatePasswordRequest

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

#### Methods

##### Descriptor

Deprecated: Use UpdateProfileResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateUserRequest

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

#### Methods

##### Descriptor

Deprecated: Use UpdateUserResponse.ProtoReflect.Descriptor instead.

##### GetUser

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### User

#### Methods

##### Descriptor

Deprecated: Use User.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetDeviceHash

##### GetEmail

##### GetExternalIds

##### GetId

##### GetLocation

##### GetMasterId

##### GetMetadata

##### GetPasswordHash

##### GetProfile

##### GetReferralCode

##### GetReferredBy

##### GetRoles

##### GetStatus

##### GetTags

##### GetUpdatedAt

##### GetUsername

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UserEvent

#### Methods

##### Descriptor

Deprecated: Use UserEvent.ProtoReflect.Descriptor instead.

##### GetDescription

##### GetEventType

##### GetId

##### GetMasterId

##### GetMetadata

##### GetOccurredAt

##### GetPayload

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UserProfile

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

UserService handles user management, authentication, RBAC, and audit operations

### UserServiceServer

UserServiceServer is the server API for UserService service. All implementations must embed
UnimplementedUserServiceServer for forward compatibility.

UserService handles user management, authentication, RBAC, and audit operations

### UserStatus

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
