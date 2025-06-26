# Documentation

version: 2025-05-14

version: 2025-05-14

version: 2025-05-14

## Overview

The User Service is responsible for all user management, authentication, authorization, RBAC
(role-based access control), and audit logging in the platform. It is the canonical service for all
identity and access management, replacing the deprecated Auth service.

## Migration Note

- **Auth is deprecated.** All authentication and authorization logic is now handled by the User
  Service.
- Update all dependencies, integrations, and event logging to use the User Service.

## Responsibilities

- User CRUD and profile management
- Authentication (login, token, session)
- Authorization (role/permission checks)
- RBAC (role assignment, permission management)
- Audit logging (login attempts, permission changes, etc.)
- Metadata-driven event logging for all user/auth actions

## Step-by-Step Implementation Plan

1. **Migrate all Auth logic to User Service**
   - Grep for `Auth` and update all references to use User Service APIs.
   - Update DI containers and service registration.
2. **Implement Authentication**
   - Secure password hashing (bcrypt/argon2)
   - Session/token management (JWT, Redis, etc.)
   - Log all authentication events in `metadata` (see pattern below)
3. **Implement Authorization & RBAC**
   - Role and permission checks
   - RBAC assignment and management
   - Log all authorization and RBAC events in `metadata`
4. **Audit Logging**
   - Log all critical user/auth events (login, failed login, permission changes, etc.)
   - Store audit logs in a queryable, extensible format
5. **OpenAPI Documentation**
   - Document all endpoints using OpenAPI, following best practices
     ([see reference](https://medium.com/itnext/practical-openapi-in-go-1e9e6c4ed439))
6. **Security Best Practices**
   - Use middleware for authentication/authorization
   - Principle of least privilege, defense-in-depth
   - Regularly review and test for privilege escalation
     ([see reference](https://medium.com/faun/from-dev-to-admin-an-easy-kubernetes-privilege-escalation-you-should-be-aware-of-the-attack-950e6cf76cac))

## Metadata Patterns

### Authentication Event

```json
{
  "user": {
    "user_id": "user_456",
    "session_id": "sess_789",
    "auth_method": "password",
    "ip_address": "203.0.113.42",
    "user_agent": "Mozilla/5.0",
    "timestamp": "2024-05-15T12:00:00Z",
    "success": true,
    "failure_reason": null
  }
}
```

### Authorization Event

```json
{
  "auth": {
    "user_id": "user_456",
    "resource": "campaign:123",
    "action": "edit",
    "role": "admin",
    "granted": true,
    "timestamp": "2024-05-15T12:01:00Z"
  }
}
```

### RBAC Assignment

```json
{
  "user": {
    "user_id": "user_456",
    "roles": ["admin", "editor"],
    "permissions": ["campaign:create", "campaign:edit", "user:invite"],
    "assigned_by": "user_001",
    "timestamp": "2024-05-15T12:05:00Z"
  }
}
```

## Content Moderation Metadata (for user-generated content)

```json
{
  "content_moderation": {
    "age_rating": "18+",
    "obscenity_score": 0.92,
    "mature_content": true,
    "provocative": true,
    "racist": false,
    "bad_actor": true,
    "bad_actor_count": 4,
    "last_flagged_at": "2024-05-15T12:10:00Z",
    "moderation_notes": "Repeated violations for hate speech"
  }
}
```

## References

- [Practical OpenAPI in Go (ITNEXT)](https://medium.com/itnext/practical-openapi-in-go-1e9e6c4ed439)
- [Kubernetes Privilege Escalation](https://medium.com/faun/from-dev-to-admin-an-easy-kubernetes-privilege-escalation-you-should-be-aware-of-the-attack-950e6cf76cac)
- [Hashnode: Auth in Go](https://tanmay-vaish.hashnode.dev/how-to-implement-authentication-and-authorization-in-golang)
- [Secure Auth API in Go](https://medium.com/@fasgolangdev/how-to-create-a-secure-authentication-api-in-golang-using-middlewares-6988632ddfd3)
- [Golang Auth Series (Stackademic)](https://blog.stackademic.com/golang-series-e63a91eb386b)
- [RBAC in Go (Permit.io)](https://www.permit.io/blog/role-based-access-control-rbac-authorization-in-golang)

## Step 1: Migrate All Auth Logic to User Service

### Checklist

- [ ] Grep for `Auth` in the codebase:
  - `grep -iR 'Auth' .`
- [ ] For each reference:
  - [ ] If it is a dependency, update to use the User Service.
  - [ ] If it is a function call (login, token, session, RBAC), use the User Service API.
  - [ ] If it is event logging, store the relevant info in the `metadata` field under the `user` or
        `auth` namespace (see patterns above).
- [ ] Update DI containers and service registration to remove Auth and use User Service.
- [ ] Update configuration files to remove Auth references.
- [ ] Test all integrations to ensure the User Service is functioning as expected.

### Migration Progress

- Track files and modules as they are updated here.
- Note any issues or special cases encountered during migration.

### Reminder

- After migration, all authentication and authorization logic, event logging, and RBAC should be
  handled exclusively by the User Service.
- All essential information should be stored in the extensible `metadata` field for future-proofing
  and discoverability.

## Step 2: Implement Authentication

### Implementation Tasks

- [ ] Implement secure password hashing (bcrypt or argon2 recommended)
- [ ] Implement session and token management (JWT, Redis, or similar)
- [ ] Design and document authentication endpoints (login, logout, refresh, etc.) using OpenAPI
- [ ] Log all authentication events in the `metadata` field (see example below)
- [ ] Ensure all authentication logic is covered by tests

### Example Authentication Event Metadata

```json
{
  "user": {
    "user_id": "user_456",
    "session_id": "sess_789",
    "auth_method": "password",
    "ip_address": "203.0.113.42",
    "user_agent": "Mozilla/5.0",
    "timestamp": "2024-05-15T12:00:00Z",
    "success": true,
    "failure_reason": null
  }
}
```

### OpenAPI Documentation

- Define all authentication endpoints in your OpenAPI spec (see
  [Practical OpenAPI in Go](https://medium.com/itnext/practical-openapi-in-go-1e9e6c4ed439))
- Include request/response schemas, error types, and security schemes

### Checklist

- [ ] Passwords are hashed securely (never stored in plaintext)
- [ ] Sessions/tokens are securely generated, stored, and validated
- [ ] All authentication events are logged in metadata
- [ ] Endpoints are documented in OpenAPI
- [ ] All logic is covered by unit and integration tests

### Progress & Notes

- Track implementation progress and any issues here.
- Note any design decisions or deviations from the plan.

## Step 2a: Authentication Middleware/Interceptor Patterns

To ensure secure, consistent authentication across all protocols, use middleware/interceptors for:

- gRPC (Unary/Stream Interceptors)
- WebSocket (on connect or first message)
- REST (middleware)

### gRPC Authentication Interceptor (Go Example)

```go
import (
    "context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
    "github.com/golang-jwt/jwt/v5"
)

func AuthInterceptor(secretKey string) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Error(codes.Unauthenticated, "missing metadata")
        }
        tokens := md["authorization"]
        if len(tokens) == 0 {
            return nil, status.Error(codes.Unauthenticated, "missing token")
        }
        tokenStr := tokens[0]
        token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
            return []byte(secretKey), nil
        })
        if err != nil || !token.Valid {
            return nil, status.Error(codes.Unauthenticated, "invalid token")
        }
        // Optionally, add user info to context here
        return handler(ctx, req)
    }
}
```

Register the interceptor:

```go
grpcServer := grpc.NewServer(
    grpc.UnaryInterceptor(AuthInterceptor(secretKey)),
)
```

### WebSocket Authentication (Go Example)

```go
func wsHandler(w http.ResponseWriter, r *http.Request) {
    tokenStr := r.Header.Get("Authorization")
    // Validate JWT as above
    // If valid, upgrade connection
    conn, err := upgrader.Upgrade(w, r, nil)
    // Store user info in connection context
}
```

Or require an auth message as the first frame after connect.

### REST Authentication (Echo Example)

```go
e.Use(middleware.JWTWithConfig(middleware.JWTConfig{
    SigningKey: []byte(secretKey),
}))
```

### Centralizing Token Validation & Metadata Logging

- Use a shared package/service for token validation logic across all protocols.
- Use the same JWT/session format and secret for all protocols.
- Log all authentication events in metadata for traceability.
- Customize error handling and context injection as needed for each protocol.

### Best Practices

- gRPC: Use interceptors for all unary and streaming calls.
- WebSocket: Authenticate on connect or first message, store user/session in context.
- REST: Use middleware for all protected routes.
- Metadata: Always log authentication attempts, successes, and failures in your metadata system.

### Summary Table

| Protocol  | Middleware/Interceptor   | Where to Check Token        | How to Log Event   |
| --------- | ------------------------ | --------------------------- | ------------------ |
| gRPC      | Unary/Stream Interceptor | Metadata (headers)          | Metadata/audit log |
| WebSocket | On connect/first msg     | HTTP header or first frame  | Metadata/audit log |
| REST      | Middleware               | HTTP header (Authorization) | Metadata/audit log |

### References

- [Secure Auth API in Go (Medium)](https://medium.com/@fasgolangdev/how-to-create-a-secure-authentication-api-in-golang-using-middlewares-6988632ddfd3)
- [gRPC Interceptors (Go)](https://grpc.io/docs/guides/auth/)
- [Golang JWT Middleware (Echo)](https://echo.labstack.com/docs/middleware/jwt)
- [WebSocket Auth Patterns (OWASP)](https://cheatsheetseries.owasp.org/cheatsheets/WebSocket_Security_Cheat_Sheet.html)

## Step 3: Implement Authorization & RBAC

### Implementation Tasks

- [ ] Implement role and permission checks for all protected resources
- [ ] Implement RBAC assignment and management endpoints (assign/remove roles, manage permissions)
- [ ] Log all authorization and RBAC events in the `metadata` field (see examples below)
- [ ] Document all authorization and RBAC endpoints in OpenAPI
- [ ] Ensure all authorization logic is covered by tests

### Example Authorization Event Metadata

```json
{
  "auth": {
    "user_id": "user_456",
    "resource": "campaign:123",
    "action": "edit",
    "role": "admin",
    "granted": true,
    "timestamp": "2024-05-15T12:01:00Z"
  }
}
```

### Example RBAC Assignment Metadata

```json
{
  "user": {
    "user_id": "user_456",
    "roles": ["admin", "editor"],
    "permissions": ["campaign:create", "campaign:edit", "user:invite"],
    "assigned_by": "user_001",
    "timestamp": "2024-05-15T12:05:00Z"
  }
}
```

### OpenAPI Documentation

- Define all authorization and RBAC endpoints in your OpenAPI spec (see
  [Practical OpenAPI in Go](https://medium.com/itnext/practical-openapi-in-go-1e9e6c4ed439))
- Include request/response schemas, error types, and security schemes

### Checklist

- [ ] Role and permission checks are enforced for all protected resources
- [ ] RBAC assignment and management endpoints are implemented and documented
- [ ] All authorization and RBAC events are logged in metadata
- [ ] Endpoints are documented in OpenAPI
- [ ] All logic is covered by unit and integration tests

### Progress & Notes

- Track implementation progress and any issues here.
- Note any design decisions or deviations from the plan.

## Step 4: Audit Logging

### Implementation Tasks

- [ ] Define the minimal audit log struct (see below)
- [ ] Add audit log entries to the metadata field for all critical user/auth events
- [ ] Ensure audit logs are queryable and retained per compliance requirements
- [ ] Document audit logging in OpenAPI and service documentation
- [ ] Ensure all audit logging logic is covered by tests

### Minimal Audit Log Struct (Go Example)

```go
// Minimal, standards-compliant audit log entry
// (GDPR, ISO 27001, NIS2)
type AuditLog struct {
    EventID    string                 `json:"event_id"`
    Timestamp  time.Time              `json:"timestamp"`
    ActorID    string                 `json:"actor_id"`
    ActorType  string                 `json:"actor_type"` // "user", "service", "system"
    Action     string                 `json:"action"`
    Resource   string                 `json:"resource"`
    Result     string                 `json:"result"`     // "success", "failure", "denied"
    IPAddress  string                 `json:"ip_address,omitempty"`
    UserAgent  string                 `json:"user_agent,omitempty"`
    Details    map[string]interface{} `json:"details,omitempty"`
    GDPRData   bool                   `json:"gdpr_data"`
    Location   string                 `json:"location,omitempty"`
}
```

### Example Audit Log Entry in Metadata

```json
{
  "audit_log": {
    "event_id": "01HY7ZK8Q9J8V7Q2K3F4B5N6M7",
    "timestamp": "2024-05-15T12:34:56Z",
    "actor_id": "user_456",
    "actor_type": "user",
    "action": "update_profile",
    "resource": "user:456",
    "result": "success",
    "ip_address": "203.0.113.42",
    "user_agent": "Mozilla/5.0",
    "details": {
      "fields_changed": ["email", "display_name"],
      "old_values": { "email": "old@example.com" },
      "new_values": { "email": "new@example.com" }
    },
    "gdpr_data": true,
    "location": "EU"
  }
}
```

### Best Practices for Compliance

- Log all critical user/auth events (login, failed login, permission changes, data access, etc.)
- Use unique event IDs and precise timestamps
- Flag events involving personal data (`gdpr_data: true`)
- Retain logs per your data retention policy (GDPR: only as long as necessary)
- Make logs queryable for incident response and compliance audits
- Store logs in a secure, tamper-evident system

### Checklist

- [ ] Audit log struct is defined and used for all critical events
- [ ] Audit log entries are added to metadata
- [ ] Logs are queryable and retained per compliance
- [ ] Audit logging is documented in OpenAPI/service docs
- [ ] All logic is covered by unit and integration tests

### Progress & Notes

- Track implementation progress and any issues here.
- Note any design decisions or deviations from the plan.

### References

- [ISO/IEC 27001:2022, A.12.4 Logging and monitoring](https://www.iso.org/isoiec-27001-information-security.html)
- [NIS2 Directive (EU) 2022/2555, Article 21](https://eur-lex.europa.eu/eli/dir/2022/2555)
- [GDPR Article 30 (Records of processing activities)](https://gdpr-info.eu/art-30-gdpr/)
- [GDPR Article 33 (Notification of a personal data breach)](https://gdpr-info.eu/art-33-gdpr/)
- [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)
- [ENISA Guidelines on Security Measures](https://www.enisa.europa.eu/publications/guidelines-on-security-measures-under-the-european-electronic-communications-code)

## Step 5: OpenAPI Documentation (Composable POST Requests)

### Implementation Tasks

- [ ] Document all User Service endpoints (authentication, RBAC, audit logging, user CRUD, etc.) in
      OpenAPI (YAML or JSON) using composable POST requests
- [ ] Define flexible, composable request and response schemas for each endpoint
- [ ] Specify error types and standard error responses
- [ ] Define security schemes (JWT, OAuth2, etc.)
- [ ] Provide examples for composable requests and responses
- [ ] Ensure OpenAPI docs are versioned and published for consumers

### Key Elements to Document

- Endpoints: `/login`, `/logout`, `/refresh`, `/users`, `/roles`, `/permissions`, `/audit-logs`,
  etc.
- Request/response schemas: Use OpenAPI `components.schemas` for reusability and composability
- Composable POST requests: Use `oneOf`, `anyOf`, or a flexible `metadata` field to allow extensible
  payloads
- Error types: Standardize error responses (e.g., 401 Unauthorized, 403 Forbidden, 422 Validation
  Error)
- Security: Use `securitySchemes` for JWT, OAuth2, etc.
- Examples: Provide example payloads for all major endpoints

### Example: Composable POST Request Schema (OpenAPI YAML)

```yaml
components:
  schemas:
    LoginRequest:
      type: object
      properties:
        email:
          type: string
        password:
          type: string
        metadata:
          type: object
          additionalProperties: true
      required:
        - email
        - password
    CreateUserRequest:
      type: object
      properties:
        username:
          type: string
        password:
          type: string
        profile:
          $ref: '#/components/schemas/UserProfile'
        metadata:
          type: object
          additionalProperties: true
      required:
        - username
        - password
    UserProfile:
      type: object
      properties:
        display_name:
          type: string
        avatar_url:
          type: string
        # ...
```

- For more advanced composability, use `oneOf` or `anyOf` to allow multiple request types or
  extensions.

### Best Practices

- Use composable POST requests with flexible schemas (`metadata`, `oneOf`, `anyOf`) for
  extensibility
- Use a consistent naming and path convention (see
  [Practical OpenAPI in Go](https://medium.com/itnext/practical-openapi-in-go-1e9e6c4ed439))
- Use `allOf`, `oneOf`, and `anyOf` for schema composition
- Version your OpenAPI spec and publish it for consumers (e.g., Swagger UI, Postman)
- Keep docs in sync with implementation (CI/CD automation recommended)

### Checklist

- [ ] All endpoints use composable POST requests where appropriate
- [ ] Schemas, errors, and security are clearly defined
- [ ] Examples are provided for all major endpoints
- [ ] OpenAPI docs are published and versioned
- [ ] Documentation is reviewed and kept up to date

### Benefits

- Extensible and future-proof API design
- Rapid onboarding of new features and request types
- Easier integration for clients and internal teams

### Progress & Notes

- Track documentation progress and any issues here.
- Note any design decisions or deviations from the plan.

### References

- [Practical OpenAPI in Go (ITNEXT)](https://medium.com/itnext/practical-openapi-in-go-1e9e6c4ed439)

## New Authentication Channels (2024)

The user service now supports the following authentication channels, following the composable
request pattern and robust metadata standard:

### 1. Email Verification & Password Reset

- **Flow:** On signup, a verification code is sent to the user's email. For password reset, a code
  is sent and must be verified before allowing password change.
- **Metadata Fields:**
  - `metadata.service_specific.user.email_verified: bool`
  - `metadata.service_specific.user.verification_data: { code, expires_at }`
  - `metadata.service_specific.user.password_reset: { code, expires_at }`
- **Endpoints:**
  - `POST /user/send_verification_email`
  - `POST /user/verify_email`
  - `POST /user/request_password_reset`
  - `POST /user/verify_password_reset`
  - `POST /user/reset_password`
- **OpenAPI Example:**

```yaml
components:
  schemas:
    SendVerificationEmailRequest:
      type: object
      properties:
        email:
          type: string
        metadata:
          type: object
          properties:
            service_specific:
              type: object
              properties:
                user:
                  type: object
                  additionalProperties: true
      required:
        - email
```

- **Reference:**
  [Email Verification and Password Reset Flow using golang](http://dvignesh1496.medium.com/email-verification-and-password-reset-flow-using-golang-c8bd037101e8)

### 2. Passkey/WebAuthn (Passwordless)

- **Flow:** Register and authenticate users using passkeys (WebAuthn credentials, e.g., biometrics
  or device PIN).
- **Metadata Fields:**
  - `metadata.service_specific.user.passkeys: [ { credential_id, public_key, transports, created_at } ]`
  - `metadata.service_specific.user.last_webauthn_login: timestamp`
- **Endpoints:**
  - `POST /user/webauthn/begin_registration`
  - `POST /user/webauthn/finish_registration`
  - `POST /user/webauthn/begin_login`
  - `POST /user/webauthn/finish_login`
- **OpenAPI Example:**

```yaml
components:
  schemas:
    WebAuthnBeginRegistrationRequest:
      type: object
      properties:
        username:
          type: string
        metadata:
          type: object
          properties:
            service_specific:
              type: object
              properties:
                user:
                  type: object
                  additionalProperties: true
      required:
        - username
```

- **Reference:** [PassKey in Go](https://dev.to/egregors/passkey-in-go-1efk)

### 3. Biometric Authentication

- **Flow:** Use device biometrics (TouchID, FaceID, Windows Hello) via WebAuthn or Passage.
- **Metadata Fields:**
  - `metadata.service_specific.user.biometric_enabled: bool`
  - `metadata.service_specific.user.biometric_last_used: timestamp`
- **Endpoints:**
  - Same as WebAuthn endpoints, or integrate with Passage SDK.
- **Reference:**
  [Build a Go app with biometric authentication](https://passage.1password.com/post/build-a-go-app-with-biometric-authentication)

### Composable Request Pattern

All new endpoints follow the composable request pattern:

- Top-level request object: `{Service}{Action}Request`
- Always include a `metadata` field with `service_specific.user` for extensibility.
- See
  [Composable Request Pattern Standard](../amadeus/amadeus_context.md#composable-request-pattern-standard)

### Checklist (Updated)

- [x] All endpoints use composable POST requests where appropriate
- [x] Schemas, errors, and security are clearly defined
- [x] Examples are provided for all major endpoints, including new authentication flows
- [x] OpenAPI docs are published and versioned
- [x] Documentation is reviewed and kept up to date

### References

- [Email Verification and Password Reset Flow using golang](http://dvignesh1496.medium.com/email-verification-and-password-reset-flow-using-golang-c8bd037101e8)
- [PassKey in Go](https://dev.to/egregors/passkey-in-go-1efk)
- [Build a Go app with biometric authentication](https://passage.1password.com/post/build-a-go-app-with-biometric-authentication)
