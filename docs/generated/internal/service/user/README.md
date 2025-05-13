# Package user

## Variables

### ErrUserNotFound

## Types

### Service

Service implements the UserService gRPC interface.

#### Methods

##### AssignRole

##### CreateReferral

##### CreateSession

--- Add stubs for all unimplemented proto RPCs ---.

##### CreateUser

CreateUser creates a new user following the Master-Client-Service-Event pattern.

##### DeleteUser

DeleteUser removes a user and its master record.

##### GetSession

##### GetUser

GetUser retrieves user information.

##### GetUserByEmail

GetUserByEmail retrieves user information by email.

##### GetUserByUsername

GetUserByUsername retrieves user information by username.

##### InitiateMFA

##### InitiateSSO

##### ListAuditLogs

##### ListPermissions

##### ListRoles

##### ListSessions

##### ListUserEvents

##### ListUsers

ListUsers retrieves a list of users with pagination and filtering.

##### RegisterInterest

##### RemoveRole

##### RevokeSession

##### SyncSCIM

##### UpdatePassword

UpdatePassword implements the UpdatePassword RPC method.

##### UpdateProfile

UpdateProfile updates a user's profile.

##### UpdateUser

UpdateUser updates a user record.

## Functions

### NewUserService

NewUserService creates a new instance of UserService.
