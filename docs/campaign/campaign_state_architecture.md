# Campaign State Management Architecture

## Overview

The campaign state management system provides real-time, direct access to campaign data without
requiring database round-trips for every operation. This architecture enables high-performance
campaign updates, real-time synchronization across multiple clients, and event-driven state
management.

## Architecture Components

### Backend Components

#### 1. CampaignStateManager (`internal/server/nexus/campaign_state.go`)

The central component that manages all campaign states in memory:

- **In-Memory State**: Maintains campaign state in `sync.Map` for thread-safe concurrent access
- **Real-time Subscribers**: Manages subscriber channels for immediate state update notifications
- **Event Processing**: Handles campaign-related events and updates state accordingly
- **Asynchronous Persistence**: Optionally persists state changes to database in background

#### 2. Event Types

The system supports the following canonical event types:

- `campaign:state:request` - Request current campaign state
- `campaign:update:v1:requested` - Direct campaign state update
- `campaign:feature:v1:requested` - Feature-specific updates (add/remove/set)
- `campaign:config:v1:requested` - Configuration updates (UI, scripts, communication)

#### 3. Nexus Server Integration (`internal/server/nexus/server.go`)

The Nexus server routes campaign events to the CampaignStateManager and handles real-time event
distribution via Redis pub/sub.

### Frontend Components

#### 1. Global State Store (`frontend/src/store/global.ts`)

Enhanced Zustand store with campaign-specific functions:

- **Campaign Update Functions**: Direct methods for updating campaigns
- **Real-time State Sync**: Automatic state updates from WebSocket events
- **Event Payloads**: Proactive state management for better performance
- **Service State Integration**: Generic service state management for campaigns

#### 2. React Hooks

Convenient hooks for campaign state management:

- `useCampaignState()` - Access and update campaign state
- `useCampaignUpdates()` - Campaign update functions
- `useServiceState('campaign')` - Generic service state access

#### 3. WASM Bridge

Handles type conversion and event routing between frontend and backend via WebSocket.

## Key Benefits

### 1. Performance Advantages

- **Sub-100ms Updates**: Direct memory operations without database latency
- **Reduced Database Load**: Frequent reads served from memory cache
- **Asynchronous Persistence**: Database updates don't block UI operations
- **Efficient Serialization**: Optimized data structures for fast updates

### 2. Real-time Capabilities

- **Instant Synchronization**: All connected clients receive updates immediately
- **Multi-user Coordination**: Concurrent users see changes in real-time
- **Event-driven Architecture**: State changes trigger immediate notifications
- **Conflict-free Operations**: Centralized state manager prevents race conditions

### 3. Scalability

- **Horizontal Scaling**: Redis pub/sub enables multi-instance coordination
- **Connection Management**: Efficient subscriber management with cleanup
- **Memory Efficiency**: Campaign states are created on-demand
- **Background Processing**: Non-blocking asynchronous operations

### 4. Developer Experience

- **Type Safety**: Full TypeScript/Go type safety for state operations
- **Unified API**: Single interface for all campaign operations
- **Event Transparency**: All operations are observable through event bus
- **Error Handling**: Graceful error handling with detailed logging

## Usage Examples

### Frontend Usage

```typescript
import { useCampaignState, useCampaignUpdates } from './store/global';

function CampaignManager() {
  const { state, updateCampaign, updateFeatures, updateConfig } = useCampaignState();
  const { updateCampaign, updateCampaignFeatures, updateCampaignConfig } = useCampaignUpdates();

  // Update campaign directly
  const handleUpdate = () => {
    updateCampaign({
      title: 'Updated Campaign Title',
      description: 'New description'
    });
  };

  // Add features
  const addFeatures = () => {
    updateCampaignFeatures(['analytics', 'notifications'], 'add');
  };

  // Update UI configuration
  const updateUI = () => {
    updateCampaignConfig('ui_content', {
      banner: 'Welcome to our updated campaign!',
      theme: 'dark'
    });
  };

  return (
    <div>
      <h2>{state.title}</h2>
      <p>Features: {state.features?.join(', ')}</p>
      <button onClick={handleUpdate}>Update Campaign</button>
      <button onClick={addFeatures}>Add Features</button>
      <button onClick={updateUI}>Update UI</button>
    </div>
  );
}
```

### Backend Usage

```go
// Handle campaign update event
func (m *CampaignStateManager) handleCampaignUpdate(event *nexusv1.EventRequest) {
    var payload struct {
        CampaignID string         `json:"campaignId"`
        Updates    map[string]any `json:"updates"`
    }

    // Extract data from event payload
    // ...

    // Update state directly (immediate)
    m.UpdateState(payload.CampaignID, userID, payload.Updates, event.Metadata)

    // Persist to database asynchronously (background)
    if m.repo != nil {
        m.safeGo(func() {
            m.persistToDB(payload.CampaignID, payload.Updates)
        })
    }
}
```

## Event Flow

### 1. Frontend → Backend Update Flow

```mermaid
[React Component]
    ↓ useCampaignUpdates().updateCampaign()
[Global Store]
    ↓ emitEvent('campaign:update:v1:requested')
[WASM Bridge]
    ↓ WebSocket message
[Nexus Server]
    ↓ HandleEvent()
[CampaignStateManager]
    ↓ UpdateState() + persistToDB()
[Redis Event Bus]
    ↓ Real-time event broadcast
[All Connected Clients]
```

### 2. Real-time Sync Flow

```mermaid
[Campaign State Update]
    ↓
[Feedback Bus] → [Redis Pub/Sub] → [All Nexus Instances]
    ↓                                      ↓
[WebSocket] → [All Connected Clients] → [Frontend State Update]
```

## Configuration

### Environment Variables

- `CAMPAIGN_STATE_ENABLED` - Enable campaign state manager (default: true)
- `CAMPAIGN_PERSISTENCE_ENABLED` - Enable database persistence (default: true)
- `CAMPAIGN_SUBSCRIBER_BUFFER` - Subscriber channel buffer size (default: 16)

### Default Campaign Loading

The system automatically loads the default campaign from `start/default_campaign.json` on startup,
providing immediate campaign state availability.

## Monitoring and Debugging

### Logging

All campaign state operations are logged with structured logging:

```go
m.log.Info("Processing campaign update",
    zap.String("campaign_id", campaignID),
    zap.String("user_id", userID),
    zap.Any("updates", updates))
```

### Event Tracing

Every state change generates events that can be traced:

- Event correlation IDs for request/response tracking
- Timestamp tracking for performance analysis
- Payload inspection for debugging

### Health Monitoring

- Campaign state metrics (active campaigns, subscribers)
- Performance metrics (update latency, persistence timing)
- Error tracking (failed updates, persistence errors)

## Migration Guide

### From Database-Only Updates

1. **Replace direct database calls** with campaign state manager updates
2. **Add event listeners** for real-time state changes
3. **Update frontend** to use new campaign hooks
4. **Test real-time synchronization** across multiple clients

### From Polling-Based Updates

1. **Remove polling intervals** and replace with event subscriptions
2. **Update state management** to use event-driven updates
3. **Implement WebSocket reconnection** for reliability
4. **Add offline state handling** for network interruptions

## Best Practices

### 1. State Management

- Use campaign state for frequently accessed data
- Persist critical changes to database asynchronously
- Handle network partitions gracefully
- Implement proper cleanup for subscribers

### 2. Performance Optimization

- Batch multiple updates when possible
- Use feature-specific updates for granular control
- Monitor memory usage for large campaign counts
- Implement cache eviction for inactive campaigns

### 3. Error Handling

- Implement retry logic for failed persistence
- Handle WebSocket disconnections gracefully
- Provide fallback to database queries when needed
- Log all errors with sufficient context

### 4. Security

- Validate all campaign updates before processing
- Check user permissions for campaign modifications
- Sanitize input data to prevent injection attacks
- Audit all campaign state changes

## Future Enhancements

1. **Campaign State Versioning** - Track state history and enable rollbacks
2. **Distributed Locking** - Coordinate updates across multiple instances
3. **State Snapshots** - Periodic state backups for recovery
4. **Analytics Integration** - Real-time campaign performance metrics
5. **A/B Testing Support** - Different states for different user segments
