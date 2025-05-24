package contentmoderation

import (
	"encoding/json"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// ContentModerationMetadata for robust, extensible moderation metadata.
type Metadata struct {
	Versioning     map[string]interface{} `json:"versioning,omitempty"`
	FlaggedSignals map[string]float64     `json:"flagged_signals,omitempty"`
	Reviewer       *ReviewerMetadata      `json:"reviewer,omitempty"`
	Audit          map[string]interface{} `json:"audit,omitempty"`
	Compliance     map[string]interface{} `json:"compliance,omitempty"`
	Notes          string                 `json:"notes,omitempty"`
}

type ReviewerMetadata struct {
	ReviewerID   string `json:"reviewer_id,omitempty"`
	ReviewerName string `json:"reviewer_name,omitempty"`
	ReviewedAt   string `json:"reviewed_at,omitempty"`
}

// ExtractAndEnrichContentModerationMetadata extracts, validates, and enriches moderation metadata.
func ExtractAndEnrichContentModerationMetadata(meta *commonpb.Metadata, userID string, isCreate bool) (*commonpb.Metadata, error) {
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	var modMeta Metadata
	ss := meta.GetServiceSpecific()
	if ss != nil {
		if m, ok := ss.AsMap()["contentmoderation"]; ok {
			b, err := json.Marshal(m)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(b, &modMeta); err != nil {
				return nil, err
			}
		}
	}
	if modMeta.Versioning == nil {
		modMeta.Versioning = map[string]interface{}{
			"system_version":     "1.0.0",
			"service_version":    "1.0.0",
			"moderation_version": "1.0.0",
			"environment":        "prod",
			"last_migrated_at":   time.Now().Format(time.RFC3339),
		}
	}
	if modMeta.Audit == nil {
		modMeta.Audit = map[string]interface{}{
			"created_by": userID,
			"history":    []string{"created"},
		}
	} else {
		modMeta.Audit["last_modified_by"] = userID
		if isCreate {
			if h, ok := modMeta.Audit["history"].([]string); ok {
				modMeta.Audit["history"] = append(h, "created")
			}
		} else {
			if h, ok := modMeta.Audit["history"].([]string); ok {
				modMeta.Audit["history"] = append(h, "updated")
			}
		}
	}
	if modMeta.Compliance == nil {
		modMeta.Compliance = map[string]interface{}{
			"policy": "platform_default",
		}
	}
	if modMeta.FlaggedSignals == nil {
		modMeta.FlaggedSignals = map[string]float64{}
	}
	if modMeta.Reviewer == nil {
		modMeta.Reviewer = &ReviewerMetadata{}
	}
	m := map[string]interface{}{
		"contentmoderation": modMeta,
	}
	ssStruct, err := structpb.NewStruct(m)
	if err != nil {
		return nil, err
	}
	return &commonpb.Metadata{
		ServiceSpecific: ssStruct,
		Tags:            meta.GetTags(),
	}, nil
}
