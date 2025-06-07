package notification

type Metadata struct {
	Channel         string                 `json:"channel,omitempty"`
	Status          string                 `json:"status,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	CampaignID      string                 `json:"campaign_id,omitempty"`
	ServiceSpecific map[string]interface{} `json:"service_specific,omitempty"`
}
