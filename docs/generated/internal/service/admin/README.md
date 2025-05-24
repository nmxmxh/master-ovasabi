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

### GetAdminMetadata

Example helper: Extract admin metadata from common.Metadata.

### NewService

### Register

Register registers the admin service with the DI container and event bus support.

### SetAdminAudit

Example: Set audit info for admin metadata.

### SetAdminImpersonation

Example: Set impersonation info for admin metadata.

### SetAdminLastAction

Example: Set last action for admin metadata.

### SetAdminLastLoginAt

Example: Set last login timestamp for admin metadata.

### SetAdminMetadataField

Example helper: Set a field in admin metadata.

### SetAdminNotes

Example: Set admin notes for admin metadata.

### SetAdminPermissions

Example: Set permissions for admin metadata.

### SetAdminRBAC

Example: Set RBAC roles for admin metadata.

### SetAdminVersioning

Example: Set versioning info for admin metadata.

### StartEventSubscribers
