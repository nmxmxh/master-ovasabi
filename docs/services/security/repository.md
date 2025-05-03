# Security Service Repository Design

## Overview

Following the master-client/service/event pattern defined in our database practices, the security
service repository implements a robust data persistence layer with special considerations for
security-related data.

## Schema Design

### 1. Master Tables

#### security_master

```sql
CREATE TABLE security_master (
    id SERIAL PRIMARY KEY,
    type VARCHAR(50) NOT NULL, -- 'identity', 'pattern', 'incident', etc.
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    metadata JSONB
);

CREATE INDEX idx_security_master_type ON security_master(type);
CREATE INDEX idx_security_master_status ON security_master(status);
CREATE INDEX idx_security_master_created ON security_master(created_at);
```

### 2. Service Tables

#### security_identity

```sql
CREATE TABLE security_identity (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES security_master(id),
    identity_type VARCHAR(50) NOT NULL,
    identifier VARCHAR(255) NOT NULL,
    credentials JSONB NOT NULL,
    attributes JSONB,
    last_authentication TIMESTAMPTZ,
    risk_score FLOAT,
    UNIQUE(identity_type, identifier)
);

CREATE INDEX idx_security_identity_master ON security_identity(master_id);
CREATE INDEX idx_security_identity_type ON security_identity(identity_type);
```

#### security_pattern

```sql
CREATE TABLE security_pattern (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES security_master(id),
    pattern_name VARCHAR(255) NOT NULL,
    description TEXT,
    vertices JSONB NOT NULL,
    edges JSONB NOT NULL,
    constraints JSONB,
    risk_assessment JSONB NOT NULL,
    UNIQUE(pattern_name)
);

CREATE INDEX idx_security_pattern_master ON security_pattern(master_id);
CREATE INDEX idx_security_pattern_risk ON security_pattern((risk_assessment->>'risk_score'));
```

#### security_incident

```sql
CREATE TABLE security_incident (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES security_master(id),
    incident_type VARCHAR(100) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    description TEXT,
    detection_time TIMESTAMPTZ NOT NULL,
    resolution_time TIMESTAMPTZ,
    context JSONB,
    risk_assessment JSONB
);

CREATE INDEX idx_security_incident_master ON security_incident(master_id);
CREATE INDEX idx_security_incident_type ON security_incident(incident_type);
CREATE INDEX idx_security_incident_severity ON security_incident(severity);
```

### 3. Event Tables

#### security_event

```sql
CREATE TABLE security_event (
    id SERIAL PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES security_master(id),
    event_type VARCHAR(100) NOT NULL,
    principal VARCHAR(255) NOT NULL,
    resource VARCHAR(255),
    action VARCHAR(100),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    context JSONB,
    metadata JSONB
);

CREATE INDEX idx_security_event_master ON security_event(master_id);
CREATE INDEX idx_security_event_type ON security_event(event_type);
CREATE INDEX idx_security_event_principal ON security_event(principal);
CREATE INDEX idx_security_event_occurred ON security_event(occurred_at);
```

## Repository Interface

```go
type SecurityRepository interface {
    // Identity Management
    CreateIdentity(ctx context.Context, identity *Identity) error
    GetIdentity(ctx context.Context, identityType, identifier string) (*Identity, error)
    UpdateIdentityRiskScore(ctx context.Context, id int64, score float64) error

    // Pattern Management
    RegisterPattern(ctx context.Context, pattern *SecurityPattern) error
    GetPattern(ctx context.Context, patternName string) (*SecurityPattern, error)
    ListPatterns(ctx context.Context, filter PatternFilter) ([]*SecurityPattern, error)

    // Incident Management
    RecordIncident(ctx context.Context, incident *SecurityIncident) error
    GetIncident(ctx context.Context, incidentID string) (*SecurityIncident, error)
    ListIncidents(ctx context.Context, filter IncidentFilter) ([]*SecurityIncident, error)
    UpdateIncidentResolution(ctx context.Context, id int64, resolution *IncidentResolution) error

    // Event Management
    RecordEvent(ctx context.Context, event *SecurityEvent) error
    GetEvents(ctx context.Context, filter EventFilter) ([]*SecurityEvent, error)

    // Analytics
    GetSecurityMetrics(ctx context.Context, filter MetricsFilter) (*SecurityMetrics, error)
    GetRiskAssessment(ctx context.Context, resourceID string) (*RiskAssessment, error)
}
```

## Implementation Guidelines

### 1. Data Protection

```go
type EncryptedField struct {
    Data      []byte    // Encrypted data
    KeyID     string    // Key version used for encryption
    Algorithm string    // Encryption algorithm
    Timestamp time.Time // Encryption timestamp
}
```

- Use column-level encryption for sensitive fields
- Implement key rotation mechanisms
- Apply access control at the row level
- Implement audit logging for all data access

### 2. Performance Considerations

1. **Indexing Strategy**

   - Index frequently queried fields
   - Use partial indexes for specific queries
   - Implement composite indexes for common query patterns

2. **Query Optimization**
   ```sql
   -- Example of an optimized security pattern query
   WITH RECURSIVE pattern_match AS (
       SELECT id, vertices, edges
       FROM security_pattern
       WHERE risk_score > threshold
       AND pattern_type = 'authentication'
   )
   SELECT * FROM pattern_match
   WHERE EXISTS (
       SELECT 1
       FROM jsonb_array_elements(vertices) v
       WHERE v->>'type' = 'critical_asset'
   );
   ```

### 3. Data Retention

```sql
CREATE TABLE security_archive (
    id SERIAL PRIMARY KEY,
    original_table VARCHAR(100) NOT NULL,
    original_id INTEGER NOT NULL,
    data JSONB NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

- Implement automated archival processes
- Define retention periods based on data type
- Maintain compliance with regulatory requirements

### 4. Transaction Management

```go
func (r *repository) WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
    tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelSerializable,
        ReadOnly:  false,
    })
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }

    if err := fn(tx); err != nil {
        if rbErr := tx.Rollback(); rbErr != nil {
            return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
        }
        return err
    }

    return tx.Commit()
}
```

## Integration with Knowledge Graph

### 1. Graph Updates

```go
type SecurityGraphUpdate struct {
    Operation    string          // 'create', 'update', 'delete'
    EntityType   string          // 'identity', 'pattern', 'incident'
    EntityID     string
    Relationships []Relationship
    Metadata      map[string]interface{}
}
```

### 2. Pattern Matching

```go
func (r *repository) MatchSecurityPatterns(ctx context.Context, graph *GraphQuery) ([]*SecurityPattern, error) {
    // Implementation using graph database capabilities
    // Integrates with the knowledge graph for pattern detection
}
```

## References

1. Database Practices:

   - Master-Client Pattern
   - Service-Event Architecture
   - Analytics Readiness

2. Security Standards:

   - NIST Database Security Guidelines
   - GDPR Data Storage Requirements
   - ISO 27001 Database Controls

3. Performance:
   - PostgreSQL Security Best Practices
   - JSONB Performance Optimization
   - Index Strategy Guidelines
