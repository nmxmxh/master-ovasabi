package talent

import (
	"encoding/json"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
	"google.golang.org/protobuf/types/known/structpb"
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

// ExtractAndEnrichTalentMetadata extracts, validates, and enriches talent metadata.
func ExtractAndEnrichTalentMetadata(meta *commonpb.Metadata, userID string, isCreate bool) (*commonpb.Metadata, error) {
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	var talentMeta Metadata
	ss := meta.GetServiceSpecific()
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
	ss, err := structpb.NewStruct(map[string]interface{}{"talent": metaMap})
	if err != nil {
		return nil, err
	}
	return &commonpb.Metadata{
		Tags:            tags,
		ServiceSpecific: ss,
	}, nil
}
