# Package security

## Types

### Service

Service implements the SecurityServiceServer interface.

#### Methods

##### Authenticate

Authenticate verifies user identity and returns a session token.

##### Authorize

Authorize checks if a session token is allowed to perform an action on a resource.

##### DetectThreats

DetectThreats analyzes a request for potential threats.

##### GetAuditLog

GetAuditLog streams audit log entries.

##### GetSecurityMetrics

GetSecurityMetrics returns security metrics and incidents.

##### RecordAuditEvent

RecordAuditEvent logs a security-related event.

##### RegisterSecurityPattern

RegisterSecurityPattern registers a new security pattern.

##### ReportIncident

ReportIncident records a security incident.

##### ValidatePattern

ValidatePattern checks if a security pattern is valid.

##### ValidateToken

ValidateToken checks if a token is valid and returns its status.
