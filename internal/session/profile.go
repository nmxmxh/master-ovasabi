package session

import (
	"encoding/json"
	"sync"
	"time"
)

// UIState mirrors the frontend UIState type for accessibility and personalization
// Add or adjust fields as needed to match your frontend
// (This is a minimal version; expand as needed).
type UIState struct {
	Theme     string `json:"theme"`
	Locale    string `json:"locale"`
	Direction string `json:"direction"`
	Contrast  struct {
		PrefersHighContrast bool   `json:"prefers_high_contrast"`
		ColorContrastRatio  string `json:"color_contrast_ratio,omitempty"`
	} `json:"contrast"`
	AnimationDirection string `json:"animation_direction,omitempty"`
	Orientation        string `json:"orientation,omitempty"`
	Menu               string `json:"menu,omitempty"`
	FooterLink         string `json:"footer_link,omitempty"`
	Viewport           struct {
		Width  int `json:"width,omitempty"`
		Height int `json:"height,omitempty"`
		DVW    int `json:"dvw,omitempty"`
		DVH    int `json:"dvh,omitempty"`
	} `json:"viewport"`
	Motion struct {
		PrefersReducedMotion bool `json:"prefers_reduced_motion"`
	} `json:"motion"`
	// ...add more as needed
}

type WaitlistData struct {
	Position      int    `json:"position"`
	ReferralCount int    `json:"referral_count"`
	Status        string `json:"status"`
	// ...add more as needed
}

type ReferralData struct {
	Milestones []int `json:"milestones"`
	// ...add more as needed
}

type QuoteData struct {
	LeadStatus string `json:"lead_status"`
	// ...add more as needed
}

type ServiceSpecific struct {
	Waitlist *WaitlistData `json:"waitlist,omitempty"`
	Referral *ReferralData `json:"referral,omitempty"`
	Quote    *QuoteData    `json:"quote,omitempty"`
	// ...add more as needed
}

type CampaignSpecific struct {
	CampaignID int64                  `json:"campaign_id"`
	Slug       string                 `json:"slug"`
	UI         map[string]interface{} `json:"ui"`
	Rules      map[string]interface{} `json:"rules"`
	// ...add more as needed
}

type UnifiedProfile struct {
	SessionID        string           `json:"session_id"`
	UserID           *int64           `json:"user_id,omitempty"`
	WaitlistID       *int64           `json:"waitlist_id,omitempty"`
	GuestID          *string          `json:"guest_id,omitempty"`
	Email            *string          `json:"email,omitempty"`
	Username         *string          `json:"username,omitempty"`
	DisplayName      *string          `json:"display_name,omitempty"`
	Privileges       []string         `json:"privileges,omitempty"`
	UIState          UIState          `json:"ui_state"`
	ServiceSpecific  ServiceSpecific  `json:"service_specific"`
	CampaignSpecific CampaignSpecific `json:"campaign_specific"`
	mu               sync.RWMutex     `json:"-"` // for per-profile concurrency
}

// In-memory session store (for demo/cheapest cost; use Redis/DB for production)
// For production, use optimistic deletion and Redis for persistence.
var (
	sessionStore = make(map[string]*UnifiedProfile)
	storeMu      sync.RWMutex
)

// ScheduleProfileDeletion marks a profile for deletion after a delay (optimistic GDPR erase).
func ScheduleProfileDeletion(sessionID string, delaySeconds int, deleteFn func(string)) {
	go func() {
		// Wait for the delay, then delete
		<-time.After(time.Duration(delaySeconds) * time.Second)
		deleteFn(sessionID)
	}()
}

// DeleteProfile removes a profile from the store and erases sensitive fields (GDPR).
func DeleteProfile(sessionID string) {
	storeMu.Lock()
	defer storeMu.Unlock()
	if profile, ok := sessionStore[sessionID]; ok {
		// Anonymize UI/dialogue state if needed
		profile.mu.Lock()
		profile.Email = nil
		profile.Username = nil
		profile.DisplayName = nil
		profile.Privileges = nil
		profile.ServiceSpecific = ServiceSpecific{}   // clear service data
		profile.CampaignSpecific = CampaignSpecific{} // clear campaign data
		// Optionally, clear UIState except for accessibility (for analytics)
		profile.UIState = UIState{
			Theme:    profile.UIState.Theme,
			Locale:   profile.UIState.Locale,
			Contrast: profile.UIState.Contrast,
		}
		profile.mu.Unlock()
		// Remove from store
		delete(sessionStore, sessionID)
	}
}

// GetProfile returns the UnifiedProfile for a session, or creates a new one if not found.
func GetProfile(sessionID string) *UnifiedProfile {
	storeMu.RLock()
	profile, ok := sessionStore[sessionID]
	storeMu.RUnlock()
	if ok {
		return profile
	}
	// Create new guest profile by default
	profile = &UnifiedProfile{
		SessionID: sessionID,
		GuestID:   &sessionID,
		UIState: UIState{
			Theme:     "light",
			Locale:    "en",
			Direction: "ltr",
			Contrast: struct {
				PrefersHighContrast bool   `json:"prefers_high_contrast"`
				ColorContrastRatio  string `json:"color_contrast_ratio,omitempty"`
			}{},
		},
		Privileges: []string{"guest"},
	}
	storeMu.Lock()
	sessionStore[sessionID] = profile
	storeMu.Unlock()
	return profile
}

// MergeProfiles merges two profiles (e.g., guest -> waitlist, waitlist -> user)
// Uses per-profile locking for thread safety.
func MergeProfiles(oldP, newP *UnifiedProfile) *UnifiedProfile {
	oldP.mu.Lock()
	defer oldP.mu.Unlock()
	if newP.UserID != nil {
		oldP.UserID = newP.UserID
	}
	if newP.WaitlistID != nil {
		oldP.WaitlistID = newP.WaitlistID
	}
	if newP.Email != nil {
		oldP.Email = newP.Email
	}
	if newP.Username != nil {
		oldP.Username = newP.Username
	}
	if newP.DisplayName != nil {
		oldP.DisplayName = newP.DisplayName
	}
	// Merge privileges (union, dedup)
	privMap := make(map[string]struct{})
	for _, p := range oldP.Privileges {
		privMap[p] = struct{}{}
	}
	for _, p := range newP.Privileges {
		privMap[p] = struct{}{}
	}
	privs := make([]string, 0, len(privMap))
	for p := range privMap {
		privs = append(privs, p)
	}
	oldP.Privileges = privs
	// Merge UIState (prefer newP if set)
	if newP.UIState.Theme != "" {
		oldP.UIState.Theme = newP.UIState.Theme
	}
	if newP.UIState.Locale != "" {
		oldP.UIState.Locale = newP.UIState.Locale
	}
	if newP.UIState.Direction != "" {
		oldP.UIState.Direction = newP.UIState.Direction
	}
	if newP.UIState.Contrast.PrefersHighContrast {
		oldP.UIState.Contrast.PrefersHighContrast = true
	}
	if newP.UIState.Contrast.ColorContrastRatio != "" {
		oldP.UIState.Contrast.ColorContrastRatio = newP.UIState.Contrast.ColorContrastRatio
	}
	if newP.UIState.AnimationDirection != "" {
		oldP.UIState.AnimationDirection = newP.UIState.AnimationDirection
	}
	if newP.UIState.Orientation != "" {
		oldP.UIState.Orientation = newP.UIState.Orientation
	}
	if newP.UIState.Menu != "" {
		oldP.UIState.Menu = newP.UIState.Menu
	}
	if newP.UIState.FooterLink != "" {
		oldP.UIState.FooterLink = newP.UIState.FooterLink
	}
	if newP.UIState.Viewport.Width != 0 {
		oldP.UIState.Viewport.Width = newP.UIState.Viewport.Width
	}
	if newP.UIState.Viewport.Height != 0 {
		oldP.UIState.Viewport.Height = newP.UIState.Viewport.Height
	}
	if newP.UIState.Viewport.DVW != 0 {
		oldP.UIState.Viewport.DVW = newP.UIState.Viewport.DVW
	}
	if newP.UIState.Viewport.DVH != 0 {
		oldP.UIState.Viewport.DVH = newP.UIState.Viewport.DVH
	}
	if newP.UIState.Motion.PrefersReducedMotion {
		oldP.UIState.Motion.PrefersReducedMotion = true
	}
	// Merge ServiceSpecific (prefer newP if set)
	if newP.ServiceSpecific.Waitlist != nil {
		oldP.ServiceSpecific.Waitlist = newP.ServiceSpecific.Waitlist
	}
	if newP.ServiceSpecific.Referral != nil {
		oldP.ServiceSpecific.Referral = newP.ServiceSpecific.Referral
	}
	if newP.ServiceSpecific.Quote != nil {
		oldP.ServiceSpecific.Quote = newP.ServiceSpecific.Quote
	}
	// Merge CampaignSpecific (prefer newP if set)
	if newP.CampaignSpecific.CampaignID != 0 {
		oldP.CampaignSpecific = newP.CampaignSpecific
	}
	return oldP
}

// ToJSON returns the JSON representation of the profile (thread safe)
// Optionally, redact sensitive fields for frontend streaming (GDPR).
func (p *UnifiedProfile) ToJSON(redact bool) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !redact {
		return json.Marshal(p)
	}
	// Redact sensitive fields
	// Avoid copying the mutex by using an anonymous struct for marshalling
	return json.Marshal(struct {
		SessionID        string           `json:"session_id"`
		UserID           *int64           `json:"user_id,omitempty"`
		WaitlistID       *int64           `json:"waitlist_id,omitempty"`
		GuestID          *string          `json:"guest_id,omitempty"`
		Username         *string          `json:"username,omitempty"`
		DisplayName      *string          `json:"display_name,omitempty"`
		UIState          UIState          `json:"ui_state"`
		CampaignSpecific CampaignSpecific `json:"campaign_specific"`
	}{
		SessionID:        p.SessionID,
		UserID:           p.UserID,
		WaitlistID:       p.WaitlistID,
		GuestID:          p.GuestID,
		Username:         p.Username,
		DisplayName:      p.DisplayName,
		UIState:          p.UIState,
		CampaignSpecific: p.CampaignSpecific,
	})
}

// UpdateProfile updates the profile in the store with a thread-safe update function.
func UpdateProfile(sessionID string, updateFn func(*UnifiedProfile)) {
	storeMu.RLock()
	profile, ok := sessionStore[sessionID]
	storeMu.RUnlock()
	if !ok {
		profile = GetProfile(sessionID)
	}
	profile.mu.Lock()
	defer profile.mu.Unlock()
	updateFn(profile)
}

// SetDialogueState updates the dialogue state for the user (e.g., last message, context).
func (p *UnifiedProfile) SetDialogueState(key string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.CampaignSpecific.UI == nil {
		p.CampaignSpecific.UI = make(map[string]interface{})
	}
	ds, ok := p.CampaignSpecific.UI["dialogue_state"].(map[string]interface{})
	if !ok {
		ds = make(map[string]interface{})
		p.CampaignSpecific.UI["dialogue_state"] = ds
	}
	ds[key] = value
}

// StreamPrivileges returns a copy of the privileges for streaming (thread safe).
func (p *UnifiedProfile) StreamPrivileges() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]string, len(p.Privileges))
	copy(out, p.Privileges)
	return out
}

// SetDisplayName safely sets the display name (thread safe).
func (p *UnifiedProfile) SetDisplayName(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.DisplayName = &name
}

// SetUsername safely sets the username (thread safe).
func (p *UnifiedProfile) SetUsername(username string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Username = &username
}
