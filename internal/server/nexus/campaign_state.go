package nexus

import (
	"context"
	"encoding/json"
	"maps"
	"os"
	"sync"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	nexusv1 "github.com/nmxmxh/master-ovasabi/api/protos/nexus/v1"
	campaignrepo "github.com/nmxmxh/master-ovasabi/internal/service/campaign"
	meta "github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"go.uber.org/zap"
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

// EmitCampaignState emits the current campaign state to a specific user (e.g., on handshake or request)
func (m *CampaignStateManager) EmitCampaignState(campaignID, userID string, metadata *commonpb.Metadata) {
	state := m.GetState(campaignID)
	structData := meta.NewStructFromMap(state, nil)
	event := &nexusv1.EventResponse{
		Success:   true,
		EventId:   "state_init:" + campaignID + ":" + userID,
		EventType: "campaign:state:v1:success",
		Message:   "state_init",
		Metadata:  metadata,
		Payload: &commonpb.Payload{
			Data: structData,
		},
	}
	m.feedbackBus(event)
}

// HandleEvent is a generic event handler for campaign-related events (to be called from the Nexus event bus)
func (m *CampaignStateManager) HandleEvent(event *nexusv1.EventRequest) {
	switch event.EventType {
	case "campaign:state:request":
		// Expect campaign_id and user_id in metadata or payload
		var campaignID, userID string
		if event.Metadata != nil {
			if global, ok := event.Metadata.ServiceSpecific.Fields["global"]; ok && global != nil {
				if globalStruct := global.GetStructValue(); globalStruct != nil {
					if v, ok := globalStruct.AsMap()["campaign_id"]; ok {
						if s, ok := v.(string); ok {
							campaignID = s
						}
					}
					if v, ok := globalStruct.AsMap()["user_id"]; ok {
						if s, ok := v.(string); ok {
							userID = s
						}
					}
				}
			}
		}
		// Fallback: try top-level fields
		if campaignID == "" && event.CampaignId != 0 {
			campaignID = "0"
			// If you want to use the int64 value as string:
			// campaignID = strconv.FormatInt(event.CampaignId, 10)
		}
		m.EmitCampaignState(campaignID, userID, event.Metadata)
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
	campaignID, _ := data["slug"].(string)
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
		return val.(*CampaignState)
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
func (m *CampaignStateManager) UpdateState(campaignID string, userID string, update map[string]any, metadata *commonpb.Metadata) {
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
	copy := make(map[string]any, len(cs.State))
	for k, v := range cs.State {
		copy[k] = v
	}
	return copy
}
