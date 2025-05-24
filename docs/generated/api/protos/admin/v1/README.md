# Package adminpb

## Constants

### AdminService_CreateUser_FullMethodName

## Variables

### AdminService_ServiceDesc

AdminService_ServiceDesc is the grpc.ServiceDesc for AdminService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

### File_admin_v1_admin_proto

## Types

### AdminNote

#### Methods

##### Descriptor

Deprecated: Use AdminNote.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetCreatedBy

##### GetNote

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AdminPreferences

#### Methods

##### Descriptor

Deprecated: Use AdminPreferences.ProtoReflect.Descriptor instead.

##### GetNotificationsEnabled

##### GetTheme

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AdminServiceClient

AdminServiceClient is the client API for AdminService service.

For semantics around ctx use and closing/ending streaming RPCs, please refer to
https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.

--- Service ---

### AdminServiceServer

AdminServiceServer is the server API for AdminService service. All implementations must embed
UnimplementedAdminServiceServer for forward compatibility.

--- Service ---

### AssignRoleRequest

Role assignment

#### Methods

##### Descriptor

Deprecated: Use AssignRoleRequest.ProtoReflect.Descriptor instead.

##### GetRoleId

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

##### GetDetails

##### GetId

##### GetMasterId

##### GetMetadata

##### GetResource

##### GetTimestamp

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CheckPermissionRequest

Add CheckPermission messages

#### Methods

##### Descriptor

Deprecated: Use CheckPermissionRequest.ProtoReflect.Descriptor instead.

##### GetAction

##### GetContext

##### GetResource

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CheckPermissionResponse

#### Methods

##### Descriptor

Deprecated: Use CheckPermissionResponse.ProtoReflect.Descriptor instead.

##### GetAllowed

##### GetReason

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateRoleRequest

Role management

#### Methods

##### Descriptor

Deprecated: Use CreateRoleRequest.ProtoReflect.Descriptor instead.

##### GetRole

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateRoleResponse

#### Methods

##### Descriptor

Deprecated: Use CreateRoleResponse.ProtoReflect.Descriptor instead.

##### GetRole

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateUserRequest

User management

#### Methods

##### Descriptor

Deprecated: Use CreateUserRequest.ProtoReflect.Descriptor instead.

##### GetUser

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

### DeleteRoleRequest

#### Methods

##### Descriptor

Deprecated: Use DeleteRoleRequest.ProtoReflect.Descriptor instead.

##### GetRoleId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteRoleResponse

#### Methods

##### Descriptor

Deprecated: Use DeleteRoleResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteUserRequest

#### Methods

##### Descriptor

Deprecated: Use DeleteUserRequest.ProtoReflect.Descriptor instead.

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

### GetAuditLogsRequest

Audit logs

#### Methods

##### Descriptor

Deprecated: Use GetAuditLogsRequest.ProtoReflect.Descriptor instead.

##### GetAction

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetAuditLogsResponse

#### Methods

##### Descriptor

Deprecated: Use GetAuditLogsResponse.ProtoReflect.Descriptor instead.

##### GetLogs

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetSettingsRequest

Settings

#### Methods

##### Descriptor

Deprecated: Use GetSettingsRequest.ProtoReflect.Descriptor instead.

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### GetSettingsResponse

#### Methods

##### Descriptor

Deprecated: Use GetSettingsResponse.ProtoReflect.Descriptor instead.

##### GetSettings

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

### ImpersonationInfo

#### Methods

##### Descriptor

Deprecated: Use ImpersonationInfo.ProtoReflect.Descriptor instead.

##### GetActive

##### GetStartedAt

##### GetTargetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListRolesRequest

#### Methods

##### Descriptor

Deprecated: Use ListRolesRequest.ProtoReflect.Descriptor instead.

##### GetPage

##### GetPageSize

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListRolesResponse

#### Methods

##### Descriptor

Deprecated: Use ListRolesResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetRoles

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUsersRequest

#### Methods

##### Descriptor

Deprecated: Use ListUsersRequest.ProtoReflect.Descriptor instead.

##### GetPage

##### GetPageSize

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

### RevokeRoleRequest

#### Methods

##### Descriptor

Deprecated: Use RevokeRoleRequest.ProtoReflect.Descriptor instead.

##### GetRoleId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RevokeRoleResponse

#### Methods

##### Descriptor

Deprecated: Use RevokeRoleResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Role

#### Methods

##### Descriptor

Deprecated: Use Role.ProtoReflect.Descriptor instead.

##### GetId

##### GetMasterId

##### GetMetadata

##### GetName

##### GetPermissions

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Settings

#### Methods

##### Descriptor

Deprecated: Use Settings.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetValues

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnimplementedAdminServiceServer

UnimplementedAdminServiceServer must be embedded to have forward compatible implementations.

NOTE: this should be embedded by value instead of pointer to avoid a nil pointer dereference when
methods are called.

#### Methods

##### AssignRole

##### CheckPermission

##### CreateRole

##### CreateUser

##### DeleteRole

##### DeleteUser

##### GetAuditLogs

##### GetSettings

##### GetUser

##### ListRoles

##### ListUsers

##### RevokeRole

##### UpdateRole

##### UpdateSettings

##### UpdateUser

### UnsafeAdminServiceServer

UnsafeAdminServiceServer may be embedded to opt out of forward compatibility for this service. Use
of this interface is not recommended, as added methods to AdminServiceServer will result in
compilation errors.

### UpdateRoleRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateRoleRequest.ProtoReflect.Descriptor instead.

##### GetRole

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateRoleResponse

#### Methods

##### Descriptor

Deprecated: Use UpdateRoleResponse.ProtoReflect.Descriptor instead.

##### GetRole

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateSettingsRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateSettingsRequest.ProtoReflect.Descriptor instead.

##### GetSettings

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateSettingsResponse

#### Methods

##### Descriptor

Deprecated: Use UpdateSettingsResponse.ProtoReflect.Descriptor instead.

##### GetSettings

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateUserRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateUserRequest.ProtoReflect.Descriptor instead.

##### GetUser

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

##### GetEmail

##### GetId

##### GetIsActive

##### GetMasterId

##### GetMasterUuid

##### GetMetadata

##### GetName

##### GetRoles

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

## Functions

### RegisterAdminServiceServer
