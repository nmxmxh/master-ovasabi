# Auth Service

The Auth service provides authentication, authorization, and session management for the OVASABI
platform.

## Architecture

```mermaid
graph TD
    A[gRPC API] -->|Implements| B[Service Layer]
    B -->|Uses| C[Repository Layer]
    C -->|Writes| D[Postgres]
    C -->|Caches| E[Redis]
    F[Master Repository] -->|References| D
    G[Event Log] -->|Records| D
    H[User Service] -->|Validates| B
    I[External Auth] -->|Integrates| B
```

## Features

1. **Authentication**

   - Password-based auth
   - OAuth/OIDC support
   - Multi-factor auth
   - Social login

2. **Authorization**

   - Token management
   - Permission checking
   - Scope validation
   - Policy enforcement

3. **Session Management**

   - Session tracking
   - Token refresh
   - Session invalidation
   - Concurrent session control

4. **Security Features**
   - Rate limiting
   - Brute force protection
   - IP blocking
   - Security events

## API Reference

### Proto Definition

```protobuf
service AuthService {
    rpc Login(LoginRequest) returns (AuthResponse);
    rpc Logout(LogoutRequest) returns (LogoutResponse);
    rpc RefreshToken(RefreshTokenRequest) returns (AuthResponse);
    rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
    rpc GenerateOTP(GenerateOTPRequest) returns (OTPResponse);
    rpc VerifyOTP(VerifyOTPRequest) returns (AuthResponse);
    rpc GetSecurityEvents(GetSecurityEventsRequest) returns (SecurityEventsResponse);
}
```

### Methods

#### Login

Authenticates a user and returns tokens.

```go
func (s *service) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error)
```

#### ValidateToken

Validates a token and returns its claims.

```go
func (s *service) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error)
```

## Data Model

### Token Model

```go
type TokenModel struct {
    ID           uuid.UUID
    UserID       uuid.UUID
    AccessToken  string
    RefreshToken string
    TokenType    string
    ExpiresAt    time.Time
    Scope        []string
    DeviceInfo   map[string]interface{} // JSONB in DB
    CreatedAt    time.Time
    RevokedAt    *time.Time
}
```

### Database Schema

```sql
CREATE TABLE auth_tokens (
    id UUID PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id),
    user_id UUID NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    token_type TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    scope TEXT[] DEFAULT '{}',
    device_info JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_auth_tokens_user_id ON auth_tokens(user_id);
CREATE INDEX idx_auth_tokens_access_token ON auth_tokens(access_token);
CREATE INDEX idx_auth_tokens_refresh_token ON auth_tokens(refresh_token);
```

## Knowledge Graph

### Capabilities

```go
type AuthCapabilities struct {
    Authentication struct {
        Password     bool `json:"password"`
        OAuth        bool `json:"oauth"`
        MFA          bool `json:"mfa"`
        Biometric    bool `json:"biometric"`
        Social       bool `json:"social"`
    } `json:"authentication"`

    Authorization struct {
        TokenBased   bool `json:"token_based"`
        RoleBased    bool `json:"role_based"`
        PolicyBased  bool `json:"policy_based"`
        Contextual   bool `json:"contextual"`
    } `json:"authorization"`

    SessionManagement struct {
        Tracking     bool `json:"tracking"`
        Refresh      bool `json:"refresh"`
        Concurrent   bool `json:"concurrent"`
        DeviceBound  bool `json:"device_bound"`
    } `json:"session_management"`

    Security struct {
        RateLimiting bool `json:"rate_limiting"`
        BruteForce   bool `json:"brute_force"`
        IPBlocking   bool `json:"ip_blocking"`
        Audit        bool `json:"audit"`
    } `json:"security"`
}
```

### Growth Patterns

```mermaid
graph TD
    A[Basic Auth] -->|Add OAuth| B[Social Auth]
    B -->|Add MFA| C[Secure Auth]
    C -->|Add Biometric| D[Advanced Auth]

    E[Token Auth] -->|Add Roles| F[Role Auth]
    F -->|Add Policies| G[Policy Auth]
    G -->|Add Context| H[Smart Auth]

    I[Basic Session] -->|Add Refresh| J[Extended Session]
    J -->|Add Device| K[Bound Session]
    K -->|Add Analytics| L[Smart Session]

    M[Basic Security] -->|Add Rate Limit| N[Protected]
    N -->|Add Geo| O[Geo Protected]
    O -->|Add ML| P[AI Protected]
```

### Evolution Tracking

```go
type AuthEvolution struct {
    AuthMethods       []string `json:"auth_methods"`
    SecurityFeatures  []string `json:"security_features"`
    TokenCapabilities []string `json:"token_capabilities"`
    PolicyFeatures    []string `json:"policy_features"`
    Version          string    `json:"version"`
    LastUpdated      time.Time `json:"last_updated"`
}
```

## Caching Strategy

### Key Structure

- Access Token: `cache:auth:access:{token}`
- Refresh Token: `cache:auth:refresh:{token}`
- User Sessions: `cache:auth:sessions:{user_id}`
- Rate Limits: `cache:auth:ratelimit:{ip}`

### TTL Values

- Access Token: 15 minutes
- Refresh Token: 7 days
- User Sessions: 24 hours
- Rate Limits: 1 hour

## Error Handling

### Error Types

1. **AuthenticationError**

   - Invalid credentials
   - Expired token
   - Invalid token

2. **AuthorizationError**

   - Insufficient permissions
   - Invalid scope
   - Policy violation

3. **SecurityError**
   - Rate limit exceeded
   - Suspicious activity
   - Blocked IP

## Monitoring

### Metrics

1. **Auth Metrics**

   - Login attempts
   - Success/failure rates
   - Token usage
   - MFA usage

2. **Security Metrics**
   - Rate limit hits
   - Blocked attempts
   - Suspicious activities
   - Policy violations

## Security

1. **Token Security**

   - JWT signing
   - Token encryption
   - Secure storage
   - Regular rotation

2. **Access Control**
   - Fine-grained permissions
   - Context-aware policies
   - Audit logging

## Future Improvements

1. **Phase 1 - Enhanced Auth**

   - Passwordless auth
   - Hardware security keys
   - Biometric integration

2. **Phase 2 - Smart Security**

   - ML-based threat detection
   - Behavioral analysis
   - Adaptive policies

3. **Phase 3 - Advanced Features**
   - Zero-trust architecture
   - Quantum-safe crypto
   - Cross-device auth

## Dependencies

- User Service (v1)
- Redis Cache
- Postgres Database
- External Auth Providers

## Configuration

```yaml
auth:
  database:
    pool_size: 20
    max_idle: 10
    max_lifetime: 1h
  cache:
    ttl:
      access_token: 15m
      refresh_token: 7d
      sessions: 24h
  security:
    rate_limit: 100/minute
    max_attempts: 5
    block_duration: 1h
  jwt:
    signing_method: RS256
    private_key_path: /etc/auth/private.pem
    public_key_path: /etc/auth/public.pem
```

## Deployment

### Resource Requirements

- CPU: 4 cores
- Memory: 8GB
- Storage: 20GB
- Cache: 4GB Redis

### Environment Variables

```bash
AUTH_DB_URL=postgres://user:pass@host:5432/db
AUTH_REDIS_URL=redis://host:6379
AUTH_LOG_LEVEL=info
AUTH_API_PORT=50053
AUTH_JWT_SECRET=your-secret-key
```
