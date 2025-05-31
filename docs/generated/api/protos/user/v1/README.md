# Package userv1

## Constants

### UserService_CreateUser_FullMethodName

## Variables

### UserStatus_name

Enum value maps for UserStatus.

### FriendshipStatus_name

Enum value maps for FriendshipStatus.

### FollowStatus_name

Enum value maps for FollowStatus.

### File_user_v1_user_proto

### UserService_ServiceDesc

UserService_ServiceDesc is the grpc.ServiceDesc for UserService service. It's only intended for
direct use with grpc.RegisterService, and not to be introspected or modified (even as a copy)

## Types

### AddFriendRequest

Social Graph

#### Methods

##### Descriptor

Deprecated: Use AddFriendRequest.ProtoReflect.Descriptor instead.

##### GetFriendId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AddFriendResponse

#### Methods

##### Descriptor

Deprecated: Use AddFriendResponse.ProtoReflect.Descriptor instead.

##### GetFriendship

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### AssignRoleRequest

RBAC

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

##### GetMasterUuid

##### GetMetadata

##### GetOccurredAt

##### GetPayload

##### GetResource

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BlockGroupContentRequest

Group/content moderation

#### Methods

##### Descriptor

Deprecated: Use BlockGroupContentRequest.ProtoReflect.Descriptor instead.

##### GetContentId

##### GetGroupId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BlockGroupContentResponse

#### Methods

##### Descriptor

Deprecated: Use BlockGroupContentResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BlockGroupIndividualsRequest

Block all members of a group for a user (optionally, with a duration)

#### Methods

##### Descriptor

Deprecated: Use BlockGroupIndividualsRequest.ProtoReflect.Descriptor instead.

##### GetDurationMinutes

##### GetGroupId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BlockGroupIndividualsResponse

#### Methods

##### Descriptor

Deprecated: Use BlockGroupIndividualsResponse.ProtoReflect.Descriptor instead.

##### GetBlockedUserIds

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BlockUserRequest

Moderation/Interaction APIs

#### Methods

##### Descriptor

Deprecated: Use BlockUserRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetTargetUserId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### BlockUserResponse

#### Methods

##### Descriptor

Deprecated: Use BlockUserResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateReferralRequest

#### Methods

##### Descriptor

Deprecated: Use CreateReferralRequest.ProtoReflect.Descriptor instead.

##### GetCampaignSlug

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

Session

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

### CreateUserGroupRequest

Groups

#### Methods

##### Descriptor

Deprecated: Use CreateUserGroupRequest.ProtoReflect.Descriptor instead.

##### GetDescription

##### GetMemberIds

##### GetMetadata

##### GetName

##### GetRoles

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateUserGroupResponse

#### Methods

##### Descriptor

Deprecated: Use CreateUserGroupResponse.ProtoReflect.Descriptor instead.

##### GetUserGroup

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### CreateUserRequest

--- Requests/Responses --- User CRUD

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

### DeleteUserGroupRequest

#### Methods

##### Descriptor

Deprecated: Use DeleteUserGroupRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetUserGroupId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteUserGroupResponse

#### Methods

##### Descriptor

Deprecated: Use DeleteUserGroupResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### DeleteUserRequest

#### Methods

##### Descriptor

Deprecated: Use DeleteUserRequest.ProtoReflect.Descriptor instead.

##### GetHardDelete

##### GetMetadata

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

### Follow

#### Methods

##### Descriptor

Deprecated: Use Follow.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetFolloweeId

##### GetFollowerId

##### GetId

##### GetMetadata

##### GetStatus

##### GetUpdatedAt

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### FollowStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use FollowStatus.Descriptor instead.

##### Number

##### String

##### Type

### FollowUserRequest

#### Methods

##### Descriptor

Deprecated: Use FollowUserRequest.ProtoReflect.Descriptor instead.

##### GetFolloweeId

##### GetFollowerId

##### GetMetadata

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### FollowUserResponse

#### Methods

##### Descriptor

Deprecated: Use FollowUserResponse.ProtoReflect.Descriptor instead.

##### GetFollow

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### Friendship

#### Methods

##### Descriptor

Deprecated: Use Friendship.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetFriendId

##### GetId

##### GetMetadata

##### GetStatus

##### GetUpdatedAt

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### FriendshipStatus

#### Methods

##### Descriptor

##### Enum

##### EnumDescriptor

Deprecated: Use FriendshipStatus.Descriptor instead.

##### Number

##### String

##### Type

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

SSO/MFA/SCIM

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

##### GetUserId

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

### ListConnectionsRequest

#### Methods

##### Descriptor

Deprecated: Use ListConnectionsRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetType

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListConnectionsResponse

#### Methods

##### Descriptor

Deprecated: Use ListConnectionsResponse.ProtoReflect.Descriptor instead.

##### GetUsers

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListFollowersRequest

#### Methods

##### Descriptor

Deprecated: Use ListFollowersRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListFollowersResponse

#### Methods

##### Descriptor

Deprecated: Use ListFollowersResponse.ProtoReflect.Descriptor instead.

##### GetFollowers

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListFollowingRequest

#### Methods

##### Descriptor

Deprecated: Use ListFollowingRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListFollowingResponse

#### Methods

##### Descriptor

Deprecated: Use ListFollowingResponse.ProtoReflect.Descriptor instead.

##### GetFollowing

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListFriendsRequest

#### Methods

##### Descriptor

Deprecated: Use ListFriendsRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListFriendsResponse

#### Methods

##### Descriptor

Deprecated: Use ListFriendsResponse.ProtoReflect.Descriptor instead.

##### GetFriends

##### GetPage

##### GetTotalCount

##### GetTotalPages

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

Audit/Event

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

### ListUserGroupMembersRequest

#### Methods

##### Descriptor

Deprecated: Use ListUserGroupMembersRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetUserGroupId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUserGroupMembersResponse

#### Methods

##### Descriptor

Deprecated: Use ListUserGroupMembersResponse.ProtoReflect.Descriptor instead.

##### GetMembers

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUserGroupsRequest

#### Methods

##### Descriptor

Deprecated: Use ListUserGroupsRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetPage

##### GetPageSize

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ListUserGroupsResponse

#### Methods

##### Descriptor

Deprecated: Use ListUserGroupsResponse.ProtoReflect.Descriptor instead.

##### GetPage

##### GetTotalCount

##### GetTotalPages

##### GetUserGroups

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

### MuteGroupContentRequest

#### Methods

##### Descriptor

Deprecated: Use MuteGroupContentRequest.ProtoReflect.Descriptor instead.

##### GetContentId

##### GetDurationMinutes

##### GetGroupId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MuteGroupContentResponse

#### Methods

##### Descriptor

Deprecated: Use MuteGroupContentResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MuteGroupIndividualsRequest

Mute all members of a group for a user (optionally, with a duration)

#### Methods

##### Descriptor

Deprecated: Use MuteGroupIndividualsRequest.ProtoReflect.Descriptor instead.

##### GetDurationMinutes

##### GetGroupId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MuteGroupIndividualsResponse

#### Methods

##### Descriptor

Deprecated: Use MuteGroupIndividualsResponse.ProtoReflect.Descriptor instead.

##### GetMutedUserIds

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MuteUserRequest

#### Methods

##### Descriptor

Deprecated: Use MuteUserRequest.ProtoReflect.Descriptor instead.

##### GetDurationMinutes

##### GetMetadata

##### GetTargetUserId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### MuteUserResponse

#### Methods

##### Descriptor

Deprecated: Use MuteUserResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RefreshSessionRequest

#### Methods

##### Descriptor

Deprecated: Use RefreshSessionRequest.ProtoReflect.Descriptor instead.

##### GetRefreshToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RefreshSessionResponse

#### Methods

##### Descriptor

Deprecated: Use RefreshSessionResponse.ProtoReflect.Descriptor instead.

##### GetAccessToken

##### GetRefreshToken

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RegisterInterestRequest

#### Methods

##### Descriptor

Deprecated: Use RegisterInterestRequest.ProtoReflect.Descriptor instead.

##### GetEmail

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

### RemoveFriendRequest

#### Methods

##### Descriptor

Deprecated: Use RemoveFriendRequest.ProtoReflect.Descriptor instead.

##### GetFriendId

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### RemoveFriendResponse

#### Methods

##### Descriptor

Deprecated: Use RemoveFriendResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

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

### ReportGroupContentRequest

#### Methods

##### Descriptor

Deprecated: Use ReportGroupContentRequest.ProtoReflect.Descriptor instead.

##### GetContentId

##### GetDetails

##### GetGroupId

##### GetMetadata

##### GetReason

##### GetReporterUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ReportGroupContentResponse

#### Methods

##### Descriptor

Deprecated: Use ReportGroupContentResponse.ProtoReflect.Descriptor instead.

##### GetReportId

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ReportUserRequest

#### Methods

##### Descriptor

Deprecated: Use ReportUserRequest.ProtoReflect.Descriptor instead.

##### GetDetails

##### GetMetadata

##### GetReason

##### GetReportedUserId

##### GetReporterUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### ReportUserResponse

#### Methods

##### Descriptor

Deprecated: Use ReportUserResponse.ProtoReflect.Descriptor instead.

##### GetReportId

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

### SuggestConnectionsRequest

Social Graph Discovery

#### Methods

##### Descriptor

Deprecated: Use SuggestConnectionsRequest.ProtoReflect.Descriptor instead.

##### GetMetadata

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### SuggestConnectionsResponse

#### Methods

##### Descriptor

Deprecated: Use SuggestConnectionsResponse.ProtoReflect.Descriptor instead.

##### GetSuggestions

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

### UnblockGroupIndividualsRequest

#### Methods

##### Descriptor

Deprecated: Use UnblockGroupIndividualsRequest.ProtoReflect.Descriptor instead.

##### GetGroupId

##### GetTargetUserIds

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnblockGroupIndividualsResponse

#### Methods

##### Descriptor

Deprecated: Use UnblockGroupIndividualsResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### GetUnblockedUserIds

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnblockUserRequest

#### Methods

##### Descriptor

Deprecated: Use UnblockUserRequest.ProtoReflect.Descriptor instead.

##### GetTargetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnblockUserResponse

#### Methods

##### Descriptor

Deprecated: Use UnblockUserResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnfollowUserRequest

#### Methods

##### Descriptor

Deprecated: Use UnfollowUserRequest.ProtoReflect.Descriptor instead.

##### GetFolloweeId

##### GetFollowerId

##### GetMetadata

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnfollowUserResponse

#### Methods

##### Descriptor

Deprecated: Use UnfollowUserResponse.ProtoReflect.Descriptor instead.

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

##### AddFriend

##### AssignRole

##### BlockGroupContent

##### BlockGroupIndividuals

##### BlockUser

##### CreateReferral

##### CreateSession

##### CreateUser

##### CreateUserGroup

##### DeleteUser

##### DeleteUserGroup

##### FollowUser

##### GetSession

##### GetUser

##### GetUserByEmail

##### GetUserByUsername

##### InitiateMFA

##### InitiateSSO

##### ListAuditLogs

##### ListConnections

##### ListFollowers

##### ListFollowing

##### ListFriends

##### ListPermissions

##### ListRoles

##### ListSessions

##### ListUserEvents

##### ListUserGroupMembers

##### ListUserGroups

##### ListUsers

##### MuteGroupContent

##### MuteGroupIndividuals

##### MuteUser

##### RefreshSession

##### RegisterInterest

##### RemoveFriend

##### RemoveRole

##### ReportGroupContent

##### ReportUser

##### RevokeSession

##### SuggestConnections

##### SyncSCIM

##### UnblockGroupIndividuals

##### UnblockUser

##### UnfollowUser

##### UnmuteGroup

##### UnmuteGroupIndividuals

##### UnmuteUser

##### UpdatePassword

##### UpdateProfile

##### UpdateUser

##### UpdateUserGroup

### UnmuteGroupIndividualsRequest

#### Methods

##### Descriptor

Deprecated: Use UnmuteGroupIndividualsRequest.ProtoReflect.Descriptor instead.

##### GetGroupId

##### GetTargetUserIds

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnmuteGroupIndividualsResponse

#### Methods

##### Descriptor

Deprecated: Use UnmuteGroupIndividualsResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### GetUnmutedUserIds

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnmuteGroupRequest

#### Methods

##### Descriptor

Deprecated: Use UnmuteGroupRequest.ProtoReflect.Descriptor instead.

##### GetGroupId

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnmuteGroupResponse

#### Methods

##### Descriptor

Deprecated: Use UnmuteGroupResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnmuteUserRequest

#### Methods

##### Descriptor

Deprecated: Use UnmuteUserRequest.ProtoReflect.Descriptor instead.

##### GetTargetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnmuteUserResponse

#### Methods

##### Descriptor

Deprecated: Use UnmuteUserResponse.ProtoReflect.Descriptor instead.

##### GetSuccess

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UnsafeUserServiceServer

UnsafeUserServiceServer may be embedded to opt out of forward compatibility for this service. Use of
this interface is not recommended, as added methods to UserServiceServer will result in compilation
errors.

### UpdatePasswordRequest

Password/Profile

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

### UpdateUserGroupRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateUserGroupRequest.ProtoReflect.Descriptor instead.

##### GetFieldsToUpdate

##### GetMetadata

##### GetUserGroup

##### GetUserGroupId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateUserGroupResponse

#### Methods

##### Descriptor

Deprecated: Use UpdateUserGroupResponse.ProtoReflect.Descriptor instead.

##### GetUserGroup

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UpdateUserRequest

#### Methods

##### Descriptor

Deprecated: Use UpdateUserRequest.ProtoReflect.Descriptor instead.

##### GetFieldsToUpdate

##### GetMetadata

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

##### GetFollowerIds

##### GetFollowingIds

##### GetFriendIds

##### GetId

##### GetLocation

##### GetMasterId

##### GetMasterUuid

##### GetMetadata

##### GetPasswordHash

##### GetProfile

##### GetReferralCode

##### GetReferredBy

##### GetRoles

##### GetStatus

##### GetTags

##### GetUpdatedAt

##### GetUserGroupIds

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

##### GetMasterUuid

##### GetMetadata

##### GetOccurredAt

##### GetPayload

##### GetUserId

##### ProtoMessage

##### ProtoReflect

##### Reset

##### String

### UserGroup

--- Social Entities --- UserGroup: Identity/social group for membership, RBAC, and social graph.

#### Methods

##### Descriptor

Deprecated: Use UserGroup.ProtoReflect.Descriptor instead.

##### GetCreatedAt

##### GetDescription

##### GetId

##### GetMemberIds

##### GetMetadata

##### GetName

##### GetRoles

##### GetUpdatedAt

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

--- User Service: Full Social, Account, and Metadata API ---

### UserServiceServer

UserServiceServer is the server API for UserService service. All implementations must embed
UnimplementedUserServiceServer for forward compatibility.

--- User Service: Full Social, Account, and Metadata API ---

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
