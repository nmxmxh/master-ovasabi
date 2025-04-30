# Package service

Package service implements the business logic for gRPC services.

## Types

### AuthService

AuthService is an alias for the gRPC server interface.

### BroadcastService

BroadcastService is an alias for the gRPC server interface.

### Container

ServiceContainer defines the interface for accessing all service implementations.

### EchoService

EchoService implements the EchoService gRPC service. It provides a simple echo functionality that
returns the same message that was sent in the request.

#### Methods

##### Echo

Echo implements the Echo RPC method. It simply returns the message that was sent in the request.
Parameters:

- ctx: Context for the request
- req: The echo request containing the message to echo

Returns:

- \*protos.EchoResponse: Response containing the echoed message
- error: Any error that occurred during processing

### I18nService

I18nService is an alias for the gRPC server interface.

### NotificationService

NotificationService handles sending notifications.

### Provider

Provider manages service instances and their dependencies.

#### Methods

##### Asset

Asset returns the AssetService instance.

##### Auth

Auth returns the AuthService instance.

##### Broadcast

Broadcast returns the BroadcastService instance.

##### Close

Close closes all resources.

##### Finance

Finance returns the FinanceService instance.

##### I18n

I18n returns the I18nService instance.

##### Notification

Notification returns the NotificationService instance.

##### Quotes

Quotes returns the QuotesService instance.

##### Referrals

Referrals returns the ReferralService instance.

##### User

User returns the UserService instance.

### Quote

Quote represents a financial quote.

### ReferralStats

ReferralStats represents referral statistics.

### Registry

Registry defines the interface for service registration.

### User

User is an alias for the models.User type.

### UserService

UserService handles user management.
