# Broadcast Service

The Broadcast service manages real-time message distribution, pub/sub operations, and streaming
communications in the OVASABI platform.

## Architecture

```mermaid
graph TD
    A[gRPC API] -->|Implements| B[Service Layer]
    B -->|Uses| C[Repository Layer]
    C -->|Writes| D[Postgres]
    C -->|Caches| E[Redis]
    C -->|Streams| F[Message Queue]
    G[Master Repository] -->|References| D
    H[Event Log] -->|Records| D
    I[Auth Service] -->|Validates| B
    J[User Service] -->|References| B
    K[Notification Service] -->|Notifies| B
```

## Features

1. **Message Broadcasting**

   - Real-time messaging
   - Topic-based publishing
   - Message persistence
   - Delivery tracking

2. **Subscription Management**

   - Topic subscriptions
   - Channel management
   - Access control
   - Subscription tracking

3. **Stream Processing**

   - Message streaming
   - Stream aggregation
   - Flow control
   - Back-pressure handling

4. **Delivery System**
   - Guaranteed delivery
   - Message ordering
   - Retry mechanisms
   - Dead letter handling

## API Reference

### Proto Definition

```protobuf
service BroadcastService {
    rpc PublishMessage(PublishMessageRequest) returns (MessageResponse);
    rpc Subscribe(SubscribeRequest) returns (stream MessageResponse);
    rpc Unsubscribe(UnsubscribeRequest) returns (UnsubscribeResponse);
    rpc GetTopics(GetTopicsRequest) returns (TopicsResponse);
    rpc CreateTopic(CreateTopicRequest) returns (TopicResponse);
    rpc DeleteTopic(DeleteTopicRequest) returns (DeleteTopicResponse);
    rpc GetMessageHistory(GetMessageHistoryRequest) returns (MessageHistoryResponse);
}
```

### Methods

#### PublishMessage

Publishes a message to specified topics.

```go
func (s *service) PublishMessage(ctx context.Context, req *pb.PublishMessageRequest) (*pb.MessageResponse, error)
```

#### Subscribe

Subscribes to topics and receives message stream.

```go
func (s *service) Subscribe(req *pb.SubscribeRequest, stream pb.BroadcastService_SubscribeServer) error
```

## Data Model

### Message Model

```go
type MessageModel struct {
    ID            uuid.UUID
    TopicID       uuid.UUID
    PublisherID   uuid.UUID
    Type          string
    Content       map[string]interface{} // JSONB in DB
    Metadata      map[string]interface{} // JSONB in DB
    Status        string // pending, delivered, failed
    DeliveredAt   *time.Time
    RetryCount    int
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type TopicModel struct {
    ID            uuid.UUID
    Name          string
    Description   string
    AccessLevel   string // public, private, protected
    Metadata      map[string]interface{} // JSONB in DB
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type SubscriptionModel struct {
    ID            uuid.UUID
    TopicID       uuid.UUID
    SubscriberID  uuid.UUID
    Status        string // active, paused, blocked
    LastSeenAt    *time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### Database Schema

```sql
CREATE TABLE broadcast_messages (
    id UUID PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id),
    topic_id UUID NOT NULL,
    publisher_id UUID NOT NULL,
    type TEXT NOT NULL,
    content JSONB NOT NULL,
    metadata JSONB DEFAULT '{}',
    status TEXT NOT NULL,
    delivered_at TIMESTAMPTZ,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE broadcast_topics (
    id UUID PRIMARY KEY,
    master_id INTEGER NOT NULL REFERENCES master(id),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    access_level TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE broadcast_subscriptions (
    id UUID PRIMARY KEY,
    topic_id UUID NOT NULL REFERENCES broadcast_topics(id),
    subscriber_id UUID NOT NULL,
    status TEXT NOT NULL,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(topic_id, subscriber_id)
);

CREATE INDEX idx_broadcast_messages_topic ON broadcast_messages(topic_id);
CREATE INDEX idx_broadcast_messages_publisher ON broadcast_messages(publisher_id);
CREATE INDEX idx_broadcast_messages_status ON broadcast_messages(status);
CREATE INDEX idx_broadcast_messages_created ON broadcast_messages(created_at);
CREATE INDEX idx_broadcast_topics_name ON broadcast_topics(name);
CREATE INDEX idx_broadcast_topics_access ON broadcast_topics(access_level);
CREATE INDEX idx_broadcast_subscriptions_topic ON broadcast_subscriptions(topic_id);
CREATE INDEX idx_broadcast_subscriptions_subscriber ON broadcast_subscriptions(subscriber_id);
```

## Knowledge Graph

### Capabilities

```go
type BroadcastCapabilities struct {
    MessageBroadcast struct {
        RealTime     bool `json:"real_time"`
        Persistence  bool `json:"persistence"`
        Ordering     bool `json:"ordering"`
        Tracking     bool `json:"tracking"`
    } `json:"message_broadcast"`

    SubscriptionManagement struct {
        Topics       bool `json:"topics"`
        Channels     bool `json:"channels"`
        Access       bool `json:"access"`
        Tracking     bool `json:"tracking"`
    } `json:"subscription_management"`

    StreamProcessing struct {
        Streaming    bool `json:"streaming"`
        Aggregation  bool `json:"aggregation"`
        FlowControl  bool `json:"flow_control"`
        BackPressure bool `json:"back_pressure"`
    } `json:"stream_processing"`

    DeliverySystem struct {
        Guaranteed   bool `json:"guaranteed"`
        Ordering     bool `json:"ordering"`
        Retry        bool `json:"retry"`
        DeadLetter   bool `json:"dead_letter"`
    } `json:"delivery_system"`
}
```

### Growth Patterns

```mermaid
graph TD
    A[Basic Broadcast] -->|Add Persistence| B[Reliable Broadcast]
    B -->|Add Tracking| C[Smart Broadcast]
    C -->|Add Analytics| D[Intelligent Broadcast]

    E[Basic Topics] -->|Add Access| F[Secure Topics]
    F -->|Add Analytics| G[Smart Topics]
    G -->|Add ML| H[Intelligent Topics]

    I[Basic Streams] -->|Add Control| J[Managed Streams]
    J -->|Add Metrics| K[Monitored Streams]
    K -->|Add AI| L[Smart Streams]

    M[Basic Delivery] -->|Add Guarantee| N[Reliable Delivery]
    N -->|Add Ordering| O[Ordered Delivery]
    O -->|Add Recovery| P[Resilient Delivery]
```

### Evolution Tracking

```go
type BroadcastEvolution struct {
    BroadcastCapabilities   []string `json:"broadcast_capabilities"`
    TopicCapabilities      []string `json:"topic_capabilities"`
    StreamCapabilities     []string `json:"stream_capabilities"`
    DeliveryCapabilities   []string `json:"delivery_capabilities"`
    Version               string    `json:"version"`
    LastUpdated           time.Time `json:"last_updated"`
}
```

## Caching Strategy

### Key Structure

- Message Cache: `cache:broadcast:message:{message_id}`
- Topic Info: `cache:broadcast:topic:{topic_id}`
- Subscriptions: `cache:broadcast:subs:{topic_id}`
- User Topics: `cache:broadcast:user:{user_id}:topics`
- Stream Status: `cache:broadcast:stream:{stream_id}`

### TTL Values

- Message Cache: 1 hour
- Topic Info: 6 hours
- Subscriptions: 15 minutes
- User Topics: 30 minutes
- Stream Status: 5 minutes

## Error Handling

### Error Types

1. **PublishError**

   - Invalid topic
   - Access denied
   - Rate limit exceeded
   - Message too large

2. **SubscriptionError**

   - Invalid subscription
   - Topic not found
   - Access denied
   - Connection error

3. **StreamError**
   - Stream closed
   - Flow control
   - Timeout
   - Buffer overflow

## Monitoring

### Metrics

1. **Message Metrics**

   - Message throughput
   - Delivery success rate
   - Processing time
   - Queue depth

2. **Performance Metrics**
   - Stream latency
   - Connection count
   - Error rates
   - Resource usage

## Security

1. **Access Control**

   - Topic-level access
   - User permissions
   - Rate limiting
   - Connection limits

2. **Message Security**
   - Content validation
   - Size limits
   - Source verification
   - Encryption

## Future Improvements

1. **Phase 1 - Enhanced Streaming**

   - Advanced flow control
   - Smart routing
   - Priority queues
   - Message compression

2. **Phase 2 - Smart Delivery**

   - ML-based routing
   - Predictive scaling
   - Smart retries
   - Load balancing

3. **Phase 3 - Advanced Features**
   - Message transformation
   - Content filtering
   - Smart batching
   - Analytics integration

## Dependencies

- Auth Service (v1)
- User Service (v1)
- Notification Service (v1)
- Redis Cache
- Postgres Database
- Message Queue (RabbitMQ/Kafka)

## Configuration

```yaml
broadcast:
  database:
    pool_size: 20
    max_idle: 10
    max_lifetime: 1h
  cache:
    ttl:
      message: 1h
      topic: 6h
      subscriptions: 15m
      user_topics: 30m
      stream: 5m
  message:
    max_size: 1048576 # 1MB
    retention: 7d
    batch_size: 100
  stream:
    buffer_size: 1000
    timeout: 30s
    retry_limit: 3
  queue:
    provider: 'rabbitmq'
    prefetch: 100
    concurrent: 10
```

## Deployment

### Resource Requirements

- CPU: 8 cores
- Memory: 16GB
- Storage: 100GB
- Cache: 8GB Redis
- Queue: 8GB RAM

### Environment Variables

```bash
BROADCAST_DB_URL=postgres://user:pass@host:5432/db
BROADCAST_REDIS_URL=redis://host:6379
BROADCAST_LOG_LEVEL=info
BROADCAST_API_PORT=50056
BROADCAST_QUEUE_URL=amqp://user:pass@host:5672
```
