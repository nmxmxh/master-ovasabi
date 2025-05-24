# Package repository

## Variables

### ErrUserNotFound

## Types

### User

User represents a user in the service_user table.

### UserProfile

### UserRepository

UserRepository handles operations on the service_user table.

#### Methods

##### AddFriend

--- Social Graph: Friends & Follows ---.

##### AssignRole

--- RBAC & Permissions ---.

##### BlockGroupContent

BlockGroupContent blocks a specific content item in a group for a user.

##### BlockGroupIndividuals

BlockGroupIndividuals blocks all members of a group for a user (optionally, with a duration).

##### Create

Create inserts a new user record.

##### CreateReferral

CreateReferral generates a referral code for a user and campaign.

##### CreateSession

--- Session Management ---.

##### CreateUserGroup

--- User Groups ---.

##### Delete

Delete removes a user and its master record.

##### DeleteUserGroup

##### FollowUser

##### GetByEmail

GetByEmail retrieves a user by email.

##### GetByID

GetByID retrieves a user by ID.

##### GetByUsername

GetByUsername retrieves a user by username.

##### GetSession

##### InitiateMFA

##### InitiateSSO

--- SSO, MFA, SCIM ---.

##### List

List retrieves a paginated list of users.

##### ListAuditLogs

ListAuditLogs fetches audit logs by user ID with pagination.

##### ListConnections

ListConnections lists connections of a given type (friend, follow, follower) for a user.

##### ListFlexible

ListFlexible retrieves a paginated, filtered list of users with flexible search.

##### ListFollowers

ListFollowers returns a paginated list of users who follow the given user.

##### ListFollowing

ListFollowing returns a paginated list of users whom the given user is following.

##### ListFriends

##### ListPermissions

##### ListRoles

##### ListSessions

##### ListUserEvents

ListUserEvents fetches user events by user ID with pagination.

##### ListUserGroupMembers

##### ListUserGroups

##### MuteGroupContent

MuteGroupContent mutes a specific content item in a group for a user.

##### MuteGroupIndividuals

MuteGroupIndividuals mutes all members of a group for a user (optionally, with a duration).

##### RegisterInterest

RegisterInterest creates or updates a pending user for interest registration.

##### RemoveFriend

##### RemoveRole

##### ReportGroupContent

ReportGroupContent reports a specific content item in a group.

##### RevokeSession

##### SuggestConnections

SuggestConnections suggests users with the most mutual friends (excluding the user).

##### SyncSCIM

##### UnblockGroupIndividuals

UnblockGroupIndividuals unblocks specific users in a group for a user.

##### UnblockUser

UnblockUser removes a block between the current user and the target user.

##### UnfollowUser

##### UnmuteGroup

UnmuteGroup unmutes all members of a group for a user.

##### UnmuteGroupIndividuals

UnmuteGroupIndividuals unmutes specific users in a group for a user.

##### UnmuteUser

UnmuteUser removes a mute between the current user and the target user.

##### Update

Update updates a user record.

##### UpdateUserGroup

## Functions

### SetLogger
