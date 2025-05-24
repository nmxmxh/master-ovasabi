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

import (
	"encoding/json"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

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

// ExtractAndEnrichCommerceMetadata extracts, validates, and enriches commerce metadata.
func ExtractAndEnrichCommerceMetadata(log *zap.Logger, meta *commonpb.Metadata, userID string, isCreate bool) (*commonpb.Metadata, error) {
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	var commMeta Metadata
	ss := meta.GetServiceSpecific()
	if ss != nil {
		if m, ok := ss.AsMap()["commerce"]; ok {
			b, err := json.Marshal(m)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(b, &commMeta); err != nil {
				return nil, err
			}
		}
	}
	// Ensure versioning
	if commMeta.Versioning == nil {
		commMeta.Versioning = map[string]interface{}{
			"system_version":   "1.0.0",
			"service_version":  "1.0.0",
			"commerce_version": "1.0.0",
			"environment":      "prod",
			"last_migrated_at": time.Now().Format(time.RFC3339),
		}
	}
	// Ensure audit
	if commMeta.Audit == nil {
		commMeta.Audit = map[string]interface{}{
			"created_by": userID,
			"history":    []string{"created"},
		}
	} else {
		commMeta.Audit["last_modified_by"] = userID
		if isCreate {
			if h, ok := commMeta.Audit["history"].([]string); ok {
				commMeta.Audit["history"] = append(h, "created")
			}
		} else {
			if h, ok := commMeta.Audit["history"].([]string); ok {
				commMeta.Audit["history"] = append(h, "updated")
			}
		}
	}
	// Ensure compliance
	if commMeta.Compliance == nil {
		commMeta.Compliance = map[string]interface{}{
			"certifications":    []string{"PCI DSS"},
			"country_of_origin": "Unknown",
		}
	}
	// Example: If certifications are missing, set a default
	if certs, ok := commMeta.Compliance["certifications"].([]string); !ok || len(certs) == 0 {
		commMeta.Compliance["certifications"] = []string{"PCI DSS"}
	}
	// Example: Mark as compliant if certifications include 'PCI DSS' or 'GDPR'
	isCompliant := false
	if certs, ok := commMeta.Compliance["certifications"].([]string); ok {
		for _, cert := range certs {
			if cert == "PCI DSS" || cert == "GDPR" {
				isCompliant = true
				break
			}
		}
	}
	commMeta.Compliance["compliant"] = isCompliant
	// Ensure fraud signals
	if commMeta.FraudSignals == nil {
		commMeta.FraudSignals = map[string]interface{}{
			"risk_score": 0.0,
			"signals":    []string{},
		}
	}
	// Example: If amount in payment context is high, flag as high risk
	if commMeta.PaymentContext != nil {
		if amt, ok := commMeta.PaymentContext["amount"].(float64); ok && amt > 10000 {
			commMeta.FraudSignals["risk_score"] = 0.9
			if signals, ok := commMeta.FraudSignals["signals"].([]string); ok {
				commMeta.FraudSignals["signals"] = append(signals, "high_amount")
			}
		}
	}
	// Ensure analytics
	if commMeta.Analytics == nil {
		commMeta.Analytics = map[string]interface{}{
			"event_count": 0,
			"last_event":  "",
		}
	}
	// Example: Increment event count if present
	if cnt, ok := commMeta.Analytics["event_count"].(int); ok {
		commMeta.Analytics["event_count"] = cnt + 1
	}
	// Ensure orchestration
	if commMeta.Orchestration == nil {
		commMeta.Orchestration = map[string]interface{}{
			"workflow": "default",
		}
	}
	// Example enrichment: If no payment partners, suggest defaults based on context
	if len(commMeta.PaymentPartners) == 0 && commMeta.PaymentContext != nil {
		var country, currency string
		if v, ok := commMeta.PaymentContext["user_country"].(string); ok {
			country = v
		} else {
			log.Warn("user_country type assertion failed in payment context", zap.Any("value", commMeta.PaymentContext["user_country"]))
		}
		if v, ok := commMeta.PaymentContext["currency"].(string); ok {
			currency = v
		} else {
			log.Warn("currency type assertion failed in payment context", zap.Any("value", commMeta.PaymentContext["currency"]))
		}
		// Example: Suggest Stripe for US/EU, M-Pesa for KE, PayPal for global
		if country == "KE" || currency == "KES" {
			commMeta.PaymentPartners = append(commMeta.PaymentPartners, PaymentPartnerMetadata{
				PartnerID:           "mpesa",
				Name:                "M-Pesa",
				SupportedLocales:    []string{"sw-KE", "en-KE"},
				SupportedCountries:  []string{"KE", "TZ"},
				SupportedCurrencies: []string{"KES"},
				Priority:            1,
				Reason:              "Mobile money leader in Kenya/Tanzania",
				Features:            map[string]interface{}{"mobile_money": true, "cash_in": true, "cash_out": true},
				Compliance:          map[string]interface{}{"cbk_kenya": true},
			})
		}
		if country == "US" || country == "FR" || currency == "USD" || currency == "EUR" {
			commMeta.PaymentPartners = append(commMeta.PaymentPartners, PaymentPartnerMetadata{
				PartnerID:           "stripe",
				Name:                "Stripe",
				SupportedLocales:    []string{"en-US", "fr-FR"},
				SupportedCountries:  []string{"US", "FR", "GB", "DE"},
				SupportedCurrencies: []string{"USD", "EUR", "GBP"},
				Priority:            1,
				Reason:              "Preferred for USD/EUR payments in US/EU",
				Features:            map[string]interface{}{"instant_payouts": true, "recurring_payments": true, "apple_pay": true},
				Compliance:          map[string]interface{}{"pci_dss": true, "gdpr": true},
			})
		}
		// Always suggest PayPal as a fallback
		commMeta.PaymentPartners = append(commMeta.PaymentPartners, PaymentPartnerMetadata{
			PartnerID:           "paypal",
			Name:                "PayPal",
			SupportedLocales:    []string{"en-US", "fr-FR", "es-ES"},
			SupportedCountries:  []string{"US", "FR", "ES", "NG"},
			SupportedCurrencies: []string{"USD", "EUR", "NGN"},
			Priority:            2,
			Reason:              "Widely used for international payments",
			Features:            map[string]interface{}{"buyer_protection": true, "multi_currency": true},
			Compliance:          map[string]interface{}{"pci_dss": true},
		})
	}
	return BuildCommerceMetadata(&commMeta, meta.GetTags())
}

// BuildCommerceMetadata builds a *commonpb.Metadata from CommerceServiceMetadata and tags.
func BuildCommerceMetadata(commMeta *Metadata, tags []string) (*commonpb.Metadata, error) {
	m := map[string]interface{}{
		"commerce": commMeta,
	}
	ss, err := structpb.NewStruct(m)
	if err != nil {
		return nil, err
	}
	return &commonpb.Metadata{
		ServiceSpecific: ss,
		Tags:            tags,
	}, nil
}

// ExtractCommerceServiceMetadata extracts CommerceServiceMetadata from a Metadata proto.
func ExtractCommerceServiceMetadata(meta *commonpb.Metadata) (*Metadata, error) {
	if meta == nil || meta.ServiceSpecific == nil {
		return &Metadata{}, nil
	}
	ss := meta.ServiceSpecific.AsMap()
	m, ok := ss["commerce"]
	if !ok {
		return &Metadata{}, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var commMeta Metadata
	if err := json.Unmarshal(b, &commMeta); err != nil {
		return nil, err
	}
	return &commMeta, nil
}
