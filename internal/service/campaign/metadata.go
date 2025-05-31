// Canonical Campaign Metadata Structure
// ------------------------------------
// This file defines the canonical, extensible metadata structure for campaigns.
// All campaign-specific metadata must be namespaced under service_specific.campaign in the common.Metadata proto.
//
// Each section is documented with its purpose, type, relation to other services, and extensibility notes.
//
// References:
// - docs/amadeus/amadeus_context.md
// - docs/services/metadata.md
// - docs-site.tar.pdf (for industry and orchestration inspiration)
// - internal/service/nexus/events.go (for event types)
//
// Usage:
// - All campaign orchestration, onboarding, localization, and feature toggles should be defined here.
// - Services should read/extend only the sections relevant to them.
// - New features/services should add new fields or namespaced sections as needed.
// - This struct is the authoritative reference for campaign metadata in the codebase.

package campaign

import (
	"fmt"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

// Supported campaign type and status constants.
const (
	CampaignTypeScheduled    = "scheduled"
	CampaignTypeFlash        = "flash"
	CampaignTypeDrip         = "drip"
	CampaignTypeEvergreen    = "evergreen"
	CampaignTypeRecurring    = "recurring"
	CampaignTypeSeasonal     = "seasonal"
	CampaignTypeTargeted     = "targeted"
	CampaignTypeTest         = "test"
	CampaignTypeExperimental = "experimental"

	CampaignStatusActive    = "active"
	CampaignStatusInactive  = "inactive"
	CampaignStatusScheduled = "scheduled"
	CampaignStatusCompleted = "completed"
	CampaignStatusArchived  = "archived"
)

// VersioningInfo tracks version and environment for traceability.
// Used by: All services (audit, migration, compliance).
type VersioningInfo struct {
	SystemVersion  string    `json:"system_version"`   // Platform-wide version
	ServiceVersion string    `json:"service_version"`  // Service-specific version
	Environment    string    `json:"environment"`      // e.g., "production", "staging"
	LastMigratedAt time.Time `json:"last_migrated_at"` // Last migration timestamp
}

// SchedulingInfo describes campaign scheduling and jobs.
// Used by: Scheduler, Notification, Analytics
// Enables time-based orchestration, triggers, and automation.
type SchedulingInfo struct {
	Start      time.Time                `json:"start"`                // Campaign start time
	End        time.Time                `json:"end"`                  // Campaign end time
	Recurrence string                   `json:"recurrence,omitempty"` // e.g., cron, interval
	Jobs       []map[string]interface{} `json:"jobs,omitempty"`       // Scheduled jobs (type, params)
}

// LocalizationInfo describes supported locales and translations.
// Used by: Localization, Content, Notification, WebSocket
// Enables multi-locale support, translation, and accessibility.
type LocalizationInfo struct {
	SupportedLocales []string                     `json:"supported_locales"` // e.g., ["en-US", "fr-FR"]
	DefaultLocale    string                       `json:"default_locale"`    // e.g., "en-US"
	Translations     map[string]map[string]string `json:"translations"`      // locale -> key -> text
}

// ContentInfo describes content types, templates, and moderation.
// Used by: Content, Moderation, Analytics
// Controls UGC, templates, and moderation settings.
type ContentInfo struct {
	Types      []string               `json:"types"`      // e.g., ["post", "image", "video"]
	Templates  map[string]interface{} `json:"templates"`  // e.g., {"welcome": {...}}
	Moderation string                 `json:"moderation"` // "auto", "manual", "none"
}

// CommunityInfo describes community features and real-time state.
// Used by: WebSocket, Content, Notification, Analytics
// Controls real-time features, chat, leaderboards, etc.
type CommunityInfo struct {
	Sections    []string               `json:"sections"`    // e.g., ["leaderboard", "chat"]
	Leaderboard map[string]interface{} `json:"leaderboard"` // e.g., {"criteria": "referrals"}
	Chat        map[string]interface{} `json:"chat"`        // e.g., {"enabled": true}
}

// OnboardingInfo describes onboarding flows and questions.
// Used by: User, Notification, Analytics, Localization
// Enables dynamic onboarding, interest types, and questionnaires.
type OnboardingInfo struct {
	InterestTypes []string                                       `json:"interest_types"` // e.g., ["talent", "business"]
	Questionnaire map[string][]map[string]map[string]interface{} `json:"questionnaire"`  // interest_type -> list of questions (id, text, type)
}

// ReferralInfo describes referral and viral growth mechanics.
// Used by: Referral, Notification, Analytics
// Enables viral growth, rewards, and referral tracking.
type ReferralInfo struct {
	Enabled bool                   `json:"enabled"` // Is referral enabled?
	Reward  map[string]interface{} `json:"reward"`  // e.g., {"type": "points", "value": 10}
}

// CommerceInfo describes payments, bookings, and monetization.
// Used by: Commerce, Booking, Notification
// Enables payments, bookings, and monetization features.
type CommerceInfo struct {
	PaymentMethods []string               `json:"payment_methods"` // e.g., ["stripe", "paypal"]
	Booking        map[string]interface{} `json:"booking"`         // e.g., {"enabled": true}
}

// AnalyticsInfo describes tracking and reporting.
// Used by: Analytics, Notification
// Enables event tracking, reporting, and optimization.
type AnalyticsInfo struct {
	TrackEvents []string               `json:"track_events"` // e.g., ["join", "purchase"]
	Goals       map[string]interface{} `json:"goals"`        // e.g., {"type": "engagement", "target": 1000}
}

// ComplianceInfo describes accessibility, legal, and audit.
// Used by: Compliance, Content, Localization
// Ensures accessibility, legal compliance, and auditability.
type ComplianceInfo struct {
	Accessibility map[string]interface{} `json:"accessibility"` // e.g., {"wcag": "AA"}
	Legal         map[string]interface{} `json:"legal"`         // e.g., {"gdpr": true}
}

// CustomInfo allows extensibility for future or domain-specific needs.
// Used by: All services (future extensibility).
type CustomInfo map[string]interface{}

// Metadata is the canonical, extensible metadata structure for campaigns.
// This struct is the authoritative reference for campaign metadata and orchestration.
// Each field is documented with its type, purpose, and relation to other services.
type Metadata struct {
	// ID is the unique campaign ID/slug.
	// Used by: All services for identification and cross-referencing.
	ID string `json:"id"`

	// Type is the campaign type (e.g., "onboarding", "promo", "ugc", ...).
	// Used by: All services for routing, orchestration, and analytics.
	Type string `json:"type"`

	// Status is the campaign status (e.g., "active", "inactive", ...).
	// Used by: All services for lifecycle management and orchestration.
	Status string `json:"status"`

	// Versioning provides version and environment info for traceability.
	// Used by: All services (audit, migration, compliance).
	Versioning *VersioningInfo `json:"versioning"`

	// Scheduling describes campaign scheduling and jobs.
	// Used by: Scheduler, Notification, Analytics for time-based orchestration.
	Scheduling *SchedulingInfo `json:"scheduling"`

	// Features is a list of feature toggles (e.g., ["waitlist", "leaderboard"]).
	// Used by: All services to enable/disable campaign features dynamically.
	Features []string `json:"features"`

	// Localization describes supported locales and translations.
	// Used by: Localization, Content, Notification, WebSocket for multi-locale support.
	Localization *LocalizationInfo `json:"localization"`

	// Content describes content types, templates, and moderation.
	// Used by: Content, Moderation, Analytics for UGC and moderation settings.
	Content *ContentInfo `json:"content"`

	// Community describes community features and real-time state.
	// Used by: WebSocket, Content, Notification, Analytics for real-time features.
	Community *CommunityInfo `json:"community"`

	// Onboarding describes onboarding flows and questions.
	// Used by: User, Notification, Analytics, Localization for dynamic onboarding.
	Onboarding *OnboardingInfo `json:"onboarding"`

	// Referral describes referral and viral growth mechanics.
	// Used by: Referral, Notification, Analytics for viral growth and rewards.
	Referral *ReferralInfo `json:"referral"`

	// Commerce describes payments, bookings, and monetization.
	// Used by: Commerce, Booking, Notification for payments and bookings.
	Commerce *CommerceInfo `json:"commerce"`

	// Analytics describes tracking and reporting.
	// Used by: Analytics, Notification for event tracking and optimization.
	Analytics *AnalyticsInfo `json:"analytics"`

	// Compliance describes accessibility, legal, and audit.
	// Used by: Compliance, Content, Localization for compliance and auditability.
	Compliance *ComplianceInfo `json:"compliance"`

	// Custom allows extensibility for future or domain-specific needs.
	// Used by: All services for future extensibility and experimental features.
	Custom CustomInfo `json:"custom"`
}

/*
Documentation:
- Each section is namespaced and relates to a specific service or cross-service concern.
- Services should only read/extend the sections relevant to them.
- New features/services should add new fields or extend the Custom section.

Relation to Other Services:
- Versioning: All services (audit, migration)
- Scheduling: Scheduler, Notification, Analytics
- Features: All (feature toggles)
- Localization: Localization, Content, Notification, WebSocket
- Content: Content, Moderation, Analytics
- Community: WebSocket, Content, Notification, Analytics
- Onboarding: User, Notification, Analytics, Localization
- Referral: Referral, Notification, Analytics
- Commerce: Commerce, Booking, Notification
- Analytics: Analytics, Notification
- Compliance: Compliance, Content, Localization
- Custom: All (future extensibility)

Extensibility:
- Add new fields to the struct as new services/features are introduced.
- Use the Custom field for domain-specific or experimental features.
- Document all changes and update the authoritative schema reference.

Example Extension:
// To add a new "gamification" feature:
type GamificationInfo struct {
	Enabled bool                   `json:"enabled"`
	Badges  []string               `json:"badges"`
	Points  int                    `json:"points"`
}
// Add to CampaignMetadata:
// Gamification *GamificationInfo `json:"gamification"`
*/

// ToStruct converts Metadata to a structpb.Struct for proto usage.
func (m *Metadata) ToStruct() (*structpb.Struct, error) {
	fields := map[string]interface{}{
		"type":   m.Type,
		"status": m.Status,
	}
	if m.Versioning != nil {
		fields["versioning"] = map[string]interface{}{
			"system_version":   m.Versioning.SystemVersion,
			"service_version":  m.Versioning.ServiceVersion,
			"environment":      m.Versioning.Environment,
			"last_migrated_at": m.Versioning.LastMigratedAt.Format(time.RFC3339),
		}
	}
	if len(m.Features) > 0 {
		fields["features"] = m.Features
	}
	if m.Localization != nil {
		fields["localization"] = m.Localization
	}
	if m.Content != nil {
		fields["content"] = m.Content
	}
	if m.Community != nil {
		fields["community"] = m.Community
	}
	if m.Onboarding != nil {
		fields["onboarding"] = m.Onboarding
	}
	if m.Referral != nil {
		fields["referral"] = m.Referral
	}
	if m.Commerce != nil {
		fields["commerce"] = m.Commerce
	}
	if m.Analytics != nil {
		fields["analytics"] = m.Analytics
	}
	if m.Compliance != nil {
		fields["compliance"] = m.Compliance
	}
	if m.Custom != nil {
		fields["custom"] = m.Custom
	}
	return metadata.NewStructFromMap(fields, nil), nil
}

// FromStruct parses a structpb.Struct into Metadata.
func FromStruct(s *structpb.Struct) (*Metadata, error) {
	m := &Metadata{}
	if v, ok := s.Fields["type"]; ok {
		m.Type = v.GetStringValue()
	} else {
		return nil, fmt.Errorf("missing required field: type")
	}
	if v, ok := s.Fields["status"]; ok {
		m.Status = v.GetStringValue()
	} else {
		return nil, fmt.Errorf("missing required field: status")
	}
	if v, ok := s.Fields["versioning"]; ok {
		if verMap := v.GetStructValue().AsMap(); verMap != nil {
			m.Versioning = &VersioningInfo{}
			if sv, ok := verMap["system_version"].(string); ok {
				m.Versioning.SystemVersion = sv
			}
			if sv, ok := verMap["service_version"].(string); ok {
				m.Versioning.ServiceVersion = sv
			}
			if env, ok := verMap["environment"].(string); ok {
				m.Versioning.Environment = env
			}
			if lm, ok := verMap["last_migrated_at"].(string); ok {
				if t, err := time.Parse(time.RFC3339, lm); err == nil {
					m.Versioning.LastMigratedAt = t
				}
			}
		}
	}
	if v, ok := s.Fields["features"]; ok {
		m.Features = toStringSlice(v)
	}
	if v, ok := s.Fields["localization"]; ok {
		m.Localization = toLocalizationInfo(v)
	}
	if v, ok := s.Fields["content"]; ok {
		m.Content = toContentInfo(v)
	}
	if v, ok := s.Fields["community"]; ok {
		m.Community = toCommunityInfo(v)
	}
	if v, ok := s.Fields["onboarding"]; ok {
		m.Onboarding = toOnboardingInfo(v)
	}
	if v, ok := s.Fields["referral"]; ok {
		m.Referral = toReferralInfo(v)
	}
	if v, ok := s.Fields["commerce"]; ok {
		m.Commerce = toCommerceInfo(v)
	}
	if v, ok := s.Fields["analytics"]; ok {
		m.Analytics = toAnalyticsInfo(v)
	}
	if v, ok := s.Fields["compliance"]; ok {
		m.Compliance = toComplianceInfo(v)
	}
	if v, ok := s.Fields["custom"]; ok {
		m.Custom = v.GetStructValue().AsMap()
	}
	return m, nil
}

// Validate checks required fields and logical consistency for campaign metadata.
func (m *Metadata) Validate() error {
	if m.Type == "" {
		return fmt.Errorf("campaign type is required")
	}
	if m.Status == "" {
		return fmt.Errorf("campaign status is required")
	}
	if m.Scheduling == nil {
		return fmt.Errorf("scheduling info is required")
	}
	// Validate allowed values
	allowedTypes := map[string]bool{
		CampaignTypeScheduled: true, CampaignTypeFlash: true, CampaignTypeDrip: true,
		CampaignTypeEvergreen: true, CampaignTypeRecurring: true, CampaignTypeSeasonal: true,
		CampaignTypeTargeted: true, CampaignTypeTest: true, CampaignTypeExperimental: true,
	}
	if !allowedTypes[m.Type] {
		return fmt.Errorf("invalid campaign type: %s", m.Type)
	}
	allowedStatus := map[string]bool{
		CampaignStatusActive: true, CampaignStatusInactive: true, CampaignStatusScheduled: true,
		CampaignStatusCompleted: true, CampaignStatusArchived: true,
	}
	if !allowedStatus[m.Status] {
		return fmt.Errorf("invalid campaign status: %s", m.Status)
	}
	if m.Scheduling != nil && !m.Scheduling.Start.IsZero() && !m.Scheduling.End.IsZero() && m.Scheduling.Start.After(m.Scheduling.End) {
		return fmt.Errorf("scheduling start must be before end")
	}
	return nil
}

// Helper: convert a structpb.Value to []string.
func toStringSlice(v *structpb.Value) []string {
	if lv := v.GetListValue(); lv != nil {
		out := make([]string, 0, len(lv.Values))
		for _, item := range lv.Values {
			if s := item.GetStringValue(); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// Helper: convert structpb.Value to LocalizationInfo.
func toLocalizationInfo(v *structpb.Value) *LocalizationInfo {
	if s := v.GetStructValue(); s != nil {
		m := s.AsMap()
		li := &LocalizationInfo{}
		if sl, ok := m["supported_locales"].([]interface{}); ok {
			for _, item := range sl {
				if s, ok := item.(string); ok {
					li.SupportedLocales = append(li.SupportedLocales, s)
				}
			}
		}
		if dl, ok := m["default_locale"].(string); ok {
			li.DefaultLocale = dl
		}
		if tr, ok := m["translations"].(map[string]interface{}); ok {
			li.Translations = make(map[string]map[string]string)
			for k, v := range tr {
				if tm, ok := v.(map[string]interface{}); ok {
					li.Translations[k] = make(map[string]string)
					for key, text := range tm {
						if s, ok := text.(string); ok {
							li.Translations[k][key] = s
						}
					}
				}
			}
		}
		return li
	}
	return nil
}

// Helper: convert structpb.Value to ContentInfo.
func toContentInfo(v *structpb.Value) *ContentInfo {
	if s := v.GetStructValue(); s != nil {
		m := s.AsMap()
		ci := &ContentInfo{}
		if lv, ok := m["types"].([]interface{}); ok {
			for _, item := range lv {
				if s, ok := item.(string); ok {
					ci.Types = append(ci.Types, s)
				}
			}
		}
		if tm, ok := m["templates"].(map[string]interface{}); ok {
			ci.Templates = tm
		}
		if mod, ok := m["moderation"].(string); ok {
			ci.Moderation = mod
		}
		return ci
	}
	return nil
}

// Helper: convert structpb.Value to CommunityInfo.
func toCommunityInfo(v *structpb.Value) *CommunityInfo {
	if s := v.GetStructValue(); s != nil {
		m := s.AsMap()
		ci := &CommunityInfo{}
		if lv, ok := m["sections"].([]interface{}); ok {
			for _, item := range lv {
				if s, ok := item.(string); ok {
					ci.Sections = append(ci.Sections, s)
				}
			}
		}
		if lb, ok := m["leaderboard"].(map[string]interface{}); ok {
			ci.Leaderboard = lb
		}
		if ch, ok := m["chat"].(map[string]interface{}); ok {
			ci.Chat = ch
		}
		return ci
	}
	return nil
}

// Helper: convert structpb.Value to OnboardingInfo.
func toOnboardingInfo(v *structpb.Value) *OnboardingInfo {
	if s := v.GetStructValue(); s != nil {
		m := s.AsMap()
		oi := &OnboardingInfo{}
		if it, ok := m["interest_types"].([]interface{}); ok {
			for _, item := range it {
				if s, ok := item.(string); ok {
					oi.InterestTypes = append(oi.InterestTypes, s)
				}
			}
		}
		if q, ok := m["questionnaire"].(map[string]interface{}); ok {
			// interest_type -> list of questions (id, text, type)
			// Each question is map[string]map[string]interface{}
			// So we need to build: map[string][]map[string]map[string]interface{}
			result := make(map[string][]map[string]map[string]interface{})
			for k, v := range q {
				if arr, ok := v.([]interface{}); ok {
					for _, qitem := range arr {
						if qm, ok := qitem.(map[string]interface{}); ok {
							// Convert map[string]interface{} to map[string]map[string]interface{} if possible
							question := make(map[string]map[string]interface{})
							for qk, qv := range qm {
								if qvm, ok := qv.(map[string]interface{}); ok {
									question[qk] = qvm
								}
							}
							result[k] = append(result[k], question)
						}
					}
				}
			}
			oi.Questionnaire = result
		}
		return oi
	}
	return nil
}

// Helper: convert structpb.Value to ReferralInfo.
func toReferralInfo(v *structpb.Value) *ReferralInfo {
	if s := v.GetStructValue(); s != nil {
		m := s.AsMap()
		ri := &ReferralInfo{}
		if en, ok := m["enabled"].(bool); ok {
			ri.Enabled = en
		}
		if r, ok := m["reward"].(map[string]interface{}); ok {
			ri.Reward = r
		}
		return ri
	}
	return nil
}

// Helper: convert structpb.Value to CommerceInfo.
func toCommerceInfo(v *structpb.Value) *CommerceInfo {
	if s := v.GetStructValue(); s != nil {
		m := s.AsMap()
		ci := &CommerceInfo{}
		if pm, ok := m["payment_methods"].([]interface{}); ok {
			for _, item := range pm {
				if s, ok := item.(string); ok {
					ci.PaymentMethods = append(ci.PaymentMethods, s)
				}
			}
		}
		if b, ok := m["booking"].(map[string]interface{}); ok {
			ci.Booking = b
		}
		return ci
	}
	return nil
}

// Helper: convert structpb.Value to AnalyticsInfo.
func toAnalyticsInfo(v *structpb.Value) *AnalyticsInfo {
	if s := v.GetStructValue(); s != nil {
		m := s.AsMap()
		ai := &AnalyticsInfo{}
		if te, ok := m["track_events"].([]interface{}); ok {
			for _, item := range te {
				if s, ok := item.(string); ok {
					ai.TrackEvents = append(ai.TrackEvents, s)
				}
			}
		}
		if g, ok := m["goals"].(map[string]interface{}); ok {
			ai.Goals = g
		}
		return ai
	}
	return nil
}

// Helper: convert structpb.Value to ComplianceInfo.
func toComplianceInfo(v *structpb.Value) *ComplianceInfo {
	if s := v.GetStructValue(); s != nil {
		m := s.AsMap()
		ci := &ComplianceInfo{}
		if acc, ok := m["accessibility"].(map[string]interface{}); ok {
			ci.Accessibility = acc
		}
		if leg, ok := m["legal"].(map[string]interface{}); ok {
			ci.Legal = leg
		}
		return ci
	}
	return nil
}

// GetUserRoleInCampaign returns the user's role ("admin", "user", or "") in the campaign based on metadata.
func GetUserRoleInCampaign(meta *commonpb.Metadata, userID, ownerID string) string {
	if ownerID != "" && userID == ownerID {
		return "admin"
	}
	if meta != nil && meta.ServiceSpecific != nil {
		ss := meta.ServiceSpecific.AsMap()
		if cmeta, ok := ss["campaign"].(map[string]interface{}); ok {
			if members, ok := cmeta["members"].([]interface{}); ok {
				for _, m := range members {
					if mm, ok := m.(map[string]interface{}); ok {
						if mm["user_id"] == userID {
							if role, ok := mm["role"].(string); ok {
								return role
							}
						}
					}
				}
			}
		}
	}
	return ""
}

// IsSystemCampaign returns true if the campaign is system/ovasabi-created.
func IsSystemCampaign(meta *commonpb.Metadata) bool {
	if meta != nil && meta.ServiceSpecific != nil {
		ss := meta.ServiceSpecific.AsMap()
		if cmeta, ok := ss["campaign"].(map[string]interface{}); ok {
			if v, ok := cmeta["system_created"].(bool); ok && v {
				return true
			}
			if v, ok := cmeta["ovasabi_created"].(bool); ok && v {
				return true
			}
		}
	}
	return false
}

// GetSubscriptionInfo extracts subscription info from campaign metadata.
func GetSubscriptionInfo(meta *commonpb.Metadata) (typ string, price float64, currency, info string) {
	if meta != nil && meta.ServiceSpecific != nil {
		ss := meta.ServiceSpecific.AsMap()
		if cmeta, ok := ss["campaign"].(map[string]interface{}); ok {
			if commerce, ok := cmeta["commerce"].(map[string]interface{}); ok {
				if sub, ok := commerce["subscription"].(map[string]interface{}); ok {
					if t, ok := sub["type"].(string); ok {
						typ = t
					}
					if p, ok := sub["price"].(float64); ok {
						price = p
					}
					if c, ok := sub["currency"].(string); ok {
						currency = c
					}
					if i, ok := sub["info"].(string); ok {
						info = i
					}
				}
			}
		}
	}
	return typ, price, currency, info
}
