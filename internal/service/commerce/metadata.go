// Provider/DI Registration Pattern (Modern, Extensible, DRY)
// ---------------------------------------------------------
//
// This file implements the robust, extensible metadata pattern for the Commerce service.
// It defines the CommerceServiceMetadata struct, which is stored under metadata.service_specific["commerce"]
// in the canonical Metadata proto. This enables dynamic enrichment, orchestration, and analytics
// for all commerce entities (orders, payments, quotes, etc.), and supports payment partner suggestions
// based on locale, country, and currency. All helpers follow the same pattern as productservice/metadata_helpers.go.
//
// Key Features:
// - Extensible, versioned metadata for commerce
// - Payment partner suggestions for different locales/locations
// - Easy marshaling/unmarshaling to/from protobuf Struct and JSONB
// - Helpers for extraction and enrichment
//
// To add new commerce-specific metadata fields, extend CommerceServiceMetadata and update the helpers.

package commerce

// PaymentPartnerMetadata describes a payment partner suggestion for a given context.
type PaymentPartnerMetadata struct {
	PartnerID           string                 `json:"partner_id"`
	Name                string                 `json:"name"`
	SupportedLocales    []string               `json:"supported_locales"`
	SupportedCountries  []string               `json:"supported_countries"`
	SupportedCurrencies []string               `json:"supported_currencies"`
	Priority            int                    `json:"priority"`
	Reason              string                 `json:"reason"`
	Features            map[string]interface{} `json:"features,omitempty"`
	Compliance          map[string]interface{} `json:"compliance,omitempty"`
}

// CommerceServiceMetadata is the service_specific.commerce metadata struct.
type Metadata struct {
	Versioning      map[string]interface{}   `json:"versioning,omitempty"`
	PaymentPartners []PaymentPartnerMetadata `json:"payment_partners,omitempty"`
	PaymentContext  map[string]interface{}   `json:"payment_context,omitempty"`
	FraudSignals    map[string]interface{}   `json:"fraud_signals,omitempty"`
	Analytics       map[string]interface{}   `json:"analytics,omitempty"`
	Audit           map[string]interface{}   `json:"audit,omitempty"`
	Compliance      map[string]interface{}   `json:"compliance,omitempty"`
	Orchestration   map[string]interface{}   `json:"orchestration,omitempty"`
	// Add other commerce-specific fields as needed
}

// [CANONICAL] All state hydration, analytics, and orchestration must use metadata.ExtractServiceVariables(meta, "commerce") directly.
// Do not add local wrappers for metadata extraction—use the canonical helper from pkg/metadata.
// Only business-specific enrichment logic should remain here.
