package waitlist

// Metadata structures for waitlist service-specific metadata

// Metadata represents the waitlist service-specific metadata structure.
type Metadata struct {
	Campaign      string             `json:"campaign,omitempty"`
	Tier          string             `json:"tier,omitempty"`
	Location      *LocationMetadata  `json:"location,omitempty"`
	Analytics     *AnalyticsMetadata `json:"analytics,omitempty"`
	Referral      *ReferralMetadata  `json:"referral,omitempty"`
	Accessibility interface{}        `json:"accessibility,omitempty"`
	Compliance    interface{}        `json:"compliance,omitempty"`
	Audit         interface{}        `json:"audit,omitempty"`
	Versioning    interface{}        `json:"versioning,omitempty"`
	Custom        interface{}        `json:"custom,omitempty"`
	Gamified      interface{}        `json:"gamified,omitempty"`
}

// LocationMetadata represents location-specific metadata.
type LocationMetadata struct {
	Country *string  `json:"country,omitempty"`
	Region  *string  `json:"region,omitempty"`
	City    *string  `json:"city,omitempty"`
	Lat     *float64 `json:"lat,omitempty"`
	Lng     *float64 `json:"lng,omitempty"`
}

// AnalyticsMetadata represents analytics tracking metadata.
type AnalyticsMetadata struct {
	IPAddress   *string `json:"ip_address,omitempty"`
	UserAgent   *string `json:"user_agent,omitempty"`
	ReferrerURL *string `json:"referrer_url,omitempty"`
	UTMSource   *string `json:"utm_source,omitempty"`
	UTMMedium   *string `json:"utm_medium,omitempty"`
	UTMCampaign *string `json:"utm_campaign,omitempty"`
	UTMTerm     *string `json:"utm_term,omitempty"`
	UTMContent  *string `json:"utm_content,omitempty"`
}

// ReferralMetadata represents referral-specific metadata.
type ReferralMetadata struct {
	ReferralUsername *string `json:"referral_username,omitempty"`
	ReferralCode     *string `json:"referral_code,omitempty"`
}
