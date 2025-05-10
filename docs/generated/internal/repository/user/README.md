# Package repository

## Variables

### ErrUserNotFound

## Types

### User

User represents a user in the service_user table.

### UserRepository

UserRepository handles operations on the service_user table.

#### Methods

##### Create

Create inserts a new user record.

##### Delete

Delete removes a user and its master record.

##### GetByEmail

GetByEmail retrieves a user by email.

##### GetByID

GetByID retrieves a user by ID.

##### GetByUsername

GetByUsername retrieves a user by username.

##### List

List retrieves a paginated list of users.

##### Update

Update updates a user record.

## Functions

### SetLogger
