package notification

import (
	"encoding/json"

	structpb "google.golang.org/protobuf/types/known/structpb"
)

type Metadata struct {
	Channel         string                 `json:"channel,omitempty"`
	Status          string                 `json:"status,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	CampaignID      string                 `json:"campaign_id,omitempty"`
	ServiceSpecific map[string]interface{} `json:"service_specific,omitempty"`
}

func MetadataFromStruct(s *structpb.Struct) (*Metadata, error) {
	if s == nil {
		return &Metadata{}, nil
	}
	b, err := json.Marshal(s.AsMap())
	if err != nil {
		return nil, err
	}
	var meta Metadata
	err = json.Unmarshal(b, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func MetadataToStruct(meta *Metadata) (*structpb.Struct, error) {
	if meta == nil {
		return structpb.NewStruct(map[string]interface{}{})
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return structpb.NewStruct(m)
}
