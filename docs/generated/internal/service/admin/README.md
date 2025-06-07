# Package admin

## Constants

### AdminNamespace

AdminMetadataFields defines the canonical keys for admin metadata.

## Variables

### AdminEventRegistry

## Types

### EventEmitter

EventEmitter defines the interface for emitting events.

### EventHandlerFunc

### EventRegistry

### EventSubscription

### Repository

#### Methods

##### AssignRole

Role assignment.

##### CreateRole

Role management.

##### CreateUser

User management.

##### DeleteRole

##### DeleteUser

##### GetAuditLogs

Audit log management.

##### GetSettings

Settings management.

##### GetUser

##### ListRoles

##### ListUsers

##### RevokeRole

##### UpdateRole

##### UpdateSettings

##### UpdateUser

### Service

#### Methods

##### AssignRole

Role assignment.

##### CreateRole

Role management.

##### CreateUser

User management.

##### DeleteRole

##### DeleteUser

##### GetAuditLogs

Audit logs.

##### GetSettings

Settings.

##### GetUser

##### ListRoles

##### ListUsers

##### RevokeRole

##### UpdateRole

##### UpdateSettings

##### UpdateUser

## Functions

### NewService

### Register

Register registers the admin service with the DI container and event bus support.

### StartEventSubscribers
