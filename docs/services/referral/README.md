# Documentation

version: 2025-05-14

version: 2025-05-14

version: 2025-05-14

The Referral service manages user referrals, referral codes, rewards, and referral analytics in the
OVASABI platform. It provides functionality for creating and tracking referrals, managing rewards,
and analyzing referral performance.

The Referral service now supports **tiered, multi-level referral chains** and a robust
taxing/commission model. Any user can be an original referrer to others, forming a referral ancestry
tree. Whenever a referred user (referee) performs a commercial activity (e.g., purchase, booking,
payment), a tiered payment fee (commission/tax) is distributed up the referral chain. This means:

- **Original referrers** (and all ancestors in the referral chain) receive a percentage of every
  commercial activity performed by their referees, according to a configurable tiered structure.
- The referral chain is tracked in metadata, allowing for transparent, auditable, and extensible
  commission logic.
- All referral and tax/commission data is stored in the canonical `common.Metadata` pattern, under
  `service_specific.referral` and `taxation` fields.
- The system supports unlimited tiers (configurable), and each tier can have its own percentage or
  rule.
- The referral ancestry and all payments are queryable for analytics, compliance, and reward
  distribution.

## Example: Tiered Referral Metadata

```json
{
  "metadata": {
    "service_specific": {
      "referral": {
        "chain": [
          { "user_id": "user_1", "wallet": "wallet_1", "tier": 1 },
          { "user_id": "user_2", "wallet": "wallet_2", "tier": 2 },
          { "user_id": "user_3", "wallet": "wallet_3", "tier": 3 }
        ],
        "original_referrer": { "user_id": "user_1", "wallet": "wallet_1" },
        "referee": { "user_id": "user_4", "wallet": "wallet_4" }
      }
    },
    "taxation": {
      "connectors": [
        {
          "type": "referral",
          "recipient": "user_1",
          "percentage": 0.1,
          "tiered": [
            { "tier": 1, "percentage": 0.1 },
            { "tier": 2, "percentage": 0.05 }
          ]
        },
        { "type": "referral", "recipient": "user_2", "percentage": 0.05 }
      ],
      "total_tax": 0.15
    }
  }
}
```

- `chain`: The ancestry of referrers, with tier and wallet info.
- `original_referrer`: The first user in the chain.
- `taxation.connectors`: How much each referrer receives per commercial activity.
- `total_tax`: The sum of all referral commissions for this activity.

**All referral, reward, and tax logic must use this metadata pattern for extensibility and
analytics.**

For more, see the
[Robust Metadata Pattern for Extensible Services](../../amadeus/amadeus_context.md#standard-robust-metadata-pattern-for-extensible-services)
and the `common.Metadata` proto.

## Architecture

```mermaid
graph TD
    A[gRPC API] -->|Implements| B[Service Layer]
    B -->|Uses| C[Repository Layer]
    C -->|Writes| D[Postgres]
    C -->|Caches| E[Redis]
    G[Master Repository] -->|References| D
    H[Event Log] -->|Records| D
    I[User Service] -->|References| B
    J[Campaign Service] -->|Uses| B
    K[Finance Service] -->|Rewards| B
```

## Features

1. **Referral Management**

   - Referral code generation
   - Referral tracking
   - Status management
   - Device tracking

2. **Reward System**

   - Reward allocation
   - Reward claiming
   - Reward tracking
   - Multi-tier rewards

3. **Analytics & Reporting**

   - Referral statistics
   - Performance metrics
   - Conversion tracking
   - ROI analysis

4. **Integration Features**
   - Campaign integration
   - User service integration
   - Finance service integration
   - Event logging

## API Reference

### Proto Definition

```protobuf
service ReferralService {
  rpc CreateReferral(CreateReferralRequest) returns (CreateReferralResponse) {}
  rpc GetReferral(GetReferralRequest) returns (GetReferralResponse) {}
  rpc GetReferralStats(GetReferralStatsRequest) returns (GetReferralStatsResponse) {}
}

message Referral {
  int32 id = 1;
  int32 referrer_master_id = 2;
  int32 campaign_id = 3;
  string device_hash = 4;
  string referral_code = 5;
  bool successful = 6;
  google.protobuf.Timestamp created_at = 7;
}
```

### Methods

#### CreateReferral

Creates a new referral code for a user.

```go
func (s *service) CreateReferral(ctx context.Context, req *pb.CreateReferralRequest) (*pb.CreateReferralResponse, error)
```

#### GetReferral

Retrieves a referral by referral code.

```go
func (s *service) GetReferral(ctx context.Context, req *pb.GetReferralRequest) (*pb.GetReferralResponse, error)
```

#### GetReferralStats

Retrieves referral statistics for a user.

```go
func (s *service) GetReferralStats(ctx context.Context, req *pb.GetReferralStatsRequest) (*pb.GetReferralStatsResponse, error)
```

## Data Model

### Referral Model

```go
type Referral struct {
    ID            int64     `json:"id"`
    MasterID      int64     `json:"master_id"`
    ReferrerID    int64     `json:"referrer_id"`
    RefereeID     int64     `json:"referee_id"`
    ReferralCode  string    `json:"referral_code"`
    Status        string    `json:"status"`
    RewardClaimed bool      `json:"reward_claimed"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}

type ReferralStats struct {
    TotalReferrals      int64 `json:"total_referrals"`
    SuccessfulReferrals int64 `json:"successful_referrals"`
    PendingReferrals    int64 `json:"pending_referrals"`
}
```

### Database Schema

```sql
CREATE TABLE referrals (
    id BIGSERIAL PRIMARY KEY,
    master_id BIGINT NOT NULL REFERENCES master(id) ON DELETE CASCADE,
    referrer_id BIGINT NOT NULL,
    referee_id BIGINT,
    referral_code VARCHAR(20) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL,
    reward_claimed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_referrals_referrer ON referrals(referrer_id);
CREATE INDEX idx_referrals_referee ON referrals(referee_id);
CREATE INDEX idx_referrals_code ON referrals(referral_code);
CREATE INDEX idx_referrals_status ON referrals(status);
CREATE INDEX idx_referrals_master_id ON referrals(master_id);
```

## Knowledge Graph Integration

### Capabilities

```go
type ReferralCapabilities struct {
    ReferralManagement struct {
        Creation     bool `json:"creation"`
        Tracking     bool `json:"tracking"`
        Status       bool `json:"status"`
        Device       bool `json:"device"`
    } `json:"referral_management"`

    RewardSystem struct {
        Allocation   bool `json:"allocation"`
        Claiming     bool `json:"claiming"`
        Tracking     bool `json:"tracking"`
        MultiTier    bool `json:"multi_tier"`
    } `json:"reward_system"`

    Analytics struct {
        Statistics   bool `json:"statistics"`
        Performance  bool `json:"performance"`
        Conversion   bool `json:"conversion"`
        ROI          bool `json:"roi"`
    } `json:"analytics"`
}
```

## Caching Strategy

### Key Structure

- Referral: `cache:referral:{referral_id}`
- Code: `cache:referral:code:{referral_code}`
- Stats: `cache:referral:stats:{referrer_id}`
- Campaign: `cache:referral:campaign:{campaign_id}`

### TTL Values

- Referral: 1 hour
- Code: 24 hours
- Stats: 5 minutes
- Campaign: 30 minutes

## Error Handling

### Error Types

1. **ReferralError**

   - Invalid code
   - Code generation
   - Status transition
   - Device validation

2. **RewardError**
   - Invalid claim
   - Double claim
   - Reward calculation
   - Tier validation

## Monitoring

### Metrics

1. **Referral Metrics**

   - Creation rate
   - Conversion rate
   - Success rate
   - Reward rate

2. **Performance Metrics**
   - Response times
   - Cache performance
   - Error rates
   - Resource usage

## Security

1. **Access Control**

   - Code validation
   - Device tracking
   - Rate limiting
   - Fraud detection

2. **Data Protection**
   - Code encryption
   - Device hashing
   - PII handling
   - Audit logging

## Future Improvements

1. **Phase 1 - Enhanced Referral System**

   - Multi-level referrals
   - Custom reward rules
   - Advanced tracking
   - Social integration

2. **Phase 2 - Smart Rewards**

   - Dynamic rewards
   - Reward optimization
   - Automated distribution
   - Loyalty integration

3. **Phase 3 - Advanced Analytics**
   - Predictive analytics
   - Network analysis
   - Fraud detection
   - ROI optimization

## Dependencies

- User Service (v1)
- Campaign Service (v1)
- Finance Service (v1)
- Redis Cache
- Postgres Database

## Configuration

```yaml
referral:
  database:
    pool_size: 20
    max_idle: 10
    max_lifetime: 1h
  cache:
    ttl:
      referral: 1h
      code: 24h
      stats: 5m
      campaign: 30m
  code:
    length: 8
    prefix: 'REF'
    max_attempts: 3
  rewards:
    base_points: 100
    tier_multiplier: 1.5
    max_tier: 3
  rate_limits:
    create: 10/minute
    get: 100/minute
    stats: 50/minute
```

## Deployment

### Resource Requirements

- CPU: 2 cores
- Memory: 4GB
- Storage: 20GB
- Cache: 2GB Redis

### Environment Variables

```bash
REFERRAL_DB_URL=postgres://user:pass@host:5432/db
REFERRAL_REDIS_URL=redis://host:6379
REFERRAL_LOG_LEVEL=info
REFERRAL_API_PORT=50060
```

## Dependency Injection & Provider Pattern

- The Referral service is registered and resolved via the central Provider using a DI container
  (`internal/service/provider.go`).
- Modular registration ensures the service is only registered once.
- Health and metrics are managed centrally and exposed for observability.
- Amadeus registration is performed at service startup for capability tracking.

## Amadeus Integration & Observability

- The service registers its capabilities and dependencies with Amadeus at startup.
- Health checks and metrics are exposed and tracked centrally.
