package talent

import (
	"encoding/json"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
)

// ServiceMetadata for talent, with diversity, inclusion, and industry-standard fields.
type Metadata struct {
	Diversity      *DiversityMetadata     `json:"diversity,omitempty"`
	Skills         []string               `json:"skills,omitempty"`
	Certifications []string               `json:"certifications,omitempty"`
	Industry       string                 `json:"industry,omitempty"`
	Languages      []string               `json:"languages,omitempty"`
	Accessibility  *AccessibilityMetadata `json:"accessibility,omitempty"`
	Compliance     *ComplianceMetadata    `json:"compliance,omitempty"`
	Audit          *AuditMetadata         `json:"audit,omitempty"`
	Versioning     map[string]interface{} `json:"versioning,omitempty"`
	Custom         map[string]interface{} `json:"custom,omitempty"`
	Gamified       *GamifiedMetadata      `json:"gamified,omitempty"`
}

type DiversityMetadata struct {
	Gender        string `json:"gender,omitempty"`
	Ethnicity     string `json:"ethnicity,omitempty"`
	Disability    string `json:"disability,omitempty"`
	VeteranStatus string `json:"veteran_status,omitempty"`
	Pronouns      string `json:"pronouns,omitempty"`
	Other         string `json:"other,omitempty"`
}

type AccessibilityMetadata struct {
	Accommodations []string `json:"accommodations,omitempty"`
	Notes          string   `json:"notes,omitempty"`
}

type ComplianceMetadata struct {
	Certifications  []string `json:"certifications,omitempty"`
	CountryOfOrigin string   `json:"country_of_origin,omitempty"`
}

type AuditMetadata struct {
	CreatedBy      string   `json:"created_by,omitempty"`
	LastModifiedBy string   `json:"last_modified_by,omitempty"`
	History        []string `json:"history,omitempty"`
}

type Badge struct {
	Name     string    `json:"name,omitempty"`
	EarnedAt time.Time `json:"earned_at,omitempty"`
}

type Guild struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Rank string `json:"rank,omitempty"`
}

type Party struct {
	ID         string   `json:"id,omitempty"`
	Name       string   `json:"name,omitempty"`
	Role       string   `json:"role,omitempty"`
	CampaignID string   `json:"campaign_id,omitempty"`
	Members    []string `json:"members,omitempty"`
}

type CampaignParticipation struct {
	ID   string `json:"id,omitempty"`
	Role string `json:"role,omitempty"`
}

type GamifiedMetadata struct {
	Level         int                     `json:"level,omitempty"`
	XP            int                     `json:"xp,omitempty"`
	Roles         []string                `json:"roles,omitempty"`
	Badges        []Badge                 `json:"badges,omitempty"`
	Guild         *Guild                  `json:"guild,omitempty"`
	Parties       []Party                 `json:"parties,omitempty"`
	Campaigns     []CampaignParticipation `json:"campaigns,omitempty"`
	TeamworkScore int                     `json:"teamwork_score,omitempty"`
	Skills        []string                `json:"skills,omitempty"`
}

// ExtractAndEnrichTalentMetadata extracts, validates, and enriches talent metadata.
func ExtractAndEnrichTalentMetadata(meta *commonpb.Metadata, userID string, isCreate bool) (*commonpb.Metadata, error) {
	if meta == nil {
		meta = &commonpb.Metadata{
			ServiceSpecific: metadata.NewStructFromMap(map[string]interface{}{"error": "metadata is nil"}, nil),
			Tags:            []string{},
			Features:        []string{},
		}
	}
	var talentMeta Metadata
	ss := meta.ServiceSpecific
	if ss != nil {
		if m, ok := ss.AsMap()["talent"]; ok {
			b, err := json.Marshal(m)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(b, &talentMeta); err != nil {
				return nil, err
			}
		}
	}
	// Ensure versioning
	if talentMeta.Versioning == nil {
		talentMeta.Versioning = map[string]interface{}{
			"system_version":   "1.0.0",
			"service_version":  "1.0.0",
			"talent_version":   "1.0.0",
			"environment":      "prod",
			"last_migrated_at": time.Now().Format(time.RFC3339),
		}
	}
	// Enrich audit
	if talentMeta.Audit == nil {
		talentMeta.Audit = &AuditMetadata{
			CreatedBy: userID,
			History:   []string{"created"},
		}
	} else {
		talentMeta.Audit.LastModifiedBy = userID
		if isCreate {
			talentMeta.Audit.History = append(talentMeta.Audit.History, "created")
		} else {
			talentMeta.Audit.History = append(talentMeta.Audit.History, "updated")
		}
	}
	// Diversity defaults
	if talentMeta.Diversity == nil {
		talentMeta.Diversity = &DiversityMetadata{}
	}
	// Accessibility defaults
	if talentMeta.Accessibility == nil {
		talentMeta.Accessibility = &AccessibilityMetadata{}
	}
	// Compliance defaults
	if talentMeta.Compliance == nil {
		talentMeta.Compliance = &ComplianceMetadata{
			Certifications:  []string{},
			CountryOfOrigin: "Unknown",
		}
	}
	// Skills, Certifications, Languages, Industry: ensure not nil
	if talentMeta.Skills == nil {
		talentMeta.Skills = []string{}
	}
	if talentMeta.Certifications == nil {
		talentMeta.Certifications = []string{}
	}
	if talentMeta.Languages == nil {
		talentMeta.Languages = []string{}
	}
	if talentMeta.Industry == "" {
		talentMeta.Industry = "unspecified"
	}
	// Safe integer handling for any number fields (example: custom fields)
	if talentMeta.Custom != nil {
		for k, v := range talentMeta.Custom {
			if f, ok := v.(float64); ok {
				talentMeta.Custom[k] = utils.ToInt32(int(f))
			}
		}
	}
	// Build and return enriched metadata
	return BuildTalentMetadata(&talentMeta, meta.GetTags())
}

// BuildTalentMetadata builds a commonpb.Metadata from ServiceMetadata and tags.
func BuildTalentMetadata(meta *Metadata, tags []string) (*commonpb.Metadata, error) {
	m, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	var metaMap map[string]interface{}
	if err := json.Unmarshal(m, &metaMap); err != nil {
		return nil, err
	}
	ss := metadata.NewStructFromMap(map[string]interface{}{"talent": metaMap}, nil)
	return &commonpb.Metadata{
		Tags:            tags,
		ServiceSpecific: ss,
	}, nil
}

// ExtractTalentRoles extracts roles from metadata.
func ExtractTalentRoles(meta *commonpb.Metadata) []string {
	if meta == nil || meta.ServiceSpecific == nil {
		return nil
	}
	ss := meta.ServiceSpecific.AsMap()
	talent, ok := ss["talent"].(map[string]interface{})
	if !ok {
		return nil
	}
	gamified, ok := talent["gamified"].(map[string]interface{})
	if !ok {
		return nil
	}
	rolesIface, ok := gamified["roles"]
	if !ok {
		return nil
	}
	roles := []string{}
	switch v := rolesIface.(type) {
	case []interface{}:
		for _, r := range v {
			if s, ok := r.(string); ok {
				roles = append(roles, s)
			}
		}
	case []string:
		roles = v
	}
	return roles
}

// ExtractTalentBadges extracts badges from metadata.
func ExtractTalentBadges(meta *commonpb.Metadata) []Badge {
	if meta == nil || meta.ServiceSpecific == nil {
		return nil
	}
	ss := meta.ServiceSpecific.AsMap()
	talent, ok := ss["talent"].(map[string]interface{})
	if !ok {
		return nil
	}
	gamified, ok := talent["gamified"].(map[string]interface{})
	if !ok {
		return nil
	}
	badgesIface, ok := gamified["badges"]
	if !ok {
		return nil
	}
	badges := []Badge{}
	switch v := badgesIface.(type) {
	case []interface{}:
		for _, b := range v {
			if m, ok := b.(map[string]interface{}); ok {
				badge := Badge{}
				if name, ok := m["name"].(string); ok {
					badge.Name = name
				}
				if earnedAt, ok := m["earned_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, earnedAt); err == nil {
						badge.EarnedAt = t
					}
				}
				badges = append(badges, badge)
			}
		}
	case []Badge:
		badges = v
	}
	return badges
}

// ExtractTalentGuild extracts guild from metadata.
func ExtractTalentGuild(meta *commonpb.Metadata) *Guild {
	if meta == nil || meta.ServiceSpecific == nil {
		return nil
	}
	ss := meta.ServiceSpecific.AsMap()
	talent, ok := ss["talent"].(map[string]interface{})
	if !ok {
		return nil
	}
	gamified, ok := talent["gamified"].(map[string]interface{})
	if !ok {
		return nil
	}
	guildIface, ok := gamified["guild"]
	if !ok {
		return nil
	}
	if m, ok := guildIface.(map[string]interface{}); ok {
		guild := &Guild{}
		if id, ok := m["id"].(string); ok {
			guild.ID = id
		}
		if name, ok := m["name"].(string); ok {
			guild.Name = name
		}
		if rank, ok := m["rank"].(string); ok {
			guild.Rank = rank
		}
		return guild
	}
	return nil
}

// ExtractTalentParties extracts parties from metadata.
func ExtractTalentParties(meta *commonpb.Metadata) []Party {
	if meta == nil || meta.ServiceSpecific == nil {
		return nil
	}
	ss := meta.ServiceSpecific.AsMap()
	talent, ok := ss["talent"].(map[string]interface{})
	if !ok {
		return nil
	}
	gamified, ok := talent["gamified"].(map[string]interface{})
	if !ok {
		return nil
	}
	partiesIface, ok := gamified["parties"]
	if !ok {
		return nil
	}
	parties := []Party{}
	switch v := partiesIface.(type) {
	case []interface{}:
		for _, p := range v {
			m, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			party := Party{}
			if id, ok := m["id"].(string); ok {
				party.ID = id
			}
			if name, ok := m["name"].(string); ok {
				party.Name = name
			}
			if role, ok := m["role"].(string); ok {
				party.Role = role
			}
			if campaignID, ok := m["campaign_id"].(string); ok {
				party.CampaignID = campaignID
			}
			if members, ok := m["members"].([]interface{}); ok {
				for _, mem := range members {
					if s, ok := mem.(string); ok {
						party.Members = append(party.Members, s)
					}
				}
			}
			parties = append(parties, party)
		}
	case []Party:
		parties = v
	}
	return parties
}

// ExtractTalentLevel extracts level from metadata.
func ExtractTalentLevel(meta *commonpb.Metadata) int {
	if meta == nil || meta.ServiceSpecific == nil {
		return 0
	}
	ss := meta.ServiceSpecific.AsMap()
	talent, ok := ss["talent"].(map[string]interface{})
	if !ok {
		return 0
	}
	gamified, ok := talent["gamified"].(map[string]interface{})
	if !ok {
		return 0
	}
	levelIface, ok := gamified["level"]
	if !ok {
		return 0
	}
	switch v := levelIface.(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// ExtractTalentTeamworkScore extracts teamwork score from metadata.
func ExtractTalentTeamworkScore(meta *commonpb.Metadata) int {
	if meta == nil || meta.ServiceSpecific == nil {
		return 0
	}
	ss := meta.ServiceSpecific.AsMap()
	talent, ok := ss["talent"].(map[string]interface{})
	if !ok {
		return 0
	}
	gamified, ok := talent["gamified"].(map[string]interface{})
	if !ok {
		return 0
	}
	scoreIface, ok := gamified["teamwork_score"]
	if !ok {
		return 0
	}
	switch v := scoreIface.(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}
