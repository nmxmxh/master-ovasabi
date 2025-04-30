# Notification Service

The Notification service manages user notifications, delivery channels, preferences, and
notification lifecycle in the OVASABI platform.

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
    K[Broadcast Service] -->|Publishes| B
    L[External Providers] -->|Delivers| B
```

## Features

1. **Notification Management**

   - Notification creation
   - Template management
   - Priority handling
   - Lifecycle tracking

2. **Delivery Channels**

   - Email notifications
   - Push notifications
   - SMS notifications
   - In-app notifications

3. **User Preferences**

   - Channel preferences
   - Frequency control
   - Category settings
   - Time zone handling

4. **Delivery System**
   - Multi-channel delivery
   - Delivery scheduling
   - Retry handling
   - Rate limiting

## API Reference

### Proto Definition

```protobuf
service NotificationService {
    rpc SendNotification(SendNotificationRequest) returns (NotificationResponse);
    rpc GetNotification(GetNotificationRequest) returns (NotificationResponse);
    rpc ListNotifications(ListNotificationsRequest) returns (ListNotificationsResponse);
    rpc UpdatePreferences(UpdatePreferencesRequest) returns (PreferencesResponse);
    rpc GetPreferences(GetPreferencesRequest) returns (PreferencesResponse);
    rpc MarkAsRead(MarkAsReadRequest) returns (NotificationResponse);
    rpc DeleteNotification(DeleteNotificationRequest) returns (DeleteNotificationResponse);
}
```

### Methods

#### SendNotification

Sends a notification through specified channels.

```go
func (s *service) SendNotification(ctx context.Context, req *pb.SendNotificationRequest) (*pb.NotificationResponse, error)
```

#### UpdatePreferences

Updates user notification preferences.

```go
func (s *service) UpdatePreferences(ctx context.Context, req *pb.UpdatePreferencesRequest) (*pb.PreferencesResponse, error)
```

## Data Model

### Notification Model

```go
type NotificationModel struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    Type        string
    Title       string
    Content     map[string]interface{} // JSONB in DB
    Channels    []string
    Priority    string // high, medium, low
    Status      string // pending, sent, failed, read
    Metadata    map[string]interface{} // JSONB in DB
    ReadAt      *time.Time
    SentAt      *time.Time
    ExpiresAt   *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type PreferencesModel struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    Channels    map[string]bool           // enabled channels
    Categories  map[string]ChannelConfig  // per-category settings
    Schedule    NotificationSchedule
    TimeZone    string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type ChannelConfig struct {
    Enabled     bool
    Frequency   string // instant, daily, weekly
    Quiet       bool
    TimeWindow  []TimeRange
}

type NotificationSchedule struct {
    QuietHours  []TimeRange
    ActiveDays  []string
    Frequency   map[string]int // max notifications per timeframe
}
```

### Database Schema

```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id),
    user_id UUID NOT NULL,
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    content JSONB NOT NULL,
    channels TEXT[] NOT NULL,
    priority TEXT NOT NULL,
    status TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    read_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id),
    user_id UUID NOT NULL UNIQUE,
    channels JSONB NOT NULL,
    categories JSONB NOT NULL,
    schedule JSONB NOT NULL,
    timezone TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_type ON notifications(type);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_created ON notifications(created_at);
CREATE INDEX idx_notifications_channels ON notifications USING gin(channels);
CREATE INDEX idx_notifications_content ON notifications USING gin(content);
CREATE INDEX idx_notification_preferences_user ON notification_preferences(user_id);
```

## Knowledge Graph

### Capabilities

```go
type NotificationCapabilities struct {
    NotificationManagement struct {
        Creation     bool `json:"creation"`
        Templates    bool `json:"templates"`
        Priority     bool `json:"priority"`
        Lifecycle    bool `json:"lifecycle"`
    } `json:"notification_management"`

    DeliveryChannels struct {
        Email        bool `json:"email"`
        Push         bool `json:"push"`
        SMS          bool `json:"sms"`
        InApp        bool `json:"in_app"`
    } `json:"delivery_channels"`

    UserPreferences struct {
        Channels     bool `json:"channels"`
        Frequency    bool `json:"frequency"`
        Categories   bool `json:"categories"`
        TimeZones    bool `json:"time_zones"`
    } `json:"user_preferences"`

    DeliverySystem struct {
        MultiChannel bool `json:"multi_channel"`
        Scheduling   bool `json:"scheduling"`
        Retry        bool `json:"retry"`
        RateLimit    bool `json:"rate_limit"`
    } `json:"delivery_system"`
}
```

### Growth Patterns

```mermaid
graph TD
    A[Basic Notifications] -->|Add Channels| B[Multi-Channel]
    B -->|Add Preferences| C[Smart Notifications]
    C -->|Add ML| D[Intelligent Notifications]

    E[Basic Delivery] -->|Add Scheduling| F[Timed Delivery]
    F -->|Add Priority| G[Smart Delivery]
    G -->|Add Analytics| H[Optimized Delivery]

    I[Basic Preferences] -->|Add Categories| J[Category Prefs]
    J -->|Add Schedule| K[Smart Prefs]
    K -->|Add ML| L[Adaptive Prefs]

    M[Basic Templates] -->|Add Dynamic| N[Dynamic Templates]
    N -->|Add Personalization| O[Personal Templates]
    O -->|Add AI| P[Smart Templates]
```

### Evolution Tracking

```go
type NotificationEvolution struct {
    NotificationCapabilities []string `json:"notification_capabilities"`
    ChannelCapabilities     []string `json:"channel_capabilities"`
    PreferenceCapabilities  []string `json:"preference_capabilities"`
    DeliveryCapabilities    []string `json:"delivery_capabilities"`
    Version                string    `json:"version"`
    LastUpdated            time.Time `json:"last_updated"`
}
```

## Caching Strategy

### Key Structure

- Notification: `cache:notification:{notification_id}`
- User Notifications: `cache:notification:user:{user_id}`
- Preferences: `cache:notification:prefs:{user_id}`
- Templates: `cache:notification:template:{template_id}`
- Counters: `cache:notification:counter:{user_id}:{type}`

### TTL Values

- Notification: 1 hour
- User Notifications: 15 minutes
- Preferences: 30 minutes
- Templates: 6 hours
- Counters: 1 hour

## Error Handling

### Error Types

1. **DeliveryError**

   - Channel unavailable
   - Invalid recipient
   - Rate limit exceeded
   - Template error

2. **PreferenceError**

   - Invalid settings
   - Channel disabled
   - Schedule conflict
   - Category error

3. **ValidationError**
   - Invalid content
   - Missing template
   - Invalid channel
   - Expired notification

## Monitoring

### Metrics

1. **Delivery Metrics**

   - Delivery success rate
   - Channel performance
   - Processing time
   - Queue depth

2. **Performance Metrics**
   - API latency
   - Cache hit ratio
   - Error rates
   - Channel availability

## Security

1. **Access Control**

   - User permissions
   - Channel verification
   - Rate limiting
   - Content validation

2. **Data Protection**
   - Content encryption
   - Channel security
   - PII handling
   - Audit logging

## Future Improvements

1. **Phase 1 - Enhanced Delivery**

   - Smart channel selection
   - Dynamic scheduling
   - Priority optimization
   - Batch processing

2. **Phase 2 - Smart Notifications**

   - ML-based timing
   - Content personalization
   - Channel optimization
   - Engagement prediction

3. **Phase 3 - Advanced Features**
   - Rich media support
   - Interactive notifications
   - Cross-channel coordination
   - Analytics integration

## Dependencies

- Auth Service (v1)
- User Service (v1)
- Broadcast Service (v1)
- Redis Cache
- Postgres Database
- Message Queue (RabbitMQ/Kafka)
- External Providers (Email, Push, SMS)

## Configuration

```yaml
notification:
  database:
    pool_size: 20
    max_idle: 10
    max_lifetime: 1h
  cache:
    ttl:
      notification: 1h
      user_notifications: 15m
      preferences: 30m
      templates: 6h
      counters: 1h
  delivery:
    max_retries: 3
    batch_size: 100
    rate_limit: 100/minute
  channels:
    email:
      provider: 'sendgrid'
      template_path: '/templates/email'
    push:
      provider: 'firebase'
      credentials: '/config/firebase.json'
    sms:
      provider: 'twilio'
      timeout: 5s
  preferences:
    default_timezone: 'UTC'
    quiet_hours: ['22:00-06:00']
    max_per_hour: 10
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
NOTIFICATION_DB_URL=postgres://user:pass@host:5432/db
NOTIFICATION_REDIS_URL=redis://host:6379
NOTIFICATION_LOG_LEVEL=info
NOTIFICATION_API_PORT=50057
NOTIFICATION_QUEUE_URL=amqp://user:pass@host:5672
NOTIFICATION_EMAIL_KEY=your-email-key
NOTIFICATION_PUSH_KEY=your-push-key
NOTIFICATION_SMS_KEY=your-sms-key
```
