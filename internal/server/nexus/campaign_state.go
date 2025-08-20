package nexus

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"strings"
	"sync"
	"time"

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
		case "campaign:feature:v1:requested":
			m.handleFeatureUpdate(ctx, event)
		case "campaign:config:v1:requested":
			m.handleConfigUpdate(ctx, event)
		case "campaign:state:v1:request":
			state := m.PrepareStateForUser(campaignID, userID)
			structData := meta.NewStructFromMap(state, nil)
			eventType := "campaign:state:v1:success"
			// Extract or create correlation ID
			var correlationID string
			if event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
				if v, ok := event.Metadata.ServiceSpecific.Fields["correlation_id"]; ok && v != nil {
					correlationID = v.GetStringValue()
				}
			}
			if correlationID == "" && event.Payload != nil && event.Payload.Data != nil {
				if v, ok := event.Payload.Data.Fields["correlationId"]; ok && v != nil {
					correlationID = v.GetStringValue()
				}
			}
			if correlationID == "" {
				// Generate a new correlation ID if missing
				correlationID = "corrid:" + campaignID + ":" + userID + ":" + time.Now().UTC().Format("20060102T150405.000Z")
			}
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
					EventId:   correlationID,
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
				EventId:   correlationID,
				EventType: eventType,
				Message:   "state_init",
				Metadata:  event.Metadata,
				Payload:   &commonpb.Payload{Data: structData},
			}
			m.feedbackBus(eventResp)
		}
	} else {
		m.log.Debug("Ignoring non-mutation event type", zap.String("event_type", event.EventType))
	}
}

// NewCampaignStateManager creates a new manager with a feedback bus callback and campaign repository.
func NewCampaignStateManager(log *zap.Logger, feedbackBus func(event *nexusv1.EventResponse), repo *campaignrepo.Repository) *CampaignStateManager {
	m := &CampaignStateManager{
		log:         log,
		feedbackBus: feedbackBus,
		repo:        repo,
	}
	// Attempt to load the default campaign at startup
	if err := m.LoadDefaultCampaign("start/default_campaign.json"); err != nil {
		log.Warn("Failed to load default campaign", zap.Error(err))
	}
	// Optionally preload all campaigns from DB
	if repo != nil {
		if err := m.LoadAllCampaignsFromDB(); err != nil {
			log.Warn("Failed to preload campaigns from DB", zap.Error(err))
		}
	}
	return m
}

// LoadAllCampaignsFromDB loads all campaigns from the DB and populates the state map.
func (m *CampaignStateManager) LoadAllCampaignsFromDB() error {
	if m.repo == nil {
		return nil
	}
	campaigns, err := m.repo.List(context.Background(), 1000, 0)
	if err != nil {
		return err
	}
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
		m.log.Info("Loaded campaign from DB into state manager", zap.String("campaign_id", c.Slug))
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
	m.log.Info("Loaded default campaign into state manager", zap.String("campaign_id", campaignID))
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

// UpdateState updates the campaign state and emits a real-time feedback event.
func (m *CampaignStateManager) UpdateState(campaignID, userID string, update map[string]any, metadata *commonpb.Metadata) {
	cs := m.GetOrCreateState(campaignID)
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
	maps.Copy(cs.State, update)
	cs.LastUpdated = time.Now()
	structData := meta.NewStructFromMap(cs.State, nil)

	event := &nexusv1.EventResponse{
		Success:   true,
		EventId:   "state_update:" + campaignID + ":" + userID,
		EventType: "campaign:state:v1:success",
		Message:   "state_updated",
		Metadata:  metadata,
		Payload: &commonpb.Payload{
			Data: structData,
		},
	}
	m.feedbackBus(event)

	// Notify all subscribers for this campaign, panic-safe
	cs.Subscribers.Range(func(key, value interface{}) bool {
		// Reference unused parameter 'key' for diagnostics
		_ = key
		ch, ok := value.(chan *nexusv1.EventResponse)
		if !ok {
			return true
		}
		m.safeGo(func() {
			select {
			case ch <- event:
			default:
			}
		})
		return true
	})
}

// Subscribe adds a user to the campaign's real-time feedback channel.
func (m *CampaignStateManager) Subscribe(campaignID, userID string) <-chan *nexusv1.EventResponse {
	cs := m.GetOrCreateState(campaignID)
	ch := make(chan *nexusv1.EventResponse, 16)
	cs.Subscribers.Store(userID, ch)
	return ch
}

// Unsubscribe removes a user from the campaign's feedback channel.
func (m *CampaignStateManager) Unsubscribe(campaignID, userID string) {
	cs := m.GetOrCreateState(campaignID)
	if chVal, ok := cs.Subscribers.Load(userID); ok {
		if ch, ok2 := chVal.(chan *nexusv1.EventResponse); ok2 {
			close(ch)
		}
		cs.Subscribers.Delete(userID)
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
	_, userID := m.extractCampaignAndUserID(event)

	// Log the type and structure of incoming metadata for debugging
	if event.Metadata != nil && event.Metadata.ServiceSpecific != nil {
		for k, v := range event.Metadata.ServiceSpecific.Fields {
			switch v.Kind.(type) {
			case *structpb.Value_StringValue:
				m.log.Warn("CampaignList: Metadata field is string", zap.String("field", k), zap.String("value", v.GetStringValue()))
			case *structpb.Value_StructValue:
				m.log.Info("CampaignList: Metadata field is struct", zap.String("field", k))
			case *structpb.Value_NumberValue:
				m.log.Info("CampaignList: Metadata field is number", zap.String("field", k), zap.Float64("value", v.GetNumberValue()))
			case *structpb.Value_BoolValue:
				m.log.Info("CampaignList: Metadata field is bool", zap.String("field", k), zap.Bool("value", v.GetBoolValue()))
			default:
				m.log.Info("CampaignList: Metadata field is other type", zap.String("field", k))
			}
		}
	} else {
		m.log.Warn("CampaignList: Metadata or ServiceSpecific is nil")
	}

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

	m.log.Info("Processing campaign list request",
		zap.String("user_id", userID),
		zap.Int("limit", payload.Limit),
		zap.Int("offset", payload.Offset))

	var campaigns []map[string]any

	// If repository is available, fetch from database
	if m.repo != nil {
		if dbCampaigns, err := m.repo.List(ctx, payload.Limit, payload.Offset); err == nil {
			for _, c := range dbCampaigns {
				campaignData := map[string]any{
					"id":    c.ID,
					"slug":  c.Slug,
					"title": c.Title,
					"name":  c.Title, // Alias for frontend compatibility
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

				campaigns = append(campaigns, campaignData)
			}
		} else {
			m.log.Error("Failed to fetch campaigns from database", zap.Error(err))
		}
	}

	// Fallback: add campaigns from memory state if database failed or is unavailable
	if len(campaigns) == 0 {
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

			campaignData := map[string]any{
				"id":   campaignID,
				"slug": campaignID,
				"name": campaignID,
			}

			// Add state data
			for k, v := range cs.State {
				campaignData[k] = v
			}

			campaigns = append(campaigns, campaignData)
			return true
		})
	}

	// Always ensure at least the default campaign exists
	if len(campaigns) == 0 {
		campaigns = append(campaigns, map[string]any{
			"id":       0,
			"slug":     "ovasabi_website",
			"name":     "Ovasabi Website",
			"title":    "Ovasabi Website",
			"features": []string{},
		})
	}

	// Extract correlationId from metadata or payload
	var correlationID string
	if event.Metadata != nil && event.Metadata.ServiceSpecific != nil && event.Metadata.ServiceSpecific.Fields != nil {
		if v, ok := event.Metadata.ServiceSpecific.Fields["correlation_id"]; ok && v != nil {
			correlationID = v.GetStringValue()
		}
	}
	if correlationID == "" && event.Payload != nil && event.Payload.Data != nil {
		if v, ok := event.Payload.Data.Fields["correlationId"]; ok && v != nil {
			correlationID = v.GetStringValue()
		}
	}

	// Create response
	responsePayload := map[string]any{
		"campaigns":     campaigns,
		"total":         len(campaigns),
		"limit":         payload.Limit,
		"offset":        payload.Offset,
		"user_id":       userID,
		"correlationId": correlationID,
	}

	structData := meta.NewStructFromMap(responsePayload, nil)
	response := &nexusv1.EventResponse{
		Success:   true,
		EventId:   "campaign_list:" + userID,
		EventType: "campaign:list:v1:success",
		Message:   "campaign_list_retrieved",
		Metadata:  event.Metadata,
		Payload: &commonpb.Payload{
			Data: structData,
		},
	}

	m.log.Info("Sending campaign list response",
		zap.String("user_id", userID),
		zap.Int("campaign_count", len(campaigns)))

	m.feedbackBus(response)
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

	// Update state directly
	m.UpdateState(payload.CampaignID, userID, payload.Updates, event.Metadata)

	// Optionally persist to database asynchronously
	if m.repo != nil {
		m.safeGo(func() {
			m.persistToDB(ctx, payload.CampaignID, payload.Updates)
		})
	}
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

// extractCampaignAndUserID extracts campaign and user IDs from event metadata.
func (m *CampaignStateManager) extractCampaignAndUserID(event *nexusv1.EventRequest) (campaignID, userID string) {
	if event.Metadata != nil {
		// Try to get from global metadata
		if global, ok := event.Metadata.ServiceSpecific.Fields["global"]; ok && global != nil {
			if globalStruct := global.GetStructValue(); globalStruct != nil {
				globalMap := globalStruct.AsMap()
				if v, ok := globalMap["campaign_id"].(string); ok {
					campaignID = v
				}
				if v, ok := globalMap["user_id"].(string); ok {
					userID = v
				}
			}
		}

		// Try to get from campaign-specific metadata
		if campaign, ok := event.Metadata.ServiceSpecific.Fields["campaign"]; ok && campaign != nil {
			if campaignStruct := campaign.GetStructValue(); campaignStruct != nil {
				campaignMap := campaignStruct.AsMap()
				if v, ok := campaignMap["campaign_id"].(string); ok && campaignID == "" {
					campaignID = v
				}
				if v, ok := campaignMap["slug"].(string); ok && campaignID == "" {
					campaignID = v
				}
			}
		}
	}

	// Fallback: try top-level campaign ID
	if campaignID == "" && event.CampaignId != 0 {
		campaignID = "0" // Default for system campaign
	}

	return campaignID, userID
}

// persistToDB asynchronously persists campaign state changes to the database.
func (m *CampaignStateManager) persistToDB(ctx context.Context, campaignID string, updates map[string]any) {
	cancel := func() {}
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	}
	defer cancel()

	// Get campaign from DB
	campaign, err := m.repo.GetBySlug(ctx, campaignID)
	if err != nil {
		m.log.Error("Failed to get campaign for persistence",
			zap.String("campaign_id", campaignID),
			zap.Error(err))
		return
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
		return
	}

	m.log.Info("Successfully persisted campaign state to database",
		zap.String("campaign_id", campaignID))
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
