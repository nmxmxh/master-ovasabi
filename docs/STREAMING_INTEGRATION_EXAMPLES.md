# Streaming Events Integration Examples

## Overview

This document shows how streaming events integrate with **Campaign State Management** and **Media
Streaming** while **enhancing** (not replacing) existing event files.

## 🎯 Event Files Are NOT Obsolete

Event files (`events.go`) remain the **primary business logic layer**. Streaming events **enhance**
them by adding:

1. **Real-time data streaming** capabilities
2. **Automatic lifecycle management**
3. **Built-in error handling and recovery**
4. **Progress tracking and heartbeats**

## Campaign State + Streaming Integration

### 1. Enhanced Campaign State Manager

```go
// internal/server/nexus/campaign_state_streaming.go
package nexus

import (
    "context"
    "time"
    nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
    "github.com/nmxmxh/master-ovasabi/internal/service"
)

// CampaignStateStreamingHandler implements StreamingEventHandler for campaign state
func (m *CampaignStateManager) HandleStreamingEvent(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
    eventType := event.GetEventType()

    switch eventType {
    case "campaign:stream_state:v1:requested":
        return m.streamCampaignState(ctx, event, streamData)
    case "campaign:stream_analytics:v1:requested":
        return m.streamCampaignAnalytics(ctx, event, streamData)
    case "campaign:stream_events:v1:requested":
        return m.streamCampaignEvents(ctx, event, streamData)
    default:
        return fmt.Errorf("unknown streaming event type: %s", eventType)
    }
}

// streamCampaignState streams real-time campaign state updates
func (m *CampaignStateManager) streamCampaignState(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
    campaignID, userID := m.extractCampaignAndUserID(event)

    // Create a ticker for periodic state updates
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    lastState := make(map[string]any)

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // Get current campaign state
            currentState := m.GetState(campaignID)

            // Only stream if state has changed
            if !maps.Equal(currentState, lastState) {
                stateEvent := &nexusv1.EventResponse{
                    EventType: "campaign:stream_state:v1:stream",
                    EventId:   generateEventID(),
                    Payload: &commonpb.Payload{
                        Data: meta.NewStructFromMap(currentState, nil),
                    },
                    Metadata: &commonpb.Metadata{
                        ServiceSpecific: &structpb.Struct{
                            Fields: map[string]*structpb.Value{
                                "campaign_id": structpb.NewStringValue(campaignID),
                                "user_id":     structpb.NewStringValue(userID),
                                "timestamp":   structpb.NewStringValue(time.Now().UTC().Format(time.RFC3339)),
                            },
                        },
                    },
                }

                select {
                case streamData <- stateEvent:
                    // Successfully queued
                    lastState = make(map[string]any)
                    maps.Copy(lastState, currentState)
                case <-ctx.Done():
                    return ctx.Err()
                }
            }
        }
    }
}

// streamCampaignAnalytics streams real-time analytics data
func (m *CampaignStateManager) streamCampaignAnalytics(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
    campaignID, _ := m.extractCampaignAndUserID(event)

    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // Collect analytics data
            analytics := m.collectCampaignAnalytics(campaignID)

            analyticsEvent := &nexusv1.EventResponse{
                EventType: "campaign:stream_analytics:v1:stream",
                EventId:   generateEventID(),
                Payload: &commonpb.Payload{
                    Data: meta.NewStructFromMap(analytics, nil),
                },
            }

            select {
            case streamData <- analyticsEvent:
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    }
}

// collectCampaignAnalytics gathers real-time analytics
func (m *CampaignStateManager) collectCampaignAnalytics(campaignID string) map[string]any {
    cs := m.GetOrCreateState(campaignID)

    // Count active subscribers
    subscriberCount := 0
    cs.Subscribers.Range(func(_, _ interface{}) bool {
        subscriberCount++
        return true
    })

    return map[string]any{
        "campaign_id":       campaignID,
        "subscriber_count":  subscriberCount,
        "last_updated":      cs.LastUpdated,
        "state_keys":        len(cs.State),
        "active_features":   cs.State["features"],
        "timestamp":         time.Now().UTC().Format(time.RFC3339),
    }
}
```

### 2. Integration with Existing Campaign Events

```go
// internal/service/campaign/events.go (Enhanced, not replaced)
package campaign

// Existing event handlers remain unchanged
func HandleCampaignServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
    // ... existing logic ...
}

// NEW: Add streaming event handlers
func (s *Service) HandleStreamingEvent(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
    // Delegate to campaign state manager for streaming
    return s.campaignStateManager.HandleStreamingEvent(ctx, event, streamData)
}

// Register streaming events alongside existing events
func init() {
    // Existing event registrations remain
    RegisterActionHandler("list", handleCampaignList)
    RegisterActionHandler("update", handleCampaignUpdate)
    // ... other existing handlers ...

    // NEW: Register streaming events
    RegisterActionHandler("stream_state", handleStreamState)
    RegisterActionHandler("stream_analytics", handleStreamAnalytics)
}

func handleStreamState(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
    // This will be handled by the streaming system automatically
    // The Provider will detect "stream_state" and route to HandleStreamingEvent
}
```

## Media Streaming + Campaign State Integration

### 1. Enhanced Media Streaming Service

```go
// cmd/media-streaming/streaming_events.go
package main

import (
    "context"
    "encoding/json"
    nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
    commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
)

// MediaStreamingHandler implements StreamingEventHandler
func (s *Server) HandleStreamingEvent(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
    eventType := event.GetEventType()

    switch eventType {
    case "media:stream_webrtc:v1:requested":
        return s.streamWebRTCConnections(ctx, event, streamData)
    case "media:stream_rooms:v1:requested":
        return s.streamRoomUpdates(ctx, event, streamData)
    case "media:stream_peers:v1:requested":
        return s.streamPeerUpdates(ctx, event, streamData)
    default:
        return fmt.Errorf("unknown media streaming event type: %s", eventType)
    }
}

// streamWebRTCConnections streams WebRTC connection status
func (s *Server) streamWebRTCConnections(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
    campaignID := extractCampaignID(event)

    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // Get all rooms for this campaign
            s.roomsMu.RLock()
            var roomStats []map[string]any
            for key, room := range s.rooms {
                if strings.HasPrefix(key, campaignID+":") {
                    room.mu.RLock()
                    peerCount := len(room.Peers)
                    roomStats = append(roomStats, map[string]any{
                        "room_key":     key,
                        "context_id":   room.ContextID,
                        "peer_count":   peerCount,
                        "state_keys":   len(room.State),
                        "last_activity": time.Now().UTC().Format(time.RFC3339),
                    })
                    room.mu.RUnlock()
                }
            }
            s.roomsMu.RUnlock()

            webrtcEvent := &nexusv1.EventResponse{
                EventType: "media:stream_webrtc:v1:stream",
                EventId:   generateEventID(),
                Payload: &commonpb.Payload{
                    Data: &structpb.Struct{
                        Fields: map[string]*structpb.Value{
                            "campaign_id": structpb.NewStringValue(campaignID),
                            "rooms":       convertToStructValue(roomStats),
                            "total_rooms": structpb.NewNumberValue(float64(len(roomStats))),
                            "timestamp":   structpb.NewStringValue(time.Now().UTC().Format(time.RFC3339)),
                        },
                    },
                },
            }

            select {
            case streamData <- webrtcEvent:
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    }
}

// streamRoomUpdates streams room state changes
func (s *Server) streamRoomUpdates(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
    campaignID := extractCampaignID(event)

    // Subscribe to room changes
    roomUpdates := make(chan RoomUpdate, 100)
    s.subscribeToRoomUpdates(campaignID, roomUpdates)
    defer s.unsubscribeFromRoomUpdates(campaignID, roomUpdates)

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case update := <-roomUpdates:
            roomEvent := &nexusv1.EventResponse{
                EventType: "media:stream_rooms:v1:stream",
                EventId:   generateEventID(),
                Payload: &commonpb.Payload{
                    Data: &structpb.Struct{
                        Fields: map[string]*structpb.Value{
                            "campaign_id":  structpb.NewStringValue(campaignID),
                            "room_key":     structpb.NewStringValue(update.RoomKey),
                            "action":       structpb.NewStringValue(update.Action),
                            "peer_count":   structpb.NewNumberValue(float64(update.PeerCount)),
                            "state":        convertToStructValue(update.State),
                            "timestamp":    structpb.NewStringValue(update.Timestamp),
                        },
                    },
                },
            }

            select {
            case streamData <- roomEvent:
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    }
}

type RoomUpdate struct {
    RoomKey   string
    Action    string // "peer_joined", "peer_left", "state_changed"
    PeerCount int
    State     map[string]any
    Timestamp string
}
```

### 2. Integration with Existing Media Streaming

```go
// cmd/media-streaming/main.go (Enhanced, not replaced)
package main

// Existing WebSocket handling remains unchanged
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    // ... existing WebSocket logic ...
}

// NEW: Add streaming event support
func (s *Server) subscribeToStreamingEvents(ctx context.Context, campaignID int64) {
    go func() {
        stream, err := s.nexusClient.Client.SubscribeEvents(ctx, &nexusv1.SubscribeRequest{
            EventTypes: []string{
                "media:stream_webrtc:v1:requested",
                "media:stream_rooms:v1:requested",
                "media:stream_peers:v1:requested",
            },
            CampaignId: campaignID,
        })
        if err != nil {
            s.logger.Error("Failed to subscribe to streaming events", zap.Error(err))
            return
        }

        for {
            event, err := stream.Recv()
            if err != nil {
                s.logger.Error("Streaming event subscription error", zap.Error(err))
                return
            }

            // Handle streaming events
            s.handleStreamingEvent(ctx, event)
        }
    }()
}
```

## Complete Integration Flow

### 1. **Event Files Define Business Logic** (Unchanged)

```go
// internal/service/campaign/events.go
func HandleCampaignServiceEvent(ctx context.Context, s *Service, event *nexusv1.EventResponse) {
    // Business logic for campaign events
    switch event.EventType {
    case "campaign:update:v1:requested":
        handleCampaignUpdate(ctx, s, event)
    case "campaign:list:v1:requested":
        handleCampaignList(ctx, s, event)
    }
}
```

### 2. **Provider Adds Lifecycle Management** (Automatic)

```
campaign:update:v1:requested → Provider → campaign:update:v1:started → Business Logic → campaign:update:v1:success
```

### 3. **Streaming Events Add Real-time Capabilities** (New)

```
campaign:stream_state:v1:requested → Provider → campaign:stream_state:v1:started → Stream Data → campaign:stream_state:v1:success
```

### 4. **Campaign State Manager Integrates Both** (Enhanced)

```go
// Handles both regular events AND streaming events
func (m *CampaignStateManager) HandleEvent(ctx context.Context, event *nexusv1.EventRequest) {
    // Regular event handling (unchanged)
    if strings.HasSuffix(event.EventType, ":requested") {
        // ... existing logic ...
    }
}

func (m *CampaignStateManager) HandleStreamingEvent(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
    // NEW: Streaming event handling
    // ... streaming logic ...
}
```

## Benefits of This Integration

### 1. **Event Files Remain Central**

- All business logic stays in `events.go` files
- No breaking changes to existing code
- Clear separation of concerns

### 2. **Streaming Events Add Value**

- Real-time data streaming
- Automatic lifecycle management
- Built-in error handling and recovery
- Progress tracking and heartbeats

### 3. **Campaign State Gets Real-time Updates**

- Live state changes streamed to clients
- Analytics data streamed in real-time
- Event history streamed as it happens

### 4. **Media Streaming Gets Enhanced**

- WebRTC connection status streaming
- Room updates in real-time
- Peer activity streaming
- Integration with campaign state

## Migration Strategy

### Phase 1: Add Streaming Support (No Breaking Changes)

1. Implement `StreamingEventHandler` interface in existing services
2. Add streaming event types to Provider
3. Test streaming functionality alongside existing events

### Phase 2: Integrate with Campaign State

1. Add streaming methods to `CampaignStateManager`
2. Connect streaming events to campaign state updates
3. Test real-time state streaming

### Phase 3: Integrate with Media Streaming

1. Add streaming support to media streaming service
2. Connect WebRTC events to streaming system
3. Test real-time media streaming

### Phase 4: Frontend Integration

1. Update frontend to handle streaming events
2. Add real-time UI updates
3. Test end-to-end streaming

## Conclusion

**Event files are NOT obsolete** - they're the foundation that streaming events build upon. The
streaming system:

- ✅ **Enhances** existing event handling
- ✅ **Adds** real-time capabilities
- ✅ **Maintains** backward compatibility
- ✅ **Integrates** with campaign state and media streaming
- ✅ **Provides** automatic lifecycle management

This creates a powerful, unified event system that handles both traditional request/response
patterns and modern real-time streaming patterns!


