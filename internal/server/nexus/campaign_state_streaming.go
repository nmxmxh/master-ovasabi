package nexus

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	meta "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// CampaignStateStreamingHandler implements StreamingEventHandler for campaign state.
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

// streamCampaignState streams real-time campaign state updates.
func (m *CampaignStateManager) streamCampaignState(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
	campaignID, userID := m.extractCampaignAndUserIDFromResponse(event)

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
					EventId:   generateStreamingEventID(),
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

// streamCampaignAnalytics streams real-time analytics data.
func (m *CampaignStateManager) streamCampaignAnalytics(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
	campaignID, _ := m.extractCampaignAndUserIDFromResponse(event)

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
				EventId:   generateStreamingEventID(),
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

// streamCampaignEvents streams real-time event history.
func (m *CampaignStateManager) streamCampaignEvents(ctx context.Context, event *nexusv1.EventResponse, streamData chan<- *nexusv1.EventResponse) error {
	campaignID, _ := m.extractCampaignAndUserIDFromResponse(event)

	// Subscribe to campaign events
	eventChan := make(chan *nexusv1.EventResponse, 100)
	m.subscribeToCampaignEvents(campaignID, eventChan)
	defer m.unsubscribeFromCampaignEvents(campaignID, eventChan)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case campaignEvent := <-eventChan:
			// Forward campaign events as stream events
			streamEvent := &nexusv1.EventResponse{
				EventType: "campaign:stream_events:v1:stream",
				EventId:   generateStreamingEventID(),
				Payload:   campaignEvent.Payload,
				Metadata:  campaignEvent.Metadata,
			}

			select {
			case streamData <- streamEvent:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// collectCampaignAnalytics gathers real-time analytics.
func (m *CampaignStateManager) collectCampaignAnalytics(campaignID string) map[string]any {
	cs := m.GetOrCreateState(campaignID)

	// Count active subscribers
	subscriberCount := 0
	cs.Subscribers.Range(func(_, _ interface{}) bool {
		subscriberCount++
		return true
	})

	return map[string]any{
		"campaign_id":      campaignID,
		"subscriber_count": subscriberCount,
		"last_updated":     cs.LastUpdated,
		"state_keys":       len(cs.State),
		"active_features":  cs.State["features"],
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}
}

// extractCampaignAndUserIDFromResponse extracts IDs from EventResponse using unified extractor.
func (m *CampaignStateManager) extractCampaignAndUserIDFromResponse(event *nexusv1.EventResponse) (campaignID, userID string) {
	if event.Metadata == nil {
		return "", ""
	}

	// Use unified metadata extractor for consistent extraction
	extractor := NewUnifiedMetadataExtractor(m.log)
	ids := extractor.ExtractFromEventResponse(event)

	campaignID = ids.CampaignID
	userID = ids.UserID

	// Fallback: use default campaign
	if campaignID == "" {
		campaignID = "0" // Default for system campaign
	}

	return campaignID, userID
}

// generateStreamingEventID creates a unique ID for streaming events.
func generateStreamingEventID() string {
	return fmt.Sprintf("stream:%d", time.Now().UnixNano())
}

// Event subscription management for campaign events.
var (
	campaignEventSubscribers   = make(map[string][]chan *nexusv1.EventResponse)
	campaignEventSubscribersMu sync.RWMutex
)

// subscribeToCampaignEvents subscribes to campaign events.
func (m *CampaignStateManager) subscribeToCampaignEvents(campaignID string, eventChan chan *nexusv1.EventResponse) {
	campaignEventSubscribersMu.Lock()
	defer campaignEventSubscribersMu.Unlock()

	if campaignEventSubscribers[campaignID] == nil {
		campaignEventSubscribers[campaignID] = make([]chan *nexusv1.EventResponse, 0)
	}
	campaignEventSubscribers[campaignID] = append(campaignEventSubscribers[campaignID], eventChan)
}

// unsubscribeFromCampaignEvents unsubscribes from campaign events.
func (m *CampaignStateManager) unsubscribeFromCampaignEvents(campaignID string, eventChan chan *nexusv1.EventResponse) {
	campaignEventSubscribersMu.Lock()
	defer campaignEventSubscribersMu.Unlock()

	if subscribers, ok := campaignEventSubscribers[campaignID]; ok {
		for i, ch := range subscribers {
			if ch == eventChan {
				campaignEventSubscribers[campaignID] = append(subscribers[:i], subscribers[i+1:]...)
				break
			}
		}
	}
}

// broadcastCampaignEvent broadcasts an event to all subscribers.
func (m *CampaignStateManager) broadcastCampaignEvent(campaignID string, event *nexusv1.EventResponse) {
	campaignEventSubscribersMu.RLock()
	defer campaignEventSubscribersMu.RUnlock()

	if subscribers, ok := campaignEventSubscribers[campaignID]; ok {
		for _, ch := range subscribers {
			select {
			case ch <- event:
			default:
				// Channel is full, skip this subscriber
				m.log.Warn("Campaign event subscriber channel full, skipping",
					zap.String("campaign_id", campaignID))
			}
		}
	}
}
