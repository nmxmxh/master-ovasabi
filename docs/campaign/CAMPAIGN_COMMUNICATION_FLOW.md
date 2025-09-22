# Campaign Communication Flow Documentation

## Overview

The campaign communication system provides real-time, event-driven communication between the
frontend, backend services, and campaign state management. This document explains the complete flow
and architecture.

## System Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐    ┌─────────────────────┐
│   Frontend      │    │  Media Streaming │    │   Nexus Server  │    │ Campaign State      │
│   (React)       │◄──►│   Service        │◄──►│                 │◄──►│ Manager             │
│                 │    │  (WebSocket)     │    │                 │    │                     │
└─────────────────┘    └──────────────────┘    └─────────────────┘    └─────────────────────┘
         │                       │                       │                       │
         │                       │                       │                       │
         ▼                       ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐    ┌─────────────────────┐
│   Global Store  │    │   Redis Event    │    │   Event Bus     │    │   Database          │
│   (Zustand)     │    │   Bus            │    │   (Multi-       │    │   (PostgreSQL)      │
│                 │    │                  │    │   instance)     │    │                     │
└─────────────────┘    └──────────────────┘    └─────────────────┘    └─────────────────────┘
```

## Communication Flow

### 1. Frontend to Backend Communication

#### Step 1: User Action

```typescript
// User clicks "Update Campaign" button
const handleCampaignUpdate = () => {
  updateCampaign(
    {
      title: 'New Campaign Title',
      status: 'active'
    },
    response => {
      // Handle response
    }
  );
};
```

#### Step 2: Event Emission

```typescript
// Global store emits event
emitEvent({
  type: 'campaign:update:v1:requested',
  payload: {
    campaign_id: '0',
    updates: { title: 'New Campaign Title', status: 'active' }
  },
  metadata: {
    campaign: { campaignId: 0 },
    user: { userId: 'user123' },
    device: { deviceId: 'device456' },
    session: { sessionId: 'session789' }
  }
});
```

#### Step 3: WebSocket Transmission

```typescript
// WebSocket sends to media streaming service
ws.send(
  JSON.stringify({
    type: 'campaign:update:v1:requested',
    payload: {
      /* ... */
    },
    metadata: {
      /* ... */
    }
  })
);
```

### 2. Backend Processing

#### Step 4: Media Streaming Service

```go
// cmd/media-streaming/main.go
func (s *Server) handleWebSocketMessage(conn *websocket.Conn, msg []byte) {
    var event nexusv1.EventRequest
    json.Unmarshal(msg, &event)

    // Forward to Nexus server
    s.nexusClient.EmitEvent(context.Background(), &event)
}
```

#### Step 5: Nexus Server Processing

```go
// internal/server/nexus/server.go
func (s *Server) EmitEvent(ctx context.Context, req *nexusv1.EventRequest) (*nexusv1.EventResponse, error) {
    // Validate event type
    if !isCanonicalEventType(req.EventType) {
        return &nexusv1.EventResponse{Success: false}, nil
    }

    // Handle campaign events
    if strings.HasPrefix(req.EventType, "campaign:") {
        s.campaignStateMgr.HandleEvent(ctx, req)
    }

    // Publish to event bus
    s.PublishEvent(envelope)

    return &nexusv1.EventResponse{Success: true}, nil
}
```

#### Step 6: Campaign State Manager

```go
// internal/server/nexus/campaign_state.go
func (m *CampaignStateManager) HandleEvent(ctx context.Context, event *nexusv1.EventRequest) {
    switch event.EventType {
    case "campaign:update:v1:requested":
        m.handleCampaignUpdate(ctx, event)
    case "campaign:feature:v1:requested":
        m.handleFeatureUpdate(ctx, event)
    case "campaign:config:v1:requested":
        m.handleConfigUpdate(ctx, event)
    }
}
```

### 3. State Management and Persistence

#### Step 7: State Update

```go
func (m *CampaignStateManager) UpdateState(campaignID, userID string, update map[string]any, metadata *commonpb.Metadata) {
    cs := m.GetOrCreateState(campaignID)

    // Apply updates
    maps.Copy(cs.State, update)
    cs.LastUpdated = time.Now()

    // Create event response
    event := &nexusv1.EventResponse{
        Success:   true,
        EventId:   fmt.Sprintf("state_update:%s:%s:%d", campaignID, userID, time.Now().UnixNano()),
        EventType: "campaign:state:v1:success",
        Message:   "state_updated",
        Payload:   &commonpb.Payload{Data: structData},
    }

    // Notify subscribers
    m.notifySubscribers(cs, event)

    // Persist to database
    if m.repo != nil {
        go m.persistToDB(ctx, campaignID, update)
    }
}
```

#### Step 8: Database Persistence

```go
func (m *CampaignStateManager) persistToDB(ctx context.Context, campaignID string, updates map[string]any) {
    campaign, err := m.repo.GetBySlug(ctx, campaignID)
    if err != nil {
        m.log.Error("Failed to get campaign for persistence", zap.Error(err))
        return
    }

    // Update metadata with state changes
    // ... merge state into campaign metadata ...

    // Update in database
    if err := m.repo.Update(ctx, campaign); err != nil {
        m.log.Error("Failed to persist campaign state to database", zap.Error(err))
        return
    }
}
```

### 4. Real-time Broadcasting

#### Step 9: Event Bus Distribution

```go
func (s *Server) PublishEvent(event *nexusv1.EventResponse) {
    service, action := parseServiceAction(event.EventType)
    key := service + ":" + action

    if bus, ok := s.eventBuses[key]; ok {
        bus.Publish(event)
    } else {
        s.eventBus.Publish(event) // fallback to default
    }
}
```

#### Step 10: Redis Event Bus

```go
func (b *RedisEventBus) Publish(event *nexusv1.EventResponse) {
    data, err := json.Marshal(event)
    if err != nil {
        b.log.Error("Failed to marshal event", zap.Error(err))
        return
    }

    if err := b.client.Publish(b.ctx, b.channel, data).Err(); err != nil {
        b.log.Error("Failed to publish event to Redis", zap.Error(err))
    }
}
```

### 5. Frontend Real-time Updates

#### Step 11: WebSocket Reception

```typescript
// Frontend receives real-time updates
ws.onmessage = event => {
  const data = JSON.parse(event.data);

  if (data.type === 'campaign:state:v1:success') {
    // Update campaign state in global store
    updateCampaignState(data.payload);

    // Trigger UI updates
    triggerUIUpdate(data);
  }
};
```

#### Step 12: State Synchronization

```typescript
// Global store updates
const updateCampaignState = (payload: any) => {
  set(state => ({
    campaignState: {
      ...state.campaignState,
      state: payload
    }
  }));
};
```

## Event Types and Patterns

### Canonical Event Types

```
{service}:{action}:v{version}:{state}

Examples:
- campaign:update:v1:requested
- campaign:update:v1:success
- campaign:feature:v1:requested
- campaign:config:v1:success
- campaign:state:v1:success
```

### Event States

- `requested` - Initial request
- `started` - Processing began
- `success` - Operation completed successfully
- `failed` - Operation failed
- `completed` - Final completion state

## Real-time Features

### 1. WebSocket Connection Management

- Automatic reconnection with exponential backoff
- Connection state monitoring
- Message queuing during disconnection

### 2. Event Deduplication

- Distributed locking using Redis
- Event ID uniqueness validation
- Duplicate event prevention

### 3. Subscriber Management

- Channel-based subscriptions
- Automatic cleanup of disconnected clients
- Load balancing across multiple instances

### 4. State Synchronization

- Real-time state updates
- Conflict resolution
- Optimistic updates with rollback

## Performance Optimizations

### 1. Logging Optimization

- Reduced verbosity for media streaming operations
- Debug-level logging for routine operations
- Error-level logging for critical failures

### 2. Event Bus Optimization

- Worker pool for event delivery
- Non-blocking event publishing
- Channel buffering for high-throughput scenarios

### 3. Database Optimization

- Asynchronous persistence
- Connection pooling
- Query optimization

## Error Handling

### 1. Connection Errors

- Automatic reconnection
- Graceful degradation
- Error state management

### 2. Event Processing Errors

- Panic recovery
- Error logging
- Fallback mechanisms

### 3. State Consistency

- Transaction management
- Rollback capabilities
- Conflict resolution

## Security Considerations

### 1. Event Validation

- Canonical event type validation
- Payload sanitization
- Metadata validation

### 2. Access Control

- User authentication
- Campaign authorization
- Feature-level permissions

### 3. Data Protection

- GDPR compliance
- Data encryption
- Audit logging

## Monitoring and Observability

### 1. Metrics

- Event throughput
- Connection counts
- Error rates
- Response times

### 2. Logging

- Structured logging with Zap
- Correlation IDs
- Request tracing

### 3. Health Checks

- Service health endpoints
- Database connectivity
- Redis connectivity

## Usage Examples

### Frontend Integration

```typescript
import { useCampaignUpdates, useCampaignState } from '../store/global';

function CampaignManager() {
  const { updateCampaign, updateCampaignFeatures } = useCampaignUpdates();
  const campaignState = useCampaignState();

  const handleUpdate = () => {
    updateCampaign({
      title: "New Title",
      status: "active"
    }, (response) => {
      console.log('Campaign updated:', response);
    });
  };

  return (
    <div>
      <h1>{campaignState.state?.title}</h1>
      <button onClick={handleUpdate}>Update Campaign</button>
    </div>
  );
}
```

### Backend Event Handling

```go
func (m *CampaignStateManager) handleCampaignUpdate(ctx context.Context, event *nexusv1.EventRequest) {
    campaignID, userID := m.extractCampaignAndUserID(event)

    var payload struct {
        CampaignID string         `json:"campaign_id"`
        Updates    map[string]any `json:"updates"`
    }

    // Extract payload
    if event.Payload != nil && event.Payload.Data != nil {
        payloadMap := event.Payload.Data.AsMap()
        if cid, ok := payloadMap["campaignId"].(string); ok {
            payload.CampaignID = cid
        }
        if updates, ok := payloadMap["updates"].(map[string]any); ok {
            payload.Updates = updates
        }
    }

    // Update state
    m.UpdateState(payload.CampaignID, userID, payload.Updates, event.Metadata)

    // Persist to database
    if m.repo != nil {
        go m.persistToDB(ctx, payload.CampaignID, payload.Updates)
    }
}
```

## Conclusion

The campaign communication system provides a robust, scalable, and real-time communication platform
that enables:

1. **Real-time Updates**: Instant state synchronization across all clients
2. **Event-driven Architecture**: Decoupled, scalable event processing
3. **High Performance**: Optimized logging, event delivery, and database operations
4. **Reliability**: Error handling, reconnection, and state consistency
5. **Observability**: Comprehensive monitoring and logging

This system forms the foundation for real-time campaign management and provides a solid base for
future enhancements and features.
