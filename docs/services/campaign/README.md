# Campaign Service

The Campaign service manages marketing campaigns, user targeting, content delivery, and campaign
analytics in the OVASABI platform.

## Architecture

```mermaid
graph TD
    A[gRPC API] -->|Implements| B[Service Layer]
    B -->|Uses| C[Repository Layer]
    C -->|Writes| D[Postgres]
    C -->|Caches| E[Redis]
    C -->|Queues| F[Message Queue]
    G[Master Repository] -->|References| D
    H[Event Log] -->|Records| D
    I[Auth Service] -->|Validates| B
    J[User Service] -->|References| B
    K[Notification Service] -->|Delivers| B
    L[Broadcast Service] -->|Publishes| B
```

## Features

1. **Campaign Management**

   - Campaign creation
   - Content management
   - Schedule planning
   - Status tracking

2. **Audience Targeting**

   - User segmentation
   - Targeting rules
   - A/B testing
   - Audience analytics

3. **Content Delivery**

   - Multi-channel delivery
   - Content personalization
   - Delivery scheduling
   - Rate control

4. **Performance Analytics**
   - Campaign metrics
   - Engagement tracking
   - Conversion analysis
   - ROI measurement

## API Reference

### Proto Definition

```protobuf
service CampaignService {
    rpc CreateCampaign(CreateCampaignRequest) returns (CampaignResponse);
    rpc GetCampaign(GetCampaignRequest) returns (CampaignResponse);
    rpc ListCampaigns(ListCampaignsRequest) returns (ListCampaignsResponse);
    rpc UpdateCampaign(UpdateCampaignRequest) returns (CampaignResponse);
    rpc DeleteCampaign(DeleteCampaignRequest) returns (DeleteCampaignResponse);
    rpc StartCampaign(StartCampaignRequest) returns (CampaignResponse);
    rpc StopCampaign(StopCampaignRequest) returns (CampaignResponse);
    rpc GetCampaignMetrics(GetCampaignMetricsRequest) returns (CampaignMetricsResponse);
}
```

### Methods

#### CreateCampaign

Creates a new marketing campaign.

```go
func (s *service) CreateCampaign(ctx context.Context, req *pb.CreateCampaignRequest) (*pb.CampaignResponse, error)
```

#### GetCampaignMetrics

Retrieves campaign performance metrics.

```go
func (s *service) GetCampaignMetrics(ctx context.Context, req *pb.GetCampaignMetricsRequest) (*pb.CampaignMetricsResponse, error)
```

## Data Model

### Campaign Model

```go
type CampaignModel struct {
    ID              uuid.UUID
    Name            string
    Description     string
    Type            string // email, push, sms, multi
    Status          string // draft, active, paused, completed
    StartTime       *time.Time
    EndTime         *time.Time
    TargetAudience  AudienceConfig
    Content         map[string]interface{} // JSONB in DB
    Schedule        CampaignSchedule
    Metrics         CampaignMetrics
    Metadata        map[string]interface{} // JSONB in DB
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type AudienceConfig struct {
    Segments        []string
    Rules           []TargetingRule
    ExcludedUsers   []uuid.UUID
    TestGroups      []TestGroup
    EstimatedSize   int
}

type TargetingRule struct {
    Field      string
    Operator   string // equals, contains, greater_than, etc.
    Value      interface{}
    Priority   int
}

type CampaignSchedule struct {
    Frequency       string // once, daily, weekly, monthly
    TimeWindows     []TimeWindow
    MaxDeliveryRate int
    TimeZone        string
}

type CampaignMetrics struct {
    Delivered       int
    Opened          int
    Clicked         int
    Converted       int
    Failed          int
    Revenue         decimal.Decimal
    ROI             decimal.Decimal
}
```

### Database Schema

```sql
CREATE TABLE campaigns (
    id UUID PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id),
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    target_audience JSONB NOT NULL,
    content JSONB NOT NULL,
    schedule JSONB NOT NULL,
    metrics JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE campaign_events (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns(id),
    user_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_campaigns_type ON campaigns(type);
CREATE INDEX idx_campaigns_status ON campaigns(status);
CREATE INDEX idx_campaigns_start_time ON campaigns(start_time);
CREATE INDEX idx_campaigns_end_time ON campaigns(end_time);
CREATE INDEX idx_campaign_events_campaign ON campaign_events(campaign_id);
CREATE INDEX idx_campaign_events_user ON campaign_events(user_id);
CREATE INDEX idx_campaign_events_type ON campaign_events(event_type);
CREATE INDEX idx_campaigns_target_audience ON campaigns USING gin(target_audience);
CREATE INDEX idx_campaigns_content ON campaigns USING gin(content);
```

## Knowledge Graph

### Capabilities

```go
type CampaignCapabilities struct {
    CampaignManagement struct {
        Creation     bool `json:"creation"`
        Content      bool `json:"content"`
        Scheduling   bool `json:"scheduling"`
        Tracking     bool `json:"tracking"`
    } `json:"campaign_management"`

    AudienceTargeting struct {
        Segmentation bool `json:"segmentation"`
        Rules        bool `json:"rules"`
        Testing      bool `json:"testing"`
        Analytics    bool `json:"analytics"`
    } `json:"audience_targeting"`

    ContentDelivery struct {
        MultiChannel    bool `json:"multi_channel"`
        Personalization bool `json:"personalization"`
        Scheduling     bool `json:"scheduling"`
        RateControl    bool `json:"rate_control"`
    } `json:"content_delivery"`

    Analytics struct {
        Metrics     bool `json:"metrics"`
        Engagement  bool `json:"engagement"`
        Conversion  bool `json:"conversion"`
        ROI         bool `json:"roi"`
    } `json:"analytics"`
}
```

### Growth Patterns

```mermaid
graph TD
    A[Basic Campaign] -->|Add Targeting| B[Smart Campaign]
    B -->|Add Analytics| C[Intelligent Campaign]
    C -->|Add ML| D[Predictive Campaign]

    E[Basic Audience] -->|Add Rules| F[Smart Audience]
    F -->|Add Testing| G[Test Audience]
    G -->|Add ML| H[Intelligent Audience]

    I[Basic Content] -->|Add Channels| J[Multi-Channel]
    J -->|Add Personal| K[Personal Content]
    K -->|Add AI| L[Smart Content]

    M[Basic Analytics] -->|Add Events| N[Event Analytics]
    N -->|Add ML| O[Predictive Analytics]
    O -->|Add AI| P[Intelligent Analytics]
```

### Evolution Tracking

```go
type CampaignEvolution struct {
    CampaignCapabilities    []string `json:"campaign_capabilities"`
    AudienceCapabilities    []string `json:"audience_capabilities"`
    DeliveryCapabilities    []string `json:"delivery_capabilities"`
    AnalyticsCapabilities   []string `json:"analytics_capabilities"`
    Version                string    `json:"version"`
    LastUpdated            time.Time `json:"last_updated"`
}
```

## Caching Strategy

### Key Structure

- Campaign: `cache:campaign:{campaign_id}`
- Campaign List: `cache:campaign:list:{status}`
- Campaign Metrics: `cache:campaign:metrics:{campaign_id}`
- Audience Rules: `cache:campaign:rules:{campaign_id}`
- Event Counters: `cache:campaign:events:{campaign_id}:{type}`

### TTL Values

- Campaign: 1 hour
- Campaign List: 15 minutes
- Campaign Metrics: 5 minutes
- Audience Rules: 30 minutes
- Event Counters: 1 minute

## Error Handling

### Error Types

1. **CampaignError**

   - Invalid configuration
   - Schedule conflict
   - Resource limit
   - Status transition

2. **AudienceError**

   - Invalid rules
   - Empty segment
   - Test group error
   - Targeting conflict

3. **DeliveryError**
   - Channel unavailable
   - Rate limit
   - Content error
   - Schedule error

## Monitoring

### Metrics

1. **Campaign Metrics**

   - Active campaigns
   - Delivery rate
   - Success rate
   - ROI tracking

2. **Performance Metrics**
   - Processing time
   - Queue depth
   - Error rates
   - Resource usage

## Security

1. **Access Control**

   - Role permissions
   - Content approval
   - Rate limits
   - Audit logging

2. **Data Protection**
   - Audience privacy
   - Content security
   - Metrics encryption
   - PII handling

## Future Improvements

1. **Phase 1 - Enhanced Targeting**

   - Advanced segmentation
   - Dynamic rules
   - Real-time targeting
   - Behavioral targeting

2. **Phase 2 - Smart Campaigns**

   - AI-driven optimization
   - Automated testing
   - Content optimization
   - Timing optimization

3. **Phase 3 - Advanced Analytics**
   - Predictive analytics
   - Attribution modeling
   - Customer journey tracking
   - ROI optimization

## Dependencies

- Auth Service (v1)
- User Service (v1)
- Notification Service (v1)
- Broadcast Service (v1)
- Redis Cache
- Postgres Database
- Message Queue (RabbitMQ/Kafka)

## Configuration

```yaml
campaign:
  database:
    pool_size: 20
    max_idle: 10
    max_lifetime: 1h
  cache:
    ttl:
      campaign: 1h
      campaign_list: 15m
      metrics: 5m
      rules: 30m
      events: 1m
  delivery:
    max_rate: 1000/minute
    batch_size: 100
    retry_limit: 3
  targeting:
    max_rules: 10
    max_segments: 5
    max_test_groups: 3
  analytics:
    update_interval: 5m
    retention_days: 90
    metrics_precision: 2
```

## Deployment

### Resource Requirements

- CPU: 4 cores
- Memory: 8GB
- Storage: 100GB
- Cache: 4GB Redis
- Queue: 4GB RAM

### Environment Variables

```bash
CAMPAIGN_DB_URL=postgres://user:pass@host:5432/db
CAMPAIGN_REDIS_URL=redis://host:6379
CAMPAIGN_LOG_LEVEL=info
CAMPAIGN_API_PORT=50058
CAMPAIGN_QUEUE_URL=amqp://user:pass@host:5672
```

## Dependency Injection & Provider Pattern

- The Campaign service is registered and resolved via the central Provider using a DI container (`internal/service/provider.go`).
- Modular registration ensures the service is only registered once.
- Health and metrics are managed centrally and exposed for observability.
- Amadeus registration is performed at service startup for capability tracking.

## Babel & Location-Based Pricing

- The Campaign service integrates with the Babel service for dynamic, location-based pricing rules and i18n support.
- Campaign targeting and pricing logic can leverage Babel for advanced segmentation and localization.

## Amadeus Integration & Observability

- The service registers its capabilities and dependencies with Amadeus at startup.
- Health checks and metrics are exposed and tracked centrally.
