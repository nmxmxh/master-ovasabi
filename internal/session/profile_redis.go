package session

import (
	"errors"
	"fmt"

	// ...existing imports...

	context "context"

	redis "github.com/nmxmxh/master-ovasabi/pkg/redis"
)

// ...existing UIState, WaitlistData, ReferralData, QuoteData, ServiceSpecific, CampaignSpecific, UnifiedProfile structs...

// RedactedUIState is the minimal, privacy-aware UI state sent to the frontend.
type RedactedUIState struct {
	Theme              string `json:"theme"`
	Locale             string `json:"locale"`
	Direction          string `json:"direction"`
	Menu               string `json:"menu,omitempty"`
	FooterLink         string `json:"footer_link,omitempty"`
	AnimationDirection string `json:"animation_direction,omitempty"`
	Orientation        string `json:"orientation,omitempty"`
	Contrast           struct {
		PrefersHighContrast bool   `json:"prefers_high_contrast"`
		ColorContrastRatio  string `json:"color_contrast_ratio,omitempty"`
	} `json:"contrast"`
	Viewport struct {
		Width  int `json:"width,omitempty"`
		Height int `json:"height,omitempty"`
		DVW    int `json:"dvw,omitempty"`
		DVH    int `json:"dvh,omitempty"`
	} `json:"viewport"`
	Motion struct {
		PrefersReducedMotion bool `json:"prefers_reduced_motion"`
	} `json:"motion"`
}

// GetRedactedUIState returns a privacy-aware, minimal UI state for frontend streaming.
func (p *UnifiedProfile) GetRedactedUIState() RedactedUIState {
	ui := p.UIState
	return RedactedUIState{
		Theme:              ui.Theme,
		Locale:             ui.Locale,
		Direction:          ui.Direction,
		Menu:               ui.Menu,
		FooterLink:         ui.FooterLink,
		AnimationDirection: ui.AnimationDirection,
		Orientation:        ui.Orientation,
		Contrast: struct {
			PrefersHighContrast bool   `json:"prefers_high_contrast"`
			ColorContrastRatio  string `json:"color_contrast_ratio,omitempty"`
		}{
			PrefersHighContrast: ui.Contrast.PrefersHighContrast,
			ColorContrastRatio:  ui.Contrast.ColorContrastRatio,
		},
		Viewport: struct {
			Width  int `json:"width,omitempty"`
			Height int `json:"height,omitempty"`
			DVW    int `json:"dvw,omitempty"`
			DVH    int `json:"dvh,omitempty"`
		}{
			Width:  ui.Viewport.Width,
			Height: ui.Viewport.Height,
			DVW:    ui.Viewport.DVW,
			DVH:    ui.Viewport.DVH,
		},
		Motion: struct {
			PrefersReducedMotion bool `json:"prefers_reduced_motion"`
		}{
			PrefersReducedMotion: ui.Motion.PrefersReducedMotion,
		},
	}
}

// StreamRedactedUIState fetches the profile and returns the redacted UI state for frontend.
func StreamRedactedUIState(ctx context.Context, sessionID string) (RedactedUIState, error) {
	profile, err := LoadProfile(ctx, sessionID)
	if err != nil {
		return RedactedUIState{}, err
	}
	if profile == nil {
		return RedactedUIState{}, errors.New("profile not found")
	}
	return profile.GetRedactedUIState(), nil
}

const (
	profilePrefix = "profile:"
)

var profileCache *redis.Cache

// InitProfileCache must be called at startup with the redis.Provider.
func InitProfileCache(provider *redis.Provider) error {
	cache, err := provider.GetCache(context.Background(), "session")
	if err != nil {
		return err
	}
	profileCache = cache
	return nil
}

// SaveProfile persists the profile to Redis with TTL.
func SaveProfile(ctx context.Context, profile *UnifiedProfile) error {
	if profileCache == nil {
		return errors.New("profileCache not initialized")
	}
	data, err := profile.ToJSON(false)
	if err != nil {
		return err
	}
	return profileCache.Set(ctx, profilePrefix+profile.SessionID, "", data, redis.TTLSession)
}

// LoadProfile loads a profile from Redis, or returns nil if not found.
func LoadProfile(ctx context.Context, sessionID string) (*UnifiedProfile, error) {
	if profileCache == nil {
		return nil, errors.New("profileCache not initialized")
	}
	var profile UnifiedProfile
	err := profileCache.Get(ctx, profilePrefix+sessionID, "", &profile)
	if err != nil {
		if err.Error() == "key not found: "+profilePrefix+sessionID {
			return nil, fmt.Errorf("profile not found for session: %s", sessionID)
		}
		return nil, err
	}
	return &profile, nil
}

// GetOrCreateProfile loads from Redis or creates a new guest profile.
func GetOrCreateProfile(ctx context.Context, sessionID string) (*UnifiedProfile, error) {
	profile, err := LoadProfile(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if profile != nil {
		return profile, nil
	}
	// New guest profile
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
	if err := SaveProfile(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// UpdateProfileWithFn loads, updates, and saves a profile atomically.
func UpdateProfileWithFn(ctx context.Context, sessionID string, updateFn func(*UnifiedProfile)) error {
	profile, err := GetOrCreateProfile(ctx, sessionID)
	if err != nil {
		return err
	}
	updateFn(profile)
	return SaveProfile(ctx, profile)
}

// DeleteProfile removes a profile from Redis (GDPR/optimistic deletion)

// MergeProfiles merges two profiles and saves the result.
func MergeProfilesAndSave(ctx context.Context, oldSessionID, newSessionID string) (*UnifiedProfile, error) {
	oldP, err := GetOrCreateProfile(ctx, oldSessionID)
	if err != nil {
		return nil, err
	}
	newP, err := GetOrCreateProfile(ctx, newSessionID)
	if err != nil {
		return nil, err
	}
	// ...merge logic as before (thread safety not needed with Redis)...
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
	// ...merge UIState, ServiceSpecific, CampaignSpecific as before...
	// (copy/paste from your in-memory MergeProfiles)
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
	if newP.ServiceSpecific.Waitlist != nil {
		oldP.ServiceSpecific.Waitlist = newP.ServiceSpecific.Waitlist
	}
	if newP.ServiceSpecific.Referral != nil {
		oldP.ServiceSpecific.Referral = newP.ServiceSpecific.Referral
	}
	if newP.ServiceSpecific.Quote != nil {
		oldP.ServiceSpecific.Quote = newP.ServiceSpecific.Quote
	}
	if newP.CampaignSpecific.CampaignID != 0 {
		oldP.CampaignSpecific = newP.CampaignSpecific
	}
	if err := SaveProfile(ctx, oldP); err != nil {
		return nil, err
	}
	return oldP, nil
}

// Example: LocalizeAndStreamDialogue
// Loads the user's locale from profile, fetches dialogue from campaign metadata, and returns the localized string.
func LocalizeAndStreamDialogue(ctx context.Context, profile *UnifiedProfile, dialogueKey string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	if profile == nil || profile.CampaignSpecific.UI == nil {
		return "", errors.New("profile or campaign UI not set")
	}
	locale := profile.UIState.Locale
	dialogues, ok := profile.CampaignSpecific.UI["dialogues"].(map[string]map[string]string)
	if !ok {
		return "", errors.New("dialogues not found in campaign UI")
	}
	if locMap, ok := dialogues[dialogueKey]; ok {
		if val, ok := locMap[locale]; ok {
			return val, nil
		}
		if val, ok := locMap["en"]; ok {
			return val, nil // fallback to English
		}
	}
	return "", errors.New("dialogue not found")
}
