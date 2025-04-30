# Package user

## Variables

### ErrUserNotFound

## Types

### Service

Service implements the UserService gRPC interface.

#### Methods

##### CreateUser

CreateUser creates a new user following the Master-Client-Service-Event pattern.

##### DeleteUser

DeleteUser removes a user and its master record.

##### GetUser

GetUser retrieves user information.

##### GetUserByUsername

GetUserByUsername retrieves user information by username.

##### ListUsers

ListUsers retrieves a list of users with pagination and filtering.

##### UpdatePassword

UpdatePassword implements the UpdatePassword RPC method.

##### UpdateProfile

UpdateProfile updates a user's profile.

##### UpdateUser

UpdateUser updates a user record.

## Functions

### NewUserService

NewUserService creates a new instance of UserService.
