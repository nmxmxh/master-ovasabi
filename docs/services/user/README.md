# User Service

The User service is a core service that manages user profiles, authentication integration, and
role-based access control in the OVASABI platform.

## Architecture

```mermaid
graph TD
    A[gRPC API] -->|Implements| B[Service Layer]
    B -->|Uses| C[Repository Layer]
    C -->|Writes| D[Postgres]
    C -->|Caches| E[Redis]
    F[Master Repository] -->|References| D
    G[Event Log] -->|Records| D
    H[Auth Service] -->|Validates| B
```

## Features

1. **User Management**

   - User registration
   - Profile updates
   - Account deactivation
   - User search and filtering

2. **Profile Handling**

   - Basic information
   - Extended attributes
   - Profile pictures
   - User preferences

3. **Role Management**

   - Role assignment
   - Permission management
   - Access level control
   - Role hierarchies

4. **Integration Points**
   - Auth service integration
   - Profile verification
   - Social connections
   - Activity tracking

## API Reference

### Proto Definition

```protobuf
service UserService {
    rpc CreateUser(CreateUserRequest) returns (UserResponse);
    rpc GetUser(GetUserRequest) returns (UserResponse);
    rpc UpdateUser(UpdateUserRequest) returns (UserResponse);
    rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
    rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
    rpc AssignRole(AssignRoleRequest) returns (UserResponse);
    rpc GetUserRoles(GetUserRolesRequest) returns (GetUserRolesResponse);
    rpc SearchUsers(SearchUsersRequest) returns (SearchUsersResponse);
}
```

### Methods

#### CreateUser

Creates a new user profile.

```go
func (s *service) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.UserResponse, error)
```

#### GetUser

Retrieves a user's profile.

```go
func (s *service) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.UserResponse, error)
```

## Data Model

### User Model

```go
type UserModel struct {
    ID            uuid.UUID
    Email         string
    Username      string
    FullName      string
    ProfilePicURL string
    Metadata      map[string]interface{} // JSONB in DB
    Roles         []string
    Status        string // active, inactive, suspended
    CreatedAt     time.Time
    UpdatedAt     time.Time
    LastLoginAt   *time.Time
}
```

### Database Schema

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id),
    email TEXT UNIQUE NOT NULL,
    username TEXT UNIQUE NOT NULL,
    full_name TEXT,
    profile_pic_url TEXT,
    metadata JSONB DEFAULT '{}',
    roles TEXT[] DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_roles ON users USING gin(roles);
CREATE INDEX idx_users_metadata ON users USING gin(metadata);
```

## Caching Strategy

### Key Structure

- User Profile: `cache:user:profile:{user_id}`
- User Roles: `cache:user:roles:{user_id}`
- Username Lookup: `cache:user:username:{username}`
- Email Lookup: `cache:user:email:{email}`

### TTL Values

- Profile: 1 hour
- Roles: 30 minutes
- Lookups: 24 hours

## Knowledge Graph

### Capabilities

```go
type UserCapabilities struct {
    ProfileManagement struct {
        BasicInfo    bool `json:"basic_info"`
        ExtendedInfo bool `json:"extended_info"`
        Preferences  bool `json:"preferences"`
        Pictures     bool `json:"pictures"`
    } `json:"profile_management"`

    RoleManagement struct {
        Assignment bool `json:"assignment"`
        Hierarchy  bool `json:"hierarchy"`
        Custom     bool `json:"custom"`
    } `json:"role_management"`

    Integration struct {
        Auth        bool `json:"auth"`
        Social      bool `json:"social"`
        Verification bool `json:"verification"`
    } `json:"integration"`

    Analytics struct {
        UserMetrics bool `json:"user_metrics"`
        Behavior    bool `json:"behavior"`
        Growth      bool `json:"growth"`
    } `json:"analytics"`
}
```

### Growth Patterns

```mermaid
graph TD
    A[Basic Profile] -->|Add Preferences| B[Enhanced Profile]
    B -->|Add Social| C[Connected Profile]
    C -->|Add Analytics| D[Smart Profile]

    E[Basic Roles] -->|Add Hierarchy| F[Role Trees]
    F -->|Add Custom| G[Dynamic Roles]
    G -->|Add ML| H[Adaptive Roles]

    I[Local Auth] -->|Add Social| J[Multi Auth]
    J -->|Add MFA| K[Secure Auth]
    K -->|Add Biometric| L[Advanced Auth]
```

### Evolution Tracking

```go
type UserEvolution struct {
    ProfileCapabilities []string `json:"profile_capabilities"`
    RoleCapabilities   []string `json:"role_capabilities"`
    IntegrationPoints  []string `json:"integration_points"`
    AnalyticsFeatures  []string `json:"analytics_features"`
    Version           string    `json:"version"`
    LastUpdated       time.Time `json:"last_updated"`
}
```

## Error Handling

### Error Types

1. **ValidationError**

   - Invalid email format
   - Invalid username format
   - Missing required fields

2. **ConflictError**

   - Duplicate email
   - Duplicate username
   - Role conflicts

3. **NotFoundError**
   - User not found
   - Role not found
   - Profile not found

## Monitoring

### Metrics

1. **User Metrics**

   - Active users
   - Registration rate
   - Profile completion
   - Role distribution

2. **Performance Metrics**
   - Query latency
   - Cache hit ratio
   - Error rates
   - API usage

## Security

1. **Data Protection**

   - Email encryption
   - Profile data encryption
   - Secure attribute storage

2. **Access Control**
   - Role-based access
   - Permission validation
   - Action auditing

## Future Improvements

1. **Phase 1 - Enhanced Profiles**

   - Custom attributes
   - Profile templates
   - Validation rules

2. **Phase 2 - Advanced Roles**

   - Dynamic permissions
   - Role inheritance
   - Context-aware roles

3. **Phase 3 - Smart Features**

   - Behavior analysis
   - Recommendation engine
   - Automated role optimization

4. **Phase 4 - Integration**
   - OAuth provider
   - SSO integration
   - External identity providers

## Dependencies

- Auth Service (v1)
- Redis Cache
- Postgres Database
- Object Storage (for profile pictures)

## Configuration

```yaml
user:
  database:
    pool_size: 20
    max_idle: 10
    max_lifetime: 1h
  cache:
    ttl:
      profile: 1h
      roles: 30m
      lookup: 24h
  limits:
    max_roles_per_user: 10
    max_profile_pic_size: 5MB
    username_min_length: 3
    username_max_length: 32
```

## Deployment

### Resource Requirements

- CPU: 4 cores
- Memory: 8GB
- Storage: 40GB
- Cache: 4GB Redis

### Environment Variables

```bash
USER_DB_URL=postgres://user:pass@host:5432/db
USER_REDIS_URL=redis://host:6379
USER_LOG_LEVEL=info
USER_API_PORT=50052
USER_STORAGE_URL=s3://bucket/profiles
```

## Dependency Injection & Provider Pattern

- The User service is registered and resolved via the central Provider using a DI container (`internal/service/provider.go`).
- Modular registration ensures the service is only registered once.
- Health and metrics are managed centrally and exposed for observability.
- Amadeus registration is performed at service startup for capability tracking.

## Amadeus Integration & Observability

- The service registers its capabilities and dependencies with Amadeus at startup.
- Health checks and metrics are exposed and tracked centrally.
