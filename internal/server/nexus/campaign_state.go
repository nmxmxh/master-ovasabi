package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	campaignrepo "github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	meta "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// CampaignState holds the state for a single campaign ("app").
type CampaignState struct {
	CampaignID  string
	State       map[string]any
	LastUpdated time.Time
	Subscribers sync.Map // userID -> chan *nexusv1.EventResponse
}

// CampaignStateManager manages all campaign states and event loops.
type CampaignStateManager struct {
	log         *zap.Logger
	campaigns   sync.Map                           // campaignID -> *CampaignState
	feedbackBus func(event *nexusv1.EventResponse) // callback to Nexus event bus
	repo        *campaignrepo.Repository           // campaign DB repository/service
	// Deduplication tracking
	processedEvents sync.Map // eventID -> time.Time
	eventMutex      sync.RWMutex
	// Event counter for uniqueness
	eventCounter int64
	counterMutex sync.Mutex
}

// safeGo runs a function in a goroutine and recovers from panics, logging them.
func (m *CampaignStateManager) safeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.log.Error("panic in goroutine", zap.Any("recover", r))
			}
		}()
		fn()
	}()
}

// isEventProcessed checks if an event has been processed recently and marks it as processed
func (m *CampaignStateManager) isEventProcessed(eventID string) bool {
	m.eventMutex.Lock()
	defer m.eventMutex.Unlock()

	now := time.Now()

	// Clean up old entries (older than 30 seconds)
	m.processedEvents.Range(func(key, value interface{}) bool {
		if lastSeen, ok := value.(time.Time); ok {
			if now.Sub(lastSeen) > 30*time.Second {
				m.processedEvents.Delete(key)
			}
		}
		return true
	})

	// Check if this event was processed recently (within 2 seconds)
	if lastSeen, exists := m.processedEvents.Load(eventID); exists {
		if now.Sub(lastSeen.(time.Time)) < 2*time.Second {
			m.log.Debug("[CampaignState] Skipping duplicate event",
				zap.String("event_id", eventID))
			return true
		}
	}

	// Mark this event as processed
	m.processedEvents.Store(eventID, now)
	return false
}

// PrepareStateForUser returns a decorated copy of campaign state for a given user/client type.
func (m *CampaignStateManager) PrepareStateForUser(campaignID, userID string) map[string]any {
	state := m.GetState(campaignID)
	stateCopy := make(map[string]any, len(state))
	for k, v := range state {
		stateCopy[k] = v
	}
	if userID == "godot" {
		stateCopy["entity_type"] = "backend"
		stateCopy["client_type"] = "godot"
		stateCopy["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
	}
	return stateCopy
}

// HandleEvent is a generic event handler for campaign-related events (to be called from the Nexus event bus).
func (m *CampaignStateManager) HandleEvent(ctx context.Context, event *nexusv1.EventRequest) {
	campaignID, userID := m.extractCampaignAndUserID(event)
	// Canonical event mutation: only process events ending in ':requested' or ':started'
	if strings.HasSuffix(event.EventType, ":requested") || strings.HasSuffix(event.EventType, ":started") {
		switch event.EventType {
		case "campaign:list:v1:requested":
			m.handleCampaignList(ctx, event)
		case "campaign:update:v1:requested":
			m.handleCampaignUpdate(ctx, event)
		case "campaign:switch:v1:requested":
			m.handleCampaignSwitch(ctx, event)
		case "campaign:feature:v1:requested":
			m.handleFeatureUpdate(ctx, event)
		case "campaign:config:v1:requested":
			m.handleConfigUpdate(ctx, event)
		case "campaign:state:v1:requested":
			// Processing campaign state request
			state := m.PrepareStateForUser(campaignID, userID)
			structData := meta.NewStructFromMap(state, nil)
			eventType := "campaign:state:v1:success"
			// Extract or create correlation ID - preserve original correlation ID
			var correlationID string
			if event.Metadata != nil && event.Metadata.GlobalContext != nil {
				correlationID = event.Metadata.GlobalContext.CorrelationId
			}
			if correlationID == "" && event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
				if v, ok := event.Metadata.ServiceSpecific.Fields["correlation_id"]; ok && v != nil {
					correlationID = v.GetStringValue()
				}
			}
			if correlationID == "" && event.Payload != nil && event.Payload.Data != nil {
				if v, ok := event.Payload.Data.Fields["correlationId"]; ok && v != nil {
					correlationID = v.GetStringValue()
				}
			}

			// Debug logging for correlation ID extraction
			m.log.Info("Campaign state correlation ID extraction",
				zap.String("extracted_correlation_id", correlationID),
				zap.String("event_type", event.EventType),
				zap.String("user_id", userID),
				zap.String("campaign_id", campaignID),
				zap.Bool("has_global_context", event.Metadata != nil && event.Metadata.GlobalContext != nil),
				zap.Bool("has_service_specific", event.Metadata != nil && event.Metadata.ServiceSpecific != nil),
				zap.Bool("has_payload_data", event.Payload != nil && event.Payload.Data != nil))
			// Only generate a new correlation ID if absolutely necessary (should not happen in normal flow)
			if correlationID == "" {
				m.log.Warn("No correlation ID found in campaign state request, generating fallback",
					zap.String("event_type", event.EventType),
					zap.String("user_id", userID),
					zap.String("campaign_id", campaignID))
				correlationID = "corr_" + fmt.Sprintf("%d", time.Now().UnixNano())
			}

			// Add routing information to the state
			stateWithRouting := make(map[string]any, len(state)+3)
			maps.Copy(stateWithRouting, state)
			stateWithRouting["user_id"] = userID
			stateWithRouting["campaign_id"] = campaignID
			stateWithRouting["correlationId"] = correlationID // Include correlation ID for request/response matching
			structData = meta.NewStructFromMap(stateWithRouting, nil)
			if userID == "godot" {
				eventType = "campaign:state:v1:godot_update"
				// Optionally: broadcast to Godot stream
				cs := m.GetOrCreateState(campaignID)
				godotStreamKey := "godot_stream:" + campaignID
				var godotStream chan *nexusv1.EventResponse
				val, ok := cs.Subscribers.Load(godotStreamKey)
				if !ok {
					godotStream = make(chan *nexusv1.EventResponse, 128)
					cs.Subscribers.Store(godotStreamKey, godotStream)
				} else {
					godotStream, _ = val.(chan *nexusv1.EventResponse)
				}
				eventResp := &nexusv1.EventResponse{
					Success:   true,
					EventId:   uuid.New().String(), // Generate proper EventId
					EventType: eventType,
					Message:   "godot_state_update",
					Metadata:  event.Metadata,
					Payload:   &commonpb.Payload{Data: structData},
				}
				select {
				case godotStream <- eventResp:
				default:
					m.log.Warn("Godot campaign state stream full, dropping update", zap.String("campaign_id", campaignID))
				}
			}
			// Always emit to feedback bus for request/response
			eventResp := &nexusv1.EventResponse{
				Success:   true,
				EventId:   uuid.New().String(), // Generate proper EventId
				EventType: eventType,
				Message:   "state_init",
				Metadata:  event.Metadata,
				Payload:   &commonpb.Payload{Data: structData},
			}
			// Sending event via feedback bus

			m.feedbackBus(eventResp)
		}
	} else {
		// Ignoring non-mutation event type
	}
}

// NewCampaignStateManager creates a new manager with a feedback bus callback and campaign repository.
func NewCampaignStateManager(log *zap.Logger, feedbackBus func(event *nexusv1.EventResponse), repo *campaignrepo.Repository) *CampaignStateManager {
	m := &CampaignStateManager{
		log:         log,
		feedbackBus: feedbackBus,
		repo:        repo,
	}
	// Always load the default campaign first - this is the initial campaign
	if err := m.LoadDefaultCampaign("start/default_campaign.json"); err != nil {
		log.Error("Failed to load default campaign from start/default_campaign.json", zap.Error(err))
		// This is critical - the default campaign must exist
		panic("Cannot start without default campaign")
	}

	// Campaigns will be seeded on-demand when needed

	// Optionally preload all campaigns from DB
	if repo != nil {
		if err := m.LoadAllCampaignsFromDB(); err != nil {
			log.Warn("Failed to preload campaigns from DB", zap.Error(err))
		}
	}
	return m
}

// GetDefaultCampaign returns the default campaign, ensuring it always exists
func (m *CampaignStateManager) GetDefaultCampaign() *CampaignState {
	// Try to get from memory first
	if val, ok := m.campaigns.Load("ovasabi_website"); ok {
		if cs, ok := val.(*CampaignState); ok {
			return cs
		}
	}

	// If not found, try to load from config file
	if err := m.LoadDefaultCampaign("start/default_campaign.json"); err != nil {
		m.log.Error("Failed to load default campaign from start/default_campaign.json", zap.Error(err))
		return nil
	}

	// Return the default campaign
	if val, ok := m.campaigns.Load("ovasabi_website"); ok {
		if cs, ok := val.(*CampaignState); ok {
			return cs
		}
	}

	// This should never happen since we panic on startup if default campaign fails to load
	m.log.Error("Default campaign not found - this should not happen")
	return nil
}

// LoadAllCampaignsFromDB loads all campaigns from the DB and populates the state map.
func (m *CampaignStateManager) LoadAllCampaignsFromDB() error {
	if m.repo == nil {
		m.log.Warn("Repository not available, skipping database campaign load")
		return nil
	}
	campaigns, err := m.repo.List(context.Background(), 1000, 0)
	if err != nil {
		m.log.Error("Failed to load campaigns from database", zap.Error(err))
		return err
	}

	// Loaded campaigns from database
	for _, c := range campaigns {
		state := make(map[string]any)
		// Flatten all relevant fields from Metadata
		if c.Metadata != nil {
			if features := c.Metadata.GetFeatures(); features != nil {
				state["features"] = features
			}
			if tags := c.Metadata.GetTags(); tags != nil {
				state["tags"] = tags
			}
			if s := c.Metadata.GetScheduling(); s != nil && s.Fields != nil {
				for k, v := range s.Fields {
					state[k] = v.AsInterface()
				}
			}
			if a := c.Metadata.GetAudit(); a != nil && a.Fields != nil {
				for k, v := range a.Fields {
					state[k] = v.AsInterface()
				}
			}
			if v := c.Metadata.GetVersioning(); v != nil && v.Fields != nil {
				for k, v2 := range v.Fields {
					state[k] = v2.AsInterface()
				}
			}
			if cr := c.Metadata.GetCustomRules(); cr != nil && cr.Fields != nil {
				for k, v := range cr.Fields {
					state[k] = v.AsInterface()
				}
			}
			if ss := c.Metadata.GetServiceSpecific(); ss != nil && ss.Fields != nil {
				if campaignField, ok := ss.Fields["campaign"]; ok && campaignField != nil {
					if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
						maps.Copy(state, campaignStruct.AsMap())
					}
				}
			}
		}
		cs := &CampaignState{
			CampaignID:  c.Slug,
			State:       state,
			LastUpdated: time.Now(),
		}
		cs.Subscribers = sync.Map{}
		m.campaigns.Store(c.Slug, cs)
		// Loaded campaign from DB into state manager
	}
	return nil
}

// LoadDefaultCampaign loads the default campaign JSON and initializes its state.
func (m *CampaignStateManager) LoadDefaultCampaign(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	var data map[string]any
	dec := json.NewDecoder(file)
	if err := dec.Decode(&data); err != nil {
		return err
	}
	campaignIDVal, campaignIDOk := data["slug"]
	var campaignID string
	if campaignIDOk {
		if s, ok := campaignIDVal.(string); ok {
			campaignID = s
		} else {
			m.log.Warn("campaignID type assertion failed in LoadDefaultCampaign", zap.Any("campaignIDVal", campaignIDVal))
		}
	}
	if campaignID == "" {
		campaignID = "0"
	}
	state := make(map[string]any)
	for k, v := range data {
		if k != "service_specific" {
			state[k] = v
		}
	}
	if ss, ok := data["service_specific"].(map[string]any); ok {
		if campaignFields, ok := ss["campaign"].(map[string]any); ok {
			maps.Copy(state, campaignFields)
		}
	}
	cs := &CampaignState{
		CampaignID:  campaignID,
		State:       state,
		LastUpdated: time.Now(),
	}
	cs.Subscribers = sync.Map{}
	m.campaigns.Store(campaignID, cs)
	// Loaded default campaign into state manager
	return nil
}

// GetOrCreateState returns the state for a campaign, creating it if needed.
func (m *CampaignStateManager) GetOrCreateState(campaignID string) *CampaignState {
	val, ok := m.campaigns.Load(campaignID)
	if ok {
		cs, ok2 := val.(*CampaignState)
		if !ok2 {
			m.log.Warn("Type assertion to *CampaignState failed in GetOrCreateState", zap.Any("val", val))
			return nil
		}
		return cs
	}
	cs := &CampaignState{
		CampaignID:  campaignID,
		State:       make(map[string]any),
		LastUpdated: time.Now(),
	}
	cs.Subscribers = sync.Map{}
	m.campaigns.Store(campaignID, cs)
	return cs
}

// updateStateWithoutEvent updates the campaign state without emitting an event.
// This is used when the caller wants to send their own specific event.
func (m *CampaignStateManager) updateStateWithoutEvent(campaignID, userID string, update map[string]any, metadata *commonpb.Metadata) {
	// Generate unique event ID using counter to prevent collisions
	m.counterMutex.Lock()
	m.eventCounter++
	counter := m.eventCounter
	m.counterMutex.Unlock()

	eventID := fmt.Sprintf("state_update:%s:%s:%d:%d", campaignID, userID, time.Now().UnixNano(), counter)

	// Check for duplicate events atomically
	if m.isEventProcessed(eventID) {
		// Skipping duplicate state update
		return
	}

	cs := m.GetOrCreateState(campaignID)

	// Validate input parameters
	if campaignID == "" {
		m.log.Warn("updateStateWithoutEvent called with empty campaignID")
		return
	}
	if len(update) == 0 {
		m.log.Debug("updateStateWithoutEvent called with empty update, skipping")
		return
	}

	// Use metadata pkg to flatten and learn campaign state from metadata
	if metadata != nil {
		if metadata.ServiceSpecific != nil && metadata.ServiceSpecific.Fields != nil {
			if campaignField, ok := metadata.ServiceSpecific.Fields["campaign"]; ok && campaignField != nil {
				if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
					maps.Copy(cs.State, campaignStruct.AsMap())
				}
			}
		}
	}

	// Apply updates with validation
	originalState := make(map[string]any, len(cs.State))
	maps.Copy(originalState, cs.State)
	maps.Copy(cs.State, update)
	cs.LastUpdated = time.Now()

	// Count subscribers for logging (but don't send to them individually to avoid duplicates)
	subscriberCount := 0
	cs.Subscribers.Range(func(key, value interface{}) bool {
		subscriberCount++
		return true
	})

	// Campaign state updated (no event sent)
	m.log.Debug("Campaign state updated without event",
		zap.String("campaign_id", campaignID),
		zap.String("user_id", userID),
		zap.Int("subscriber_count", subscriberCount))
}

// UpdateState updates the campaign state and emits a real-time feedback event.
func (m *CampaignStateManager) UpdateState(campaignID, userID string, update map[string]any, metadata *commonpb.Metadata) {
	// Generate unique event ID using counter to prevent collisions
	m.counterMutex.Lock()
	m.eventCounter++
	counter := m.eventCounter
	m.counterMutex.Unlock()

	eventID := fmt.Sprintf("state_update:%s:%s:%d:%d", campaignID, userID, time.Now().UnixNano(), counter)

	// Check for duplicate events atomically
	if m.isEventProcessed(eventID) {
		// Skipping duplicate state update
		return
	}

	cs := m.GetOrCreateState(campaignID)

	// Validate input parameters
	if campaignID == "" {
		m.log.Warn("UpdateState called with empty campaignID")
		return
	}
	if len(update) == 0 {
		m.log.Debug("UpdateState called with empty update, skipping")
		return
	}

	// Use metadata pkg to flatten and learn campaign state from metadata
	if metadata != nil {
		if metadata.ServiceSpecific != nil && metadata.ServiceSpecific.Fields != nil {
			if campaignField, ok := metadata.ServiceSpecific.Fields["campaign"]; ok && campaignField != nil {
				if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
					maps.Copy(cs.State, campaignStruct.AsMap())
				}
			}
		}
	}

	// Apply updates with validation
	originalState := make(map[string]any, len(cs.State))
	maps.Copy(originalState, cs.State)
	maps.Copy(cs.State, update)
	cs.LastUpdated = time.Now()

	// Use the same event ID for consistency
	stateWithRouting := make(map[string]any, len(cs.State)+2)
	maps.Copy(stateWithRouting, cs.State)
	stateWithRouting["user_id"] = userID
	stateWithRouting["campaign_id"] = campaignID

	event := &nexusv1.EventResponse{
		Success:   true,
		EventId:   uuid.New().String(), // Generate proper EventId
		EventType: "campaign:state:v1:success",
		Message:   "state_updated",
		Metadata:  metadata,
		Payload: &commonpb.Payload{
			Data: meta.NewStructFromMap(stateWithRouting, nil),
		},
	}

	// Send event only once through feedback bus (which handles Redis publishing)
	if m.feedbackBus != nil {
		m.safeGo(func() {
			defer func() {
				if r := recover(); r != nil {
					m.log.Error("panic in feedback bus", zap.Any("recover", r), zap.String("campaign_id", campaignID))
				}
			}()
			// Debug: Log all events being sent through feedbackBus
			m.log.Info("[FEEDBACKBUS] Sending event",
				zap.String("event_type", event.EventType),
				zap.String("event_id", event.EventId),
				zap.Bool("success", event.Success),
				zap.String("message", event.Message))

			m.feedbackBus(event)
		})
	}

	// Count subscribers for logging (but don't send to them individually to avoid duplicates)
	subscriberCount := 0
	cs.Subscribers.Range(func(key, value interface{}) bool {
		subscriberCount++
		return true
	})

	// Campaign state updated
}

// Subscribe adds a user to the campaign's real-time feedback channel.
func (m *CampaignStateManager) Subscribe(campaignID, userID string) <-chan *nexusv1.EventResponse {
	if campaignID == "" || userID == "" {
		m.log.Warn("Subscribe called with empty campaignID or userID", zap.String("campaign_id", campaignID), zap.String("user_id", userID))
		return nil
	}

	cs := m.GetOrCreateState(campaignID)

	// Check if user is already subscribed and clean up old channel
	if existingCh, ok := cs.Subscribers.Load(userID); ok {
		if ch, ok := existingCh.(chan *nexusv1.EventResponse); ok {
			m.log.Debug("User already subscribed, cleaning up old channel", zap.String("campaign_id", campaignID), zap.String("user_id", userID))
			close(ch)
		}
	}

	// Create new channel with larger buffer for better performance
	ch := make(chan *nexusv1.EventResponse, 32)
	cs.Subscribers.Store(userID, ch)

	m.log.Debug("User subscribed to campaign", zap.String("campaign_id", campaignID), zap.String("user_id", userID))
	return ch
}

// Unsubscribe removes a user from the campaign's feedback channel.
func (m *CampaignStateManager) Unsubscribe(campaignID, userID string) {
	if campaignID == "" || userID == "" {
		m.log.Warn("Unsubscribe called with empty campaignID or userID", zap.String("campaign_id", campaignID), zap.String("user_id", userID))
		return
	}

	cs := m.GetOrCreateState(campaignID)
	if chVal, ok := cs.Subscribers.Load(userID); ok {
		if ch, ok2 := chVal.(chan *nexusv1.EventResponse); ok2 {
			// Drain any remaining messages before closing
			go func() {
				defer func() {
					if r := recover(); r != nil {
						m.log.Debug("Panic while draining channel", zap.Any("recover", r))
					}
				}()
				for range ch {
					// Drain channel
				}
			}()
			close(ch)
		}
		cs.Subscribers.Delete(userID)
		m.log.Debug("User unsubscribed from campaign", zap.String("campaign_id", campaignID), zap.String("user_id", userID))
	}
}

// GetState returns a copy of the current state for a campaign.
func (m *CampaignStateManager) GetState(campaignID string) map[string]any {
	cs := m.GetOrCreateState(campaignID)
	stateCopy := make(map[string]any, len(cs.State))
	for k, v := range cs.State {
		stateCopy[k] = v
	}
	return stateCopy
}

// handleCampaignList processes campaign list requests and returns all available campaigns.
func (m *CampaignStateManager) handleCampaignList(ctx context.Context, event *nexusv1.EventRequest) {
	campaignID, userID := m.extractCampaignAndUserID(event)

	// Generate event ID for deduplication
	eventID := "campaign_list:" + userID
	if event.Metadata != nil && event.Metadata.GlobalContext != nil {
		eventID = "campaign_list:" + userID + ":" + event.Metadata.GlobalContext.CorrelationId
	}

	// Check for duplicate events
	if m.isEventProcessed(eventID) {
		m.log.Debug("[CampaignState] Skipping duplicate campaign list request",
			zap.String("event_id", eventID),
			zap.String("user_id", userID))
		return
	}

	// Log the type and structure of incoming metadata for debugging
	// m.log.Info("CampaignList: Received event metadata",
	// 	zap.Any("metadata", event.Metadata),
	// 	zap.Bool("metadata_nil", event.Metadata == nil),
	// 	zap.Bool("service_specific_nil", event.Metadata != nil && event.Metadata.ServiceSpecific == nil))

	// Removed unused metadata field processing loop

	var payload struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}

	// Extract pagination parameters from payload
	if event.Payload != nil && event.Payload.Data != nil {
		payloadMap := event.Payload.Data.AsMap()
		if limit, ok := payloadMap["limit"].(float64); ok {
			payload.Limit = int(limit)
		}
		if offset, ok := payloadMap["offset"].(float64); ok {
			payload.Offset = int(offset)
		}
	}

	// Set default pagination if not provided
	if payload.Limit <= 0 {
		payload.Limit = 50
	}

	// Campaign list processing log removed for performance
	// Only log errors and critical information

	// Use a map to deduplicate campaigns by slug
	campaignMap := make(map[string]map[string]any)

	// Only seed from config if no campaigns exist in database
	if err := m.EnsureCampaignsSeeded(); err != nil {
		m.log.Warn("Failed to ensure campaigns are seeded", zap.Error(err))
	}

	// If repository is available, fetch from database first
	if m.repo != nil {
		if dbCampaigns, err := m.repo.List(ctx, payload.Limit, payload.Offset); err == nil {
			for _, c := range dbCampaigns {
				// Use slug as the key for deduplication
				slug := c.Slug
				if slug == "" {
					slug = fmt.Sprintf("%v", c.ID) // Fallback to ID if no slug
				}

				campaignData := map[string]any{
					"id":    slug, // Use slug as ID for consistency
					"slug":  slug,
					"title": c.Title,
					"name":  c.Slug,
				}

				// Add metadata if available
				if c.Metadata != nil {
					if features := c.Metadata.GetFeatures(); features != nil {
						campaignData["features"] = features
					}
					if tags := c.Metadata.GetTags(); tags != nil {
						campaignData["tags"] = tags
					}
					// Add service-specific campaign data
					if ss := c.Metadata.GetServiceSpecific(); ss != nil && ss.Fields != nil {
						if campaignField, ok := ss.Fields["campaign"]; ok && campaignField != nil {
							if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
								for k, v := range campaignStruct.AsMap() {
									campaignData[k] = v
								}
							}
						}
					}
				}

				// Only add if not already present (database takes precedence)
				if _, exists := campaignMap[slug]; !exists {
					campaignMap[slug] = campaignData
				}
			}
		} else {
			m.log.Error("Failed to fetch campaigns from database", zap.Error(err), zap.String("error_type", fmt.Sprintf("%T", err)))
		}
	}

	// Add campaigns from memory state (includes config data) - only if not already present
	m.campaigns.Range(func(key, value interface{}) bool {
		campaignID, ok := key.(string)
		if !ok {
			m.log.Warn("Type assertion to string failed for campaignID in handleCampaignList", zap.Any("key", key))
			return true
		}
		cs, ok2 := value.(*CampaignState)
		if !ok2 {
			m.log.Warn("Type assertion to *CampaignState failed in handleCampaignList", zap.Any("value", value))
			return true
		}

		// Only add if not already present from database
		if _, exists := campaignMap[campaignID]; !exists {
			campaignData := map[string]any{
				"id":   campaignID,
				"slug": campaignID,
				"name": campaignID,
			}

			// Add state data (includes ui_components from config)
			for k, v := range cs.State {
				campaignData[k] = v
			}

			campaignMap[campaignID] = campaignData
		}
		return true
	})

	// Always ensure the default campaign is included
	if _, exists := campaignMap["ovasabi_website"]; !exists {
		m.log.Info("Adding default campaign to campaign list")
		if defaultState, ok := m.campaigns.Load("ovasabi_website"); ok {
			if cs, ok := defaultState.(*CampaignState); ok {
				campaignData := map[string]any{
					"id":   "ovasabi_website",
					"slug": "ovasabi_website",
					"name": "Ovasabi Website",
				}
				// Add all state data including ui_components
				for k, v := range cs.State {
					campaignData[k] = v
				}
				campaignMap["ovasabi_website"] = campaignData
			}
		} else {
			// Fallback: try to load default campaign from file
			m.log.Warn("Default campaign not found in memory, attempting to load from file")
			if err := m.LoadDefaultCampaign("start/default_campaign.json"); err != nil {
				m.log.Error("Failed to load default campaign from file", zap.Error(err))
			} else {
				// Try again to get from memory after loading
				if defaultState, ok := m.campaigns.Load("ovasabi_website"); ok {
					if cs, ok := defaultState.(*CampaignState); ok {
						campaignData := map[string]any{
							"id":   "ovasabi_website",
							"slug": "ovasabi_website",
							"name": "Ovasabi Website",
						}
						// Add all state data including ui_components
						for k, v := range cs.State {
							campaignData[k] = v
						}
						campaignMap["ovasabi_website"] = campaignData
					}
				}
			}
		}
	}

	// Convert campaignMap to slice and ensure all campaigns have required fields
	var campaigns []map[string]any
	for slug, campaignData := range campaignMap {
		// Ensure required fields are present
		if campaignData["title"] == nil || campaignData["title"] == "" {
			// Use slug as title if missing
			campaignData["title"] = strings.ReplaceAll(slug, "_", " ")
			// Capitalize first letter
			if title, ok := campaignData["title"].(string); ok && len(title) > 0 {
				campaignData["title"] = strings.ToUpper(title[:1]) + title[1:]
			}
		}
		if campaignData["name"] == nil || campaignData["name"] == "" {
			campaignData["name"] = slug
		}
		if campaignData["slug"] == nil || campaignData["slug"] == "" {
			campaignData["slug"] = slug
		}
		if campaignData["id"] == nil || campaignData["id"] == "" {
			campaignData["id"] = slug
		}

		campaigns = append(campaigns, campaignData)
	}

	// Extract correlationId from metadata or payload - preserve original correlation ID
	var correlationID string
	if event.Metadata != nil && event.Metadata.GlobalContext != nil {
		correlationID = event.Metadata.GlobalContext.CorrelationId
	}
	if correlationID == "" && event.Metadata != nil && event.Metadata.ServiceSpecific != nil && event.Metadata.ServiceSpecific.Fields != nil {
		if v, ok := event.Metadata.ServiceSpecific.Fields["correlation_id"]; ok && v != nil {
			correlationID = v.GetStringValue()
		}
	}
	if correlationID == "" && event.Payload != nil && event.Payload.Data != nil {
		if v, ok := event.Payload.Data.Fields["correlationId"]; ok && v != nil {
			correlationID = v.GetStringValue()
		}
	}

	// Debug logging for correlation ID extraction
	m.log.Info("Campaign list correlation ID extraction",
		zap.String("extracted_correlation_id", correlationID),
		zap.String("event_type", event.EventType),
		zap.String("user_id", userID),
		zap.String("campaign_id", campaignID),
		zap.Bool("has_global_context", event.Metadata != nil && event.Metadata.GlobalContext != nil),
		zap.Bool("has_service_specific", event.Metadata != nil && event.Metadata.ServiceSpecific != nil),
		zap.Bool("has_payload_data", event.Payload != nil && event.Payload.Data != nil))
	// Only generate a new correlation ID if absolutely necessary (should not happen in normal flow)
	if correlationID == "" {
		m.log.Warn("No correlation ID found in campaign list request, generating fallback",
			zap.String("event_type", event.EventType),
			zap.String("user_id", userID),
			zap.String("campaign_id", campaignID))
		correlationID = "corr_" + fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Convert campaigns to proper JSON-serializable format
	jsonCampaigns := make([]map[string]interface{}, len(campaigns))
	for i, campaign := range campaigns {
		jsonCampaign := make(map[string]interface{})
		for k, v := range campaign {
			jsonCampaign[k] = v
		}
		jsonCampaigns[i] = jsonCampaign
	}

	// Create simplified response first to test serialization
	responsePayload := map[string]any{
		"campaigns":     jsonCampaigns,
		"total":         len(campaigns),
		"limit":         payload.Limit,
		"offset":        payload.Offset,
		"user_id":       userID,
		"campaign_id":   campaignID,    // Add campaign_id for proper routing
		"correlationId": correlationID, // Include correlation ID for request/response matching
		"source":        "nexus",
	}

	// Validate campaign data structure
	m.validateCampaignData(campaigns)

	// Campaign data debug logging removed for performance
	// Only log errors and critical information

	// Serialization debug logs removed for performance
	// Only log serialization errors if they occur

	// Use a more robust serialization approach for campaign data
	structData := m.serializeCampaignResponse(responsePayload)

	// Serialization debug logging removed for performance

	// Update event ID with correlation ID if available
	if correlationID != "" {
		eventID = "campaign_list:" + userID + ":" + correlationID
	} else {
		eventID = "campaign_list:" + userID + ":" + time.Now().UTC().Format("20060102T150405.000Z")
	}

	response := &nexusv1.EventResponse{
		Success:   true,
		EventId:   uuid.New().String(), // Generate proper EventId
		EventType: "campaign:list:v1:success",
		Message:   "campaign_list_retrieved",
		Metadata:  event.Metadata,
		Payload: &commonpb.Payload{
			Data: structData,
		},
	}

	m.log.Info("Sending campaign list response",
		zap.String("user_id", userID),
		zap.Int("campaign_count", len(campaigns)),
		zap.String("event_id", response.EventId),
		zap.String("response_event_type", response.EventType),
		zap.Bool("success", response.Success))

	// Debug: Log all events being sent through feedbackBus
	m.log.Info("[FEEDBACKBUS] Sending event",
		zap.String("event_type", response.EventType),
		zap.String("event_id", response.EventId),
		zap.Bool("success", response.Success),
		zap.String("message", response.Message))

	m.feedbackBus(response)
}

// EnsureCampaignsSeeded ensures campaigns are properly seeded from config if database is empty
func (m *CampaignStateManager) EnsureCampaignsSeeded() error {
	if m.repo == nil {
		m.log.Warn("Repository not available, cannot seed campaigns")
		return nil
	}

	// Check if campaigns exist in database
	campaigns, err := m.repo.List(context.Background(), 10, 0)
	if err != nil {
		m.log.Error("Failed to check existing campaigns", zap.Error(err))
		return err
	}

	if len(campaigns) == 0 {
		m.log.Info("No campaigns found in database, attempting to seed from config")
		return m.seedCampaignsFromConfig()
	}

	m.log.Info("Campaigns already exist in database", zap.Int("count", len(campaigns)))
	return nil
}

// seedCampaignsFromConfig seeds campaigns from the configuration file
func (m *CampaignStateManager) seedCampaignsFromConfig() error {
	m.log.Debug("Seeding campaigns from configuration file")

	// Load campaigns from config/campaign.json
	file, err := os.Open("config/campaign.json")
	if err != nil {
		m.log.Error("Failed to open campaign config file", zap.Error(err))
		return err
	}
	defer file.Close()

	var campaigns []map[string]any
	dec := json.NewDecoder(file)
	if err := dec.Decode(&campaigns); err != nil {
		m.log.Error("Failed to decode campaign config", zap.Error(err))
		return err
	}

	// Process each campaign from config
	for i, campaign := range campaigns {
		// Extract campaign ID and slug
		campaignID := fmt.Sprintf("%d", i+1) // Use index as ID
		if slug, ok := campaign["slug"].(string); ok && slug != "" {
			campaignID = slug
		}

		// Create campaign state with full data including ui_components
		state := make(map[string]any)
		for k, v := range campaign {
			state[k] = v
		}

		// Add service-specific data if it exists
		if metadata, ok := campaign["metadata"].(map[string]any); ok {
			if serviceSpecific, ok := metadata["service_specific"].(map[string]any); ok {
				if campaignData, ok := serviceSpecific["campaign"].(map[string]any); ok {
					// Merge service-specific campaign data into state
					for k, v := range campaignData {
						state[k] = v
					}
				}
			}
		}

		// Create campaign state
		cs := &CampaignState{
			CampaignID:  campaignID,
			State:       state,
			LastUpdated: time.Now(),
		}
		cs.Subscribers = sync.Map{}
		m.campaigns.Store(campaignID, cs)

		m.log.Debug("Seeded campaign from config",
			zap.String("campaign_id", campaignID),
			zap.String("title", state["title"].(string)),
			zap.Bool("has_ui_components", state["ui_components"] != nil))
	}

	m.log.Debug("Successfully seeded campaigns from configuration", zap.Int("count", len(campaigns)))
	return nil
}

// validateCampaignData validates the structure and completeness of campaign data
func (m *CampaignStateManager) validateCampaignData(campaigns []map[string]any) {
	requiredFields := []string{"id", "slug", "title", "name"}

	for i, campaign := range campaigns {
		missingFields := []string{}
		for _, field := range requiredFields {
			if _, exists := campaign[field]; !exists {
				missingFields = append(missingFields, field)
			}
		}

		if len(missingFields) > 0 {
			m.log.Warn("Campaign missing required fields",
				zap.Int("index", i),
				zap.Strings("missing_fields", missingFields),
				zap.Any("campaign", campaign))
		}

		// Check for empty or invalid data
		if slug, ok := campaign["slug"].(string); ok && slug == "" {
			m.log.Warn("Campaign has empty slug", zap.Int("index", i), zap.Any("campaign", campaign))
		}

		if title, ok := campaign["title"].(string); ok && title == "" {
			m.log.Warn("Campaign has empty title", zap.Int("index", i), zap.Any("campaign", campaign))
		}
	}

	// Log summary
	validCount := 0
	for _, campaign := range campaigns {
		hasAllFields := true
		for _, field := range requiredFields {
			if _, exists := campaign[field]; !exists {
				hasAllFields = false
				break
			}
		}
		if hasAllFields {
			validCount++
		}
	}

	m.log.Info("Campaign data validation complete",
		zap.Int("total_campaigns", len(campaigns)),
		zap.Int("valid_campaigns", validCount))
}

// handleCampaignUpdate processes direct campaign update requests.
func (m *CampaignStateManager) handleCampaignUpdate(ctx context.Context, event *nexusv1.EventRequest) {
	var payload struct {
		CampaignID string         `json:"campaign_id"`
		Updates    map[string]any `json:"updates"`
	}

	// Extract campaign ID and user ID from metadata
	campaignID, userID := m.extractCampaignAndUserID(event)

	// Try to unmarshal from payload first
	if event.Payload != nil && event.Payload.Data != nil {
		payloadMap := event.Payload.Data.AsMap()
		if cid, ok := payloadMap["campaignId"].(string); ok && cid != "" {
			payload.CampaignID = cid
		}
		if updates, ok := payloadMap["updates"].(map[string]any); ok {
			payload.Updates = updates
		}
	}

	// Fallback to metadata-derived campaign ID
	if payload.CampaignID == "" {
		payload.CampaignID = campaignID
	}

	if payload.CampaignID == "" {
		m.log.Error("Campaign update: missing campaign ID")
		return
	}

	if payload.Updates == nil {
		m.log.Error("Campaign update: missing updates")
		return
	}

	m.log.Info("Processing campaign update",
		zap.String("campaign_id", payload.CampaignID),
		zap.String("user_id", userID),
		zap.Any("updates", payload.Updates))

	// First, validate that the campaign exists before attempting to persist
	var persistErr error
	if m.repo != nil {
		// Use a separate context with timeout for database operations
		dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dbCancel()

		// Check if campaign exists first and get it for persistence
		campaign, err := m.repo.GetBySlug(dbCtx, payload.CampaignID)
		if err != nil {
			// If slug lookup fails, check if this might be a numeric ID
			// and suggest using the slug instead
			if _, parseErr := strconv.Atoi(payload.CampaignID); parseErr == nil {
				m.log.Error("Campaign ID appears to be numeric, but database expects slug",
					zap.String("campaign_id", payload.CampaignID),
					zap.String("user_id", userID),
					zap.Error(err),
					zap.String("suggestion", "Use campaign slug instead of numeric ID"))
			} else {
				m.log.Error("Campaign not found in database",
					zap.String("campaign_id", payload.CampaignID),
					zap.String("user_id", userID),
					zap.Error(err))
			}

			// Send failure event for non-existent campaign
			m.sendFailureEvent(payload.CampaignID, userID, "Campaign not found - use slug instead of ID", event.Metadata)
			return
		}

		// Campaign exists, proceed with persistence using the retrieved campaign
		persistErr = m.persistToDBSyncWithCampaign(dbCtx, campaign, payload.Updates)
		if persistErr != nil {
			m.log.Error("Failed to persist campaign update to database",
				zap.String("campaign_id", payload.CampaignID),
				zap.String("user_id", userID),
				zap.Error(persistErr))

			// Send failure event instead of success
			m.sendFailureEvent(payload.CampaignID, userID, "Database persistence failed", event.Metadata)
			return
		}
	}

	// Only update state and send success event if database persistence succeeded
	m.UpdateState(payload.CampaignID, userID, payload.Updates, event.Metadata)
}

// handleCampaignSwitch processes campaign switching requests.
func (m *CampaignStateManager) handleCampaignSwitch(ctx context.Context, event *nexusv1.EventRequest) {
	// Extract campaign and user IDs
	campaignID, userID := m.extractCampaignAndUserID(event)
	if campaignID == "" || userID == "" {
		m.log.Error("Missing campaign or user ID in campaign switch request")
		return
	}

	// Parse payload using the same pattern as handleCampaignUpdate
	var payload struct {
		CampaignID string                 `json:"campaignId"`
		Slug       string                 `json:"slug"`
		Updates    map[string]interface{} `json:"updates"`
	}

	// Try to unmarshal from payload first using AsMap() method
	if event.Payload != nil && event.Payload.Data != nil {
		payloadMap := event.Payload.Data.AsMap()
		if cid, ok := payloadMap["campaignId"].(string); ok && cid != "" {
			payload.CampaignID = cid
		}
		if slug, ok := payloadMap["slug"].(string); ok && slug != "" {
			payload.Slug = slug
		}
		if updates, ok := payloadMap["updates"].(map[string]any); ok {
			payload.Updates = updates
		}
	}

	// Fallback to metadata-derived campaign ID
	if payload.CampaignID == "" {
		payload.CampaignID = campaignID
	}

	if payload.CampaignID == "" {
		m.log.Error("Campaign switch: missing campaign ID")
		return
	}

	// Campaign switch processing log removed for performance
	// Only log errors and critical information

	// For campaign switching, we primarily update the user's active campaign
	// without necessarily persisting to database (this is more of a session state change)

	// Update the campaign state to reflect the switch
	updates := map[string]interface{}{
		"status":        "active",
		"last_switched": time.Now().UTC().Format(time.RFC3339),
		"switch_reason": "user_initiated",
	}

	// Merge any additional updates from the payload
	if payload.Updates != nil {
		for k, v := range payload.Updates {
			updates[k] = v
		}
	}

	// Update state without sending an event (we'll send our own switch success event)
	m.updateStateWithoutEvent(payload.CampaignID, userID, updates, event.Metadata)

	// Campaign switch success logging removed for performance
	// Only log errors and critical information

	// Get current campaign state to include in switch success response
	cs := m.GetOrCreateState(payload.CampaignID)
	stateWithRouting := make(map[string]any, len(cs.State)+2)
	maps.Copy(stateWithRouting, cs.State)
	stateWithRouting["user_id"] = userID
	stateWithRouting["campaign_id"] = payload.CampaignID

	// Generate unique event ID for switch success
	m.counterMutex.Lock()
	m.eventCounter++
	counter := m.eventCounter
	m.counterMutex.Unlock()

	switchEventID := fmt.Sprintf("campaign_switch:%s:%s:%d:%d", payload.CampaignID, userID, time.Now().UnixNano(), counter)

	// Send campaign switch success event
	switchSuccessEvent := &nexusv1.EventResponse{
		Success:   true,
		EventId:   uuid.New().String(), // Generate proper EventId
		EventType: "campaign:switch:v1:success",
		Message:   "campaign_switched",
		Metadata:  event.Metadata,
		Payload: &commonpb.Payload{
			Data: meta.NewStructFromMap(stateWithRouting, nil),
		},
	}

	// Send through feedback bus with proper sequencing
	if m.feedbackBus != nil {
		// Send immediately to ensure event is delivered before WebSocket close
		m.feedbackBus(switchSuccessEvent)

		// Log the successful switch
		m.log.Info("Campaign switch success event sent",
			zap.String("campaign_id", payload.CampaignID),
			zap.String("user_id", userID),
			zap.String("event_id", switchEventID))
	}

	m.log.Info("Campaign switch completed",
		zap.String("campaign_id", payload.CampaignID),
		zap.String("user_id", userID))
}

// handleFeatureUpdate processes feature-specific updates.
func (m *CampaignStateManager) handleFeatureUpdate(_ context.Context, event *nexusv1.EventRequest) {
	var payload struct {
		CampaignID string   `json:"campaign_id"`
		Features   []string `json:"features"`
		Action     string   `json:"action"` // "add", "remove", "set"
	}

	campaignID, userID := m.extractCampaignAndUserID(event)

	if event.Payload != nil && event.Payload.Data != nil {
		payloadMap := event.Payload.Data.AsMap()
		if cid, ok := payloadMap["campaignId"].(string); ok && cid != "" {
			payload.CampaignID = cid
		}
		if features, ok := payloadMap["features"].([]any); ok {
			for _, f := range features {
				if s, ok := f.(string); ok {
					payload.Features = append(payload.Features, s)
				}
			}
		}
		if action, ok := payloadMap["action"].(string); ok {
			payload.Action = action
		}
	}

	if payload.CampaignID == "" {
		payload.CampaignID = campaignID
	}

	if payload.CampaignID == "" || len(payload.Features) == 0 {
		m.log.Error("Feature update: missing campaign ID or features")
		return
	}

	cs := m.GetOrCreateState(payload.CampaignID)
	currentFeatures := []string{}

	// Get current features using type switch
	switch existing := cs.State["features"].(type) {
	case []string:
		currentFeatures = existing
	case []any:
		for _, f := range existing {
			if s, ok := f.(string); ok {
				currentFeatures = append(currentFeatures, s)
			}
		}
	}

	// Apply feature changes
	switch payload.Action {
	case "add":
		for _, newFeature := range payload.Features {
			found := false
			for _, existing := range currentFeatures {
				if existing == newFeature {
					found = true
					break
				}
			}
			if !found {
				currentFeatures = append(currentFeatures, newFeature)
			}
		}
	case "remove":
		filtered := []string{}
		for _, existing := range currentFeatures {
			shouldKeep := true
			for _, toRemove := range payload.Features {
				if existing == toRemove {
					shouldKeep = false
					break
				}
			}
			if shouldKeep {
				filtered = append(filtered, existing)
			}
		}
		currentFeatures = filtered
	case "set":
		currentFeatures = payload.Features
	default:
		m.log.Error("Feature update: invalid action", zap.String("action", payload.Action))
		return
	}

	updates := map[string]any{
		"features": currentFeatures,
	}

	m.log.Info("Processing feature update",
		zap.String("campaign_id", payload.CampaignID),
		zap.String("action", payload.Action),
		zap.Strings("features", payload.Features))

	m.UpdateState(payload.CampaignID, userID, updates, event.Metadata)
}

// handleConfigUpdate processes configuration updates (UI content, scripts, etc.)
func (m *CampaignStateManager) handleConfigUpdate(_ context.Context, event *nexusv1.EventRequest) {
	var payload struct {
		CampaignID string         `json:"campaign_id"`
		ConfigType string         `json:"config_type"` // "ui_content", "scripts", "communication"
		Config     map[string]any `json:"config"`
	}

	campaignID, userID := m.extractCampaignAndUserID(event)

	if event.Payload != nil && event.Payload.Data != nil {
		payloadMap := event.Payload.Data.AsMap()
		if cid, ok := payloadMap["campaignId"].(string); ok && cid != "" {
			payload.CampaignID = cid
		}
		if configType, ok := payloadMap["configType"].(string); ok {
			payload.ConfigType = configType
		}
		if config, ok := payloadMap["config"].(map[string]any); ok {
			payload.Config = config
		}
	}

	if payload.CampaignID == "" {
		payload.CampaignID = campaignID
	}

	if payload.CampaignID == "" || payload.ConfigType == "" || payload.Config == nil {
		m.log.Error("Config update: missing required fields")
		return
	}

	updates := map[string]any{
		payload.ConfigType: payload.Config,
	}

	m.log.Info("Processing config update",
		zap.String("campaign_id", payload.CampaignID),
		zap.String("config_type", payload.ConfigType))

	m.UpdateState(payload.CampaignID, userID, updates, event.Metadata)
}

// extractCampaignAndUserID extracts campaign and user IDs from event metadata using unified extractor.
func (m *CampaignStateManager) extractCampaignAndUserID(event *nexusv1.EventRequest) (campaignID, userID string) {
	m.log.Debug("[extractCampaignAndUserID] Starting extraction", zap.Any("metadata", event.Metadata))

	if event.Metadata == nil {
		m.log.Debug("[extractCampaignAndUserID] No metadata available")
		return "", ""
	}

	// Use unified metadata extractor for consistent extraction
	extractor := NewUnifiedMetadataExtractor(m.log)
	ids := extractor.ExtractFromEventRequest(context.Background(), event)

	campaignID = ids.CampaignID
	userID = ids.UserID

	m.log.Debug("[extractCampaignAndUserID] Final result", zap.String("campaign_id", campaignID), zap.String("user_id", userID))
	return campaignID, userID
}

// persistToDBSync synchronously persists campaign state changes to the database.
func (m *CampaignStateManager) persistToDBSync(ctx context.Context, campaignID string, updates map[string]any) error {
	// Get campaign from DB
	campaign, err := m.repo.GetBySlug(ctx, campaignID)
	if err != nil {
		m.log.Error("Failed to get campaign for persistence",
			zap.String("campaign_id", campaignID),
			zap.Error(err))
		return err
	}

	// Get current campaign state
	cs := m.GetOrCreateState(campaignID)
	// Merge updates into campaign state before persisting
	if updates != nil {
		maps.Copy(cs.State, updates)
	}

	// Update metadata with state changes
	if campaign.Metadata != nil && campaign.Metadata.ServiceSpecific != nil {
		// Merge current state into campaign metadata
		if campaignField, ok := campaign.Metadata.ServiceSpecific.Fields["campaign"]; ok && campaignField != nil {
			if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
				// Merge state into existing campaign metadata
				existingMap := campaignStruct.AsMap()
				maps.Copy(existingMap, cs.State)
				structData := meta.NewStructFromMap(existingMap, nil)
				campaign.Metadata.ServiceSpecific.Fields["campaign"] = &structpb.Value{
					Kind: &structpb.Value_StructValue{StructValue: structData},
				}
			}
		} else {
			// Create new campaign metadata
			structData := meta.NewStructFromMap(cs.State, nil)
			campaign.Metadata.ServiceSpecific.Fields["campaign"] = &structpb.Value{
				Kind: &structpb.Value_StructValue{StructValue: structData},
			}
		}
	}

	// Update in database
	if err := m.repo.Update(ctx, campaign); err != nil {
		m.log.Error("Failed to persist campaign state to database",
			zap.String("campaign_id", campaignID),
			zap.Error(err))
		return err
	}

	m.log.Info("Successfully persisted campaign state to database",
		zap.String("campaign_id", campaignID))
	return nil
}

// persistToDBSyncWithCampaign synchronously persists campaign state changes using a pre-retrieved campaign object
func (m *CampaignStateManager) persistToDBSyncWithCampaign(ctx context.Context, campaign *campaignrepo.Campaign, updates map[string]any) error {
	// Get current campaign state
	cs := m.GetOrCreateState(campaign.Slug)
	// Merge updates into campaign state before persisting
	if updates != nil {
		maps.Copy(cs.State, updates)
	}

	// Update metadata with state changes
	if campaign.Metadata != nil && campaign.Metadata.ServiceSpecific != nil {
		// Merge current state into campaign metadata
		if campaignField, ok := campaign.Metadata.ServiceSpecific.Fields["campaign"]; ok && campaignField != nil {
			if campaignStruct := campaignField.GetStructValue(); campaignStruct != nil {
				// Merge state into existing campaign metadata
				existingMap := campaignStruct.AsMap()
				maps.Copy(existingMap, cs.State)
				structData := meta.NewStructFromMap(existingMap, nil)
				campaign.Metadata.ServiceSpecific.Fields["campaign"] = &structpb.Value{
					Kind: &structpb.Value_StructValue{StructValue: structData},
				}
			}
		} else {
			// Create new campaign metadata
			structData := meta.NewStructFromMap(cs.State, nil)
			campaign.Metadata.ServiceSpecific.Fields["campaign"] = &structpb.Value{
				Kind: &structpb.Value_StructValue{StructValue: structData},
			}
		}
	}

	// Update in database
	if err := m.repo.Update(ctx, campaign); err != nil {
		m.log.Error("Failed to persist campaign state to database",
			zap.String("campaign_id", campaign.Slug),
			zap.Error(err))
		return err
	}

	m.log.Info("Successfully persisted campaign state to database",
		zap.String("campaign_id", campaign.Slug))
	return nil
}

// sendFailureEvent sends a failure event when database persistence fails
func (m *CampaignStateManager) sendFailureEvent(campaignID, userID, errorMessage string, metadata *commonpb.Metadata) {
	// Generate unique event ID for failure event
	// Create failure event
	stateWithRouting := make(map[string]any, 3)
	stateWithRouting["user_id"] = userID
	stateWithRouting["campaign_id"] = campaignID
	stateWithRouting["error"] = errorMessage

	event := &nexusv1.EventResponse{
		Success:   false,
		EventId:   uuid.New().String(), // Generate proper EventId
		EventType: "campaign:state:v1:failed",
		Message:   "state_update_failed",
		Metadata:  metadata,
		Payload: &commonpb.Payload{
			Data: meta.NewStructFromMap(stateWithRouting, nil),
		},
	}

	// Send failure event through feedback bus
	if m.feedbackBus != nil {
		m.safeGo(func() {
			defer func() {
				if r := recover(); r != nil {
					m.log.Error("panic in failure feedback bus", zap.Any("recover", r), zap.String("campaign_id", campaignID))
				}
			}()
			m.log.Info("[FEEDBACKBUS] Sending failure event",
				zap.String("event_type", event.EventType),
				zap.String("event_id", event.EventId),
				zap.Bool("success", event.Success),
				zap.String("message", event.Message))

			m.feedbackBus(event)
		})
	}
}

// persistToDB asynchronously persists campaign state changes to the database.
// This is kept for backward compatibility but now uses the sync version internally.
func (m *CampaignStateManager) persistToDB(ctx context.Context, campaignID string, updates map[string]any) {
	cancel := func() {}
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	}
	defer cancel()

	// Use the sync version internally
	if err := m.persistToDBSync(ctx, campaignID, updates); err != nil {
		m.log.Error("Failed to persist campaign state to database",
			zap.String("campaign_id", campaignID),
			zap.Error(err))
	}
}

// serializeCampaignResponse handles the serialization of campaign response data
// with special handling for complex nested structures
func (m *CampaignStateManager) serializeCampaignResponse(data map[string]interface{}) *structpb.Struct {
	// First try the standard approach
	structData := meta.NewStructFromMap(data, m.log)
	if len(structData.AsMap()) > 0 {
		m.log.Info("Standard serialization successful")
		return structData
	}

	// If standard approach fails, try a more conservative approach
	m.log.Warn("Standard serialization failed, trying conservative approach")

	// Create a simplified version with only essential fields
	simplifiedData := map[string]interface{}{
		"campaigns":     data["campaigns"],
		"total":         data["total"],
		"limit":         data["limit"],
		"offset":        data["offset"],
		"user_id":       data["user_id"],
		"campaign_id":   data["campaign_id"],
		"correlationId": data["correlationId"],
		"source":        data["source"],
	}

	simplifiedStruct := meta.NewStructFromMap(simplifiedData, m.log)
	if len(simplifiedStruct.AsMap()) > 0 {
		m.log.Info("Conservative serialization successful")
		return simplifiedStruct
	}

	// If all else fails, create a minimal response
	m.log.Error("All serialization approaches failed, creating minimal response")
	minimalData := map[string]interface{}{
		"campaigns": []interface{}{},
		"total":     0,
		"source":    "nexus",
	}

	return meta.NewStructFromMap(minimalData, m.log)
}

// GetCampaignArchitectureSummary returns a summary of campaign architecture and health for dashboards.
func (m *CampaignStateManager) GetCampaignArchitectureSummary() map[string]any {
	summary := map[string]any{}
	campaigns := []map[string]any{}
	totalSubscribers := 0
	totalCampaigns := 0

	m.campaigns.Range(func(key, value interface{}) bool {
		// Reference unused parameter 'key' for diagnostics
		_ = key
		cs, ok := value.(*CampaignState)
		if !ok {
			m.log.Warn("Type assertion to *CampaignState failed in GetCampaignArchitectureSummary", zap.Any("value", value))
			return true
		}
		campaignInfo := map[string]any{
			"campaign_id":  cs.CampaignID,
			"last_updated": cs.LastUpdated,
			"state_keys":   len(cs.State),
			"features":     cs.State["features"],
			"tags":         cs.State["tags"],
		}
		// Count subscribers
		subscribers := 0
		cs.Subscribers.Range(func(_, _ interface{}) bool {
			subscribers++
			return true
		})
		campaignInfo["subscribers"] = subscribers
		totalSubscribers += subscribers
		return true
	})

	summary["total_campaigns"] = totalCampaigns
	summary["total_subscribers"] = totalSubscribers
	summary["campaigns"] = campaigns

	return summary
}
