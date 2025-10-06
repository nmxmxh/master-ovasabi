# Streaming Events Guide

## Overview

The OVASABI platform now supports **streaming events** through the enhanced Provider system. This
allows services to emit continuous streams of data in real-time, perfect for live updates, progress
tracking, and real-time communication.

## How Streaming Events Work

### 1. **Automatic Detection**

The Provider automatically detects streaming events based on patterns:

- Events containing `stream_`, `:stream:`, `:typing:`, `:presence:`, `:chunks:`, `:live:`,
  `:realtime:`
- Specific event types like `messaging:stream_messages:v1:requested`

### 2. **Event Lifecycle for Streaming**

```
Client sends: messaging:stream_messages:v1:requested
↓
Provider emits: messaging:stream_messages:v1:started
↓
Service streams data continuously
↓
Provider emits: messaging:stream_messages:v1:stream (for each data chunk)
↓
Provider emits: messaging:stream_messages:v1:success (when complete)
```

### 3. **Built-in Features**

- **Heartbeat**: Automatic heartbeat every second to keep streams alive
- **Timeout**: 30-minute default timeout (configurable)
- **Error Handling**: Automatic panic recovery and error event emission
- **Progress Tracking**: Stream count and timing information

## Implementation Examples

### Basic Streaming Service

```go
package messaging

import (
    "context"
    "time"
    nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
)

// Implement StreamingEventHandler interface
func (s *Service) HandleStreamingEvent(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
    // Extract request data
    userID := extractUserID(event)
    conversationID := extractConversationID(event)

    // Start streaming messages
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // Get new messages from database
            messages, err := s.getNewMessages(userID, conversationID)
            if err != nil {
                return err
            }

            // Send each message as a stream event
            for _, msg := range messages {
                streamEvent := &nexusv1.EventResponse{
                    EventType: "messaging:stream_messages:v1:stream",
                    EventId:   generateEventID(),
                    Payload: &commonpb.Payload{
                        Data: convertMessageToStruct(msg),
                    },
                }

                select {
                case streamData <- streamEvent:
                    // Successfully queued
                case <-ctx.Done():
                    return ctx.Err()
                }
            }
        }
    }
}
```

### Real-time Analytics Streaming

```go
func (s *AnalyticsService) HandleStreamingEvent(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
    // Stream analytics data every 5 seconds
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // Collect metrics
            metrics := s.collectMetrics()

            streamEvent := &nexusv1.EventResponse{
                EventType: "analytics:stream_metrics:v1:stream",
                EventId:   generateEventID(),
                Payload: &commonpb.Payload{
                    Data: convertMetricsToStruct(metrics),
                },
            }

            select {
            case streamData <- streamEvent:
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    }
}
```

### File Upload Progress Streaming

```go
func (s *MediaService) HandleStreamingEvent(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
    // Extract upload information
    uploadID := extractUploadID(event)

    // Stream upload progress
    for progress := range s.uploadProgress(uploadID) {
        streamEvent := &nexusv1.EventResponse{
            EventType: "media:stream_upload_progress:v1:stream",
            EventId:   generateEventID(),
            Payload: &commonpb.Payload{
                Data: &structpb.Struct{
                    Fields: map[string]*structpb.Value{
                        "upload_id": structpb.NewStringValue(uploadID),
                        "progress":  structpb.NewNumberValue(progress.Percentage),
                        "bytes_sent": structpb.NewNumberValue(float64(progress.BytesSent)),
                        "total_bytes": structpb.NewNumberValue(float64(progress.TotalBytes)),
                    },
                },
            },
        }

        select {
        case streamData <- streamEvent:
        case <-ctx.Done():
            return ctx.Err()
        }
    }

    return nil
}
```

## Event Types for Streaming

### Predefined Streaming Events

- `messaging:stream_messages:v1:requested` - Real-time message streaming
- `messaging:stream_typing:v1:requested` - Typing indicators
- `messaging:stream_presence:v1:requested` - User presence updates
- `notification:stream_asset_chunks:v1:requested` - File chunk streaming
- `media:stream_media_content:v1:requested` - Media content streaming
- `search:stream_results:v1:requested` - Search result streaming
- `analytics:stream_metrics:v1:requested` - Analytics data streaming

### Custom Streaming Events

You can create custom streaming events by following the pattern:

```
{service}:stream_{action}:v1:requested
```

Examples:

- `campaign:stream_analytics:v1:requested`
- `user:stream_activity:v1:requested`
- `product:stream_inventory:v1:requested`

## Client-Side Usage

### WebSocket Integration

```typescript
// Subscribe to streaming events
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = event => {
  const data = JSON.parse(event.data);

  switch (data.type) {
    case 'messaging:stream_messages:v1:stream':
      // Handle new message
      displayMessage(data.payload);
      break;

    case 'messaging:stream_messages:v1:started':
      // Stream started
      showStreamingIndicator();
      break;

    case 'messaging:stream_messages:v1:success':
      // Stream completed
      hideStreamingIndicator();
      break;

    case 'messaging:stream_messages:v1:heartbeat':
      // Keep-alive heartbeat
      updateLastSeen();
      break;
  }
};

// Start streaming
ws.send(
  JSON.stringify({
    type: 'messaging:stream_messages:v1:requested',
    payload: {
      user_id: 'user123',
      conversation_id: 'conv456'
    }
  })
);
```

### gRPC Streaming

```go
// Subscribe to streaming events via gRPC
req := &nexusv1.SubscribeRequest{
    EventTypes: []string{"messaging:stream_messages:v1:requested"},
}

stream, err := nexusClient.SubscribeEvents(ctx, req)
if err != nil {
    log.Fatal(err)
}

for {
    event, err := stream.Recv()
    if err != nil {
        log.Fatal(err)
    }

    switch event.EventType {
    case "messaging:stream_messages:v1:stream":
        // Handle streamed message
        handleMessage(event.Payload)
    case "messaging:stream_messages:v1:success":
        // Stream completed
        log.Info("Stream completed")
    }
}
```

## Configuration

### Timeout Configuration

```go
// In your service initialization
provider := service.NewProvider(logger, nexusClient)
provider.SetStreamTimeout(60 * time.Minute) // Custom timeout
```

### Heartbeat Configuration

```go
// Customize heartbeat interval
provider.SetHeartbeatInterval(5 * time.Second)
```

## Best Practices

1. **Always implement the StreamingEventHandler interface** for proper streaming support
2. **Use appropriate buffer sizes** for streamData channels (default: 100)
3. **Handle context cancellation** properly to clean up resources
4. **Implement proper error handling** and recovery
5. **Use heartbeats** to detect dead connections
6. **Limit stream duration** to prevent resource leaks
7. **Monitor stream performance** and adjust buffer sizes as needed

## Monitoring and Debugging

The Provider automatically logs:

- Stream start/completion events
- Heartbeat emissions
- Error conditions and panics
- Stream statistics (count, duration)

Check logs for patterns like:

```
INFO Processing streaming event event_type=messaging:stream_messages:v1:requested
INFO Event processing succeeded action=stream_messages stream_count=42
WARN Failed to emit stream event stream_count=15 error=connection reset
```

This streaming system provides a robust foundation for real-time features across the OVASABI
platform!
