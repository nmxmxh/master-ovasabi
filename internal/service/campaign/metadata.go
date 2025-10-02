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
	Custom *structpb.Struct `json:"custom"`
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

// [CANONICAL] All metadata must be normalized and calculated via metadata.NormalizeAndCalculate before persistence or emission.
// Ensure required fields (versioning, audit, etc.) are present under the correct namespace.

// CanonicalizeFromProto extracts, fills, and validates a canonical Metadata struct from *commonpb.Metadata.
func CanonicalizeFromProto(meta *commonpb.Metadata, fallbackID string) (*Metadata, error) {
	m := &Metadata{}
	// Set defaults
	m.ID = fallbackID
	m.Type = CampaignTypeScheduled
	m.Status = CampaignStatusInactive
	m.Versioning = &VersioningInfo{
		SystemVersion:  "1.0.0",
		ServiceVersion: "1.0.0",
		Environment:    "production",
		LastMigratedAt: time.Now(),
	}
	m.Scheduling = &SchedulingInfo{}
	m.Features = []string{}
	m.Localization = &LocalizationInfo{}
	m.Content = &ContentInfo{}
	m.Community = &CommunityInfo{}
	m.Onboarding = &OnboardingInfo{}
	m.Referral = &ReferralInfo{}
	m.Commerce = &CommerceInfo{}
	m.Analytics = &AnalyticsInfo{}
	m.Compliance = &ComplianceInfo{}
	m.Custom = &structpb.Struct{}

	if meta != nil && meta.ServiceSpecific != nil {
		ss := meta.ServiceSpecific.AsMap()
		if cmeta, ok := ss["campaign"].(map[string]interface{}); ok {
			if id, ok := cmeta["id"].(string); ok && id != "" {
				m.ID = id
			}
			if typ, ok := cmeta["type"].(string); ok && typ != "" {
				m.Type = typ
			}
			if status, ok := cmeta["status"].(string); ok && status != "" {
				m.Status = status
			}
			if v, ok := cmeta["versioning"].(map[string]interface{}); ok {
				if m.Versioning == nil {
					m.Versioning = &VersioningInfo{}
				}
				if sv, ok := v["system_version"].(string); ok {
					m.Versioning.SystemVersion = sv
				}
				if sv, ok := v["service_version"].(string); ok {
					m.Versioning.ServiceVersion = sv
				}
				if env, ok := v["environment"].(string); ok {
					m.Versioning.Environment = env
				}
				if lm, ok := v["last_migrated_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, lm); err == nil {
						m.Versioning.LastMigratedAt = t
					}
				}
			}
			if sched, ok := cmeta["scheduling"].(map[string]interface{}); ok {
				if m.Scheduling == nil {
					m.Scheduling = &SchedulingInfo{}
				}
				if start, ok := sched["start"].(string); ok {
					if t, err := time.Parse(time.RFC3339, start); err == nil {
						m.Scheduling.Start = t
					}
				}
				if end, ok := sched["end"].(string); ok {
					if t, err := time.Parse(time.RFC3339, end); err == nil {
						m.Scheduling.End = t
					}
				}
				if rec, ok := sched["recurrence"].(string); ok {
					m.Scheduling.Recurrence = rec
				}
				if jobs, ok := sched["jobs"].([]interface{}); ok {
					for _, job := range jobs {
						if jm, ok := job.(map[string]interface{}); ok {
							m.Scheduling.Jobs = append(m.Scheduling.Jobs, jm)
						}
					}
				}
			}
			if feats, ok := cmeta["features"].([]interface{}); ok {
				m.Features = []string{}
				for _, f := range feats {
					if fs, ok := f.(string); ok {
						m.Features = append(m.Features, fs)
					}
				}
			}
			// START: Added parsing for all other fields
			if loc, ok := cmeta["localization"].(map[string]interface{}); ok {
				if m.Localization == nil {
					m.Localization = &LocalizationInfo{}
				}
				if sl, ok := loc["supported_locales"].([]interface{}); ok {
					for _, l := range sl {
						if ls, ok := l.(string); ok {
							m.Localization.SupportedLocales = append(m.Localization.SupportedLocales, ls)
						}
					}
				}
				if dl, ok := loc["default_locale"].(string); ok {
					m.Localization.DefaultLocale = dl
				}
				if tr, ok := loc["translations"].(map[string]interface{}); ok {
					m.Localization.Translations = make(map[string]map[string]string)
					for lang, keys := range tr {
						if keyMap, ok := keys.(map[string]interface{}); ok {
							m.Localization.Translations[lang] = make(map[string]string)
							for key, val := range keyMap {
								if vs, ok := val.(string); ok {
									m.Localization.Translations[lang][key] = vs
								}
							}
						}
					}
				}
			}
			if cont, ok := cmeta["content"].(map[string]interface{}); ok {
				if m.Content == nil {
					m.Content = &ContentInfo{}
				}
				if types, ok := cont["types"].([]interface{}); ok {
					for _, t := range types {
						if ts, ok := t.(string); ok {
							m.Content.Types = append(m.Content.Types, ts)
						}
					}
				}
				if tmpl, ok := cont["templates"].(map[string]interface{}); ok {
					m.Content.Templates = tmpl
				}
				if mod, ok := cont["moderation"].(string); ok {
					m.Content.Moderation = mod
				}
			}
			if comm, ok := cmeta["community"].(map[string]interface{}); ok {
				if m.Community == nil {
					m.Community = &CommunityInfo{}
				}
				if secs, ok := comm["sections"].([]interface{}); ok {
					for _, s := range secs {
						if ss, ok := s.(string); ok {
							m.Community.Sections = append(m.Community.Sections, ss)
						}
					}
				}
				if lb, ok := comm["leaderboard"].(map[string]interface{}); ok {
					m.Community.Leaderboard = lb
				}
				if chat, ok := comm["chat"].(map[string]interface{}); ok {
					m.Community.Chat = chat
				}
			}
			if ob, ok := cmeta["onboarding"].(map[string]interface{}); ok {
				if m.Onboarding == nil {
					m.Onboarding = &OnboardingInfo{}
				}
				if it, ok := ob["interest_types"].([]interface{}); ok {
					for _, i := range it {
						if is, ok := i.(string); ok {
							m.Onboarding.InterestTypes = append(m.Onboarding.InterestTypes, is)
						}
					}
				}
				if q, ok := ob["questionnaire"].(map[string]interface{}); ok {
					m.Onboarding.Questionnaire = make(map[string][]map[string]map[string]interface{})
					for key, val := range q {
						if valSlice, ok := val.([]interface{}); ok {
							var questions []map[string]map[string]interface{}
							for _, item := range valSlice {
								if question, ok := item.(map[string]interface{}); ok {
									qMap := make(map[string]map[string]interface{})
									for qKey, qVal := range question {
										if qValMap, ok := qVal.(map[string]interface{}); ok {
											qMap[qKey] = qValMap
										}
									}
									questions = append(questions, qMap)
								}
							}
							m.Onboarding.Questionnaire[key] = questions
						}
					}
				}
			}
			if ref, ok := cmeta["referral"].(map[string]interface{}); ok {
				if m.Referral == nil {
					m.Referral = &ReferralInfo{}
				}
				if en, ok := ref["enabled"].(bool); ok {
					m.Referral.Enabled = en
				}
				if rew, ok := ref["reward"].(map[string]interface{}); ok {
					m.Referral.Reward = rew
				}
			}
			if com, ok := cmeta["commerce"].(map[string]interface{}); ok {
				if m.Commerce == nil {
					m.Commerce = &CommerceInfo{}
				}
				if pm, ok := com["payment_methods"].([]interface{}); ok {
					for _, p := range pm {
						if ps, ok := p.(string); ok {
							m.Commerce.PaymentMethods = append(m.Commerce.PaymentMethods, ps)
						}
					}
				}
				if book, ok := com["booking"].(map[string]interface{}); ok {
					m.Commerce.Booking = book
				}
			}
			if an, ok := cmeta["analytics"].(map[string]interface{}); ok {
				if m.Analytics == nil {
					m.Analytics = &AnalyticsInfo{}
				}
				if te, ok := an["track_events"].([]interface{}); ok {
					for _, e := range te {
						if es, ok := e.(string); ok {
							m.Analytics.TrackEvents = append(m.Analytics.TrackEvents, es)
						}
					}
				}
				if goals, ok := an["goals"].(map[string]interface{}); ok {
					m.Analytics.Goals = goals
				}
			}
			if comp, ok := cmeta["compliance"].(map[string]interface{}); ok {
				if m.Compliance == nil {
					m.Compliance = &ComplianceInfo{}
				}
				if acc, ok := comp["accessibility"].(map[string]interface{}); ok {
					m.Compliance.Accessibility = acc
				}
				if leg, ok := comp["legal"].(map[string]interface{}); ok {
					m.Compliance.Legal = leg
				}
			}
			if custom, ok := cmeta["custom"].(map[string]interface{}); ok {
				if s, err := structpb.NewStruct(custom); err == nil {
					m.Custom = s
				}
			}
			// END: Added parsing for all other fields
		}
	}
	// Validate before returning
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return m, nil
}

// ToProto converts a canonical Metadata struct to *commonpb.Metadata under service_specific.campaign.
func ToProto(m *Metadata) *commonpb.Metadata {
	campaignMap := map[string]interface{}{
		"id":     m.ID,
		"type":   m.Type,
		"status": m.Status,
	}
	if m.Versioning != nil {
		campaignMap["versioning"] = map[string]interface{}{
			"system_version":   m.Versioning.SystemVersion,
			"service_version":  m.Versioning.ServiceVersion,
			"environment":      m.Versioning.Environment,
			"last_migrated_at": m.Versioning.LastMigratedAt.Format(time.RFC3339),
		}
	}
	if m.Scheduling != nil {
		campaignMap["scheduling"] = map[string]interface{}{
			"start":      m.Scheduling.Start.Format(time.RFC3339),
			"end":        m.Scheduling.End.Format(time.RFC3339),
			"recurrence": m.Scheduling.Recurrence,
			"jobs":       m.Scheduling.Jobs,
		}
	}
	if len(m.Features) > 0 {
		campaignMap["features"] = m.Features
	}
	// START: Added serialization for all other fields
	if m.Localization != nil {
		campaignMap["localization"] = map[string]interface{}{
			"supported_locales": m.Localization.SupportedLocales,
			"default_locale":    m.Localization.DefaultLocale,
			"translations":      m.Localization.Translations,
		}
	}
	if m.Content != nil {
		campaignMap["content"] = map[string]interface{}{
			"types":      m.Content.Types,
			"templates":  m.Content.Templates,
			"moderation": m.Content.Moderation,
		}
	}
	if m.Community != nil {
		campaignMap["community"] = map[string]interface{}{
			"sections":    m.Community.Sections,
			"leaderboard": m.Community.Leaderboard,
			"chat":        m.Community.Chat,
		}
	}
	if m.Onboarding != nil {
		campaignMap["onboarding"] = map[string]interface{}{
			"interest_types": m.Onboarding.InterestTypes,
			"questionnaire":  m.Onboarding.Questionnaire,
		}
	}
	if m.Referral != nil {
		campaignMap["referral"] = map[string]interface{}{
			"enabled": m.Referral.Enabled,
			"reward":  m.Referral.Reward,
		}
	}
	if m.Commerce != nil {
		campaignMap["commerce"] = map[string]interface{}{
			"payment_methods": m.Commerce.PaymentMethods,
			"booking":         m.Commerce.Booking,
		}
	}
	if m.Analytics != nil {
		campaignMap["analytics"] = map[string]interface{}{
			"track_events": m.Analytics.TrackEvents,
			"goals":        m.Analytics.Goals,
		}
	}
	if m.Compliance != nil {
		campaignMap["compliance"] = map[string]interface{}{
			"accessibility": m.Compliance.Accessibility,
			"legal":         m.Compliance.Legal,
		}
	}
	if m.Custom != nil {
		campaignMap["custom"] = m.Custom.AsMap()
	}
	// END: Added serialization for all other fields
	ss := map[string]interface{}{"campaign": campaignMap}
	return &commonpb.Metadata{
		ServiceSpecific: metadata.NewStructFromMap(ss, nil),
	}
}
