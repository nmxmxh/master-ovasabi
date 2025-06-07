package talent

import (
	"time"
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
