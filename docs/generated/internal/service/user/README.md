# Package user

## Variables

### ErrUserExists

### UserEventRegistry

## Types

### AccessibilityMetadata

### AuditMetadata

### BadActorMetadata

### BiometricProvider

BiometricProvider defines the interface for biometric authentication (e.g., Passage).

### ComplianceIssue

### ComplianceMetadata

### ComplianceStandard

### EmailProvider

EmailProvider defines the interface for sending emails (verification, password reset, etc.).

### EventEmitter

EventEmitter defines the interface for emitting events in the user service.

### EventHandlerFunc

### EventMetadata

### EventRegistry

### EventSubscription

### FrequencyMetadata

### LocalizationMetadata

LocalizationMetadata holds localization and compliance info for the user.

### LocationMetadata

### MFAChallengeData

--- ServiceMetadata struct and helpers --- Extend ServiceMetadata to support MFAChallenge for MFA
flows.

### Metadata

ServiceMetadata holds all user service-specific metadata fields.

### MockBiometricProvider

MockBiometricProvider is a mock implementation for testing.

#### Methods

##### IsBiometricEnabled

##### MarkBiometricUsed

### MockEmailProvider

MockEmailProvider is a mock implementation for testing.

#### Methods

##### SendPasswordResetEmail

##### SendVerificationEmail

### MockWebAuthnProvider

MockWebAuthnProvider is a mock implementation for testing.

#### Methods

##### BeginLogin

##### BeginRegistration

##### FinishLogin

##### FinishRegistration

### PasswordResetData

PasswordResetData holds password reset code and expiry.

### Profile

### Repository

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

### Service

Service implements the UserService gRPC interface.

#### Methods

##### AddFriend

--- Social Graph APIs ---.

##### AssignRole

AssignRole assigns a role to a user and updates metadata.

##### BeginWebAuthnLogin

BeginWebAuthnLogin emits an event instead of direct WebAuthn logic.

##### BeginWebAuthnRegistration

BeginWebAuthnRegistration emits an event instead of direct WebAuthn logic.

##### BlockGroupContent

##### BlockGroupIndividuals

##### BlockUser

--- Moderation/Interaction APIs ---.

##### CreateReferral

##### CreateSession

##### CreateUser

CreateUser creates a new user following the Master-Client-Service-Event pattern.

##### CreateUserGroup

--- Group APIs ---.

##### DeleteUser

DeleteUser removes a user and its master record.

##### DeleteUserGroup

##### FindOrCreateOAuthUser

FindOrCreateOAuthUser looks up a user by email/provider, creates if not found, updates OAuth and
audit metadata, and returns the user.

##### FinishWebAuthnLogin

FinishWebAuthnLogin emits an event instead of direct WebAuthn logic.

##### FinishWebAuthnRegistration

FinishWebAuthnRegistration emits an event instead of direct WebAuthn logic.

##### FollowUser

##### GetSession

##### GetUser

GetUser retrieves user information.

##### GetUserByEmail

GetUserByEmail retrieves user information by email.

##### GetUserByUsername

GetUserByUsername retrieves user information by username.

##### InitiateMFA

##### InitiateSSO

--- Add stubs for all unimplemented proto RPCs ---.

##### IsBiometricEnabled

IsBiometricEnabled emits an event instead of direct biometric check.

##### ListAuditLogs

ListAuditLogs lists audit logs for a user (stub).

##### ListConnections

##### ListFollowers

##### ListFollowing

##### ListFriends

##### ListPermissions

ListPermissions lists all permissions for a user.

##### ListRoles

ListRoles lists all roles for a user.

##### ListSessions

##### ListUserEvents

ListUserEvents lists user events (stub).

##### ListUserGroupMembers

##### ListUserGroups

##### ListUsers

ListUsers retrieves a list of users with pagination and filtering.

##### MarkBiometricUsed

MarkBiometricUsed emits an event instead of direct biometric usage.

##### MuteGroupContent

##### MuteGroupIndividuals

##### MuteUser

##### RegisterInterest

##### RemoveFriend

##### RemoveRole

RemoveRole removes a role from a user and updates metadata.

##### ReportGroupContent

##### ReportUser

##### RequestPasswordReset

RequestPasswordReset emits a notification event instead of direct call.

##### ResetPassword

ResetPassword resets the user's password after code verification.

##### RevokeSession

##### SendVerificationEmail

--- Composable Auth Channel Methods --- SendVerificationEmail emits a notification event instead of
direct call.

##### SuggestConnections

--- Social Graph Discovery ---.

##### SyncSCIM

##### UnblockGroupIndividuals

##### UnblockUser

##### UnfollowUser

##### UnmuteGroup

##### UnmuteGroupIndividuals

##### UnmuteUser

##### UpdatePassword

UpdatePassword implements the UpdatePassword RPC method.

##### UpdateProfile

UpdateProfile updates a user's profile.

##### UpdateUser

UpdateUser updates a user record.

##### UpdateUserGroup

##### VerifyEmail

VerifyEmail verifies the code and marks email as verified.

##### VerifyPasswordReset

VerifyPasswordReset verifies the password reset code.

### User

User represents a user in the service_user table.

### VerificationData

VerificationData holds email verification code and expiry.

### WebAuthnCredential

WebAuthnCredential holds a registered passkey credential.

### WebAuthnProvider

WebAuthnProvider defines the interface for WebAuthn operations (passkey registration/login).

## Functions

### MetadataToStruct

ServiceMetadataToStruct converts ServiceMetadata to structpb.Struct.

### NewService

NewUserService creates a new instance of UserService.

### Register

Register registers the user service with the DI container and event bus support.

### SetLogger

### StartEventSubscribers
