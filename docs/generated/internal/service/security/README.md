# Package security

## Variables

### SecurityEventRegistry

## Types

### AuditEntry

### BadActorMetadata

### ComplianceIssue

### ComplianceMetadata

### ComplianceStandard

### Event

### EventHandlerFunc

### EventMetadata

### EventRegistry

### EventSubscription

### FrequencyMetadata

### Identity

### Incident

### LocationMetadata

### Master

### Pattern

### Repository

#### Methods

##### CreateIdentity

Identity.

##### CreateMaster

Master.

##### GetEvents

##### GetIdentity

##### GetIncident

##### GetMaster

##### GetPattern

##### GetRiskAssessment

##### GetSecurityMetrics

Analytics.

##### ListIncidents

##### ListPatterns

##### RecordEvent

Event.

##### RecordIncident

Incident.

##### RegisterPattern

Pattern.

##### UpdateIdentityRiskScore

##### UpdateIncidentResolution

##### UpdateMaster

### RepositoryItf

### Service

Service implements the SecurityServiceServer interface with rich metadata handling and repository
integration.

#### Methods

##### AuditEvent

AuditEvent logs a security-related event.

##### Authenticate

Authenticate verifies user identity and returns a session token.

##### Authorize

Authorize checks if a session token is allowed to perform an action on a resource.

##### DetectThreats

DetectThreats analyzes a request for potential threats.

##### QueryEvents

QueryEvents streams security events (audit log entries).

##### ValidateCredential

ValidateCredential checks if a credential is valid and returns its status.

### ServiceMetadata

ServiceMetadata holds all security service-specific metadata fields.

## Functions

### Register

Register registers the security service with the DI container and event bus support.

### ServiceMetadataToStruct

ServiceMetadataToStruct converts ServiceMetadata to structpb.Struct.

### StartEventSubscribers
