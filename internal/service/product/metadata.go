// Product Metadata Builder (Service-Specific Standard)
// ---------------------------------------------------
// This file implements the canonical builder and struct for product metadata.
// All service-specific metadata builders (e.g., BuildProductMetadata) must:
//   - Be implemented in their respective service packages (not in pkg/metadata)
//   - Follow the extensible, versioned, and namespaced pattern
//   - Be referenced in docs/services/metadata.md and onboarding docs
//   - Reference docs/amadeus/amadeus_context.md for cross-service standards
//
// This is the standard for all service-specific metadata in the OVASABI platform.

package product

import (
	"encoding/json"
	"fmt"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// ServiceMetadata holds all product service-specific metadata fields (Amazon-style, extensible).
type ServiceMetadata struct {
	Versioning     map[string]interface{} `json:"versioning,omitempty"`
	Identifiers    *IdentifiersMetadata   `json:"identifiers,omitempty"`
	Category       *CategoryMetadata      `json:"category,omitempty"`
	Brand          string                 `json:"brand,omitempty"`
	Model          string                 `json:"model,omitempty"`
	Dimensions     *DimensionsMetadata    `json:"dimensions,omitempty"`
	Materials      []string               `json:"materials,omitempty"`
	Color          []string               `json:"color,omitempty"`
	Images         []MediaMetadata        `json:"images,omitempty"`
	Videos         []MediaMetadata        `json:"videos,omitempty"`
	Features       []string               `json:"features,omitempty"`
	Specifications map[string]interface{} `json:"specifications,omitempty"`
	Warranty       *WarrantyMetadata      `json:"warranty,omitempty"`
	Compliance     *ComplianceMetadata    `json:"compliance,omitempty"`
	Availability   *AvailabilityMetadata  `json:"availability,omitempty"`
	Pricing        *PricingMetadata       `json:"pricing,omitempty"`
	Shipping       *ShippingMetadata      `json:"shipping,omitempty"`
	Reviews        *ReviewsMetadata       `json:"reviews,omitempty"`
	BadActor       *BadActorMetadata      `json:"bad_actor,omitempty"`
	Audit          *AuditMetadata         `json:"audit,omitempty"`
	// Extensible: add more fields as needed (e.g., localization, accessibility, custom_rules)
}

type IdentifiersMetadata struct {
	ASIN string `json:"asin,omitempty"`
	UPC  string `json:"upc,omitempty"`
	EAN  string `json:"ean,omitempty"`
	SKU  string `json:"sku,omitempty"`
}

type CategoryMetadata struct {
	Main          string   `json:"main,omitempty"`
	Subcategories []string `json:"subcategories,omitempty"`
}

type DimensionsMetadata struct {
	LengthCM float64 `json:"length_cm,omitempty"`
	WidthCM  float64 `json:"width_cm,omitempty"`
	HeightCM float64 `json:"height_cm,omitempty"`
	WeightKG float64 `json:"weight_kg,omitempty"`
}

type MediaMetadata struct {
	URL  string `json:"url,omitempty"`
	Alt  string `json:"alt,omitempty"`
	Type string `json:"type,omitempty"` // e.g., demo, unboxing
}

type WarrantyMetadata struct {
	Type           string `json:"type,omitempty"`
	DurationMonths int    `json:"duration_months,omitempty"`
	Details        string `json:"details,omitempty"`
}

type ComplianceMetadata struct {
	Certifications  []string `json:"certifications,omitempty"`
	CountryOfOrigin string   `json:"country_of_origin,omitempty"`
}

type AvailabilityMetadata struct {
	InStock     bool   `json:"in_stock,omitempty"`
	StockLevel  int    `json:"stock_level,omitempty"`
	RestockDate string `json:"restock_date,omitempty"`
}

type PricingMetadata struct {
	MSRP          float64 `json:"msrp,omitempty"`
	CurrentPrice  float64 `json:"current_price,omitempty"`
	Currency      string  `json:"currency,omitempty"`
	Discount      float64 `json:"discount,omitempty"`
	DiscountType  string  `json:"discount_type,omitempty"`
	PrimeEligible bool    `json:"prime_eligible,omitempty"`
}

type ShippingMetadata struct {
	WeightKG        float64   `json:"weight_kg,omitempty"`
	DimensionsCM    []float64 `json:"dimensions_cm,omitempty"`
	ShipsFrom       string    `json:"ships_from,omitempty"`
	ShippingMethods []string  `json:"shipping_methods,omitempty"`
}

type ReviewsMetadata struct {
	AverageRating float64    `json:"average_rating,omitempty"`
	ReviewCount   int        `json:"review_count,omitempty"`
	TopReview     *TopReview `json:"top_review,omitempty"`
}

type TopReview struct {
	UserID string `json:"user_id,omitempty"`
	Rating int    `json:"rating,omitempty"`
	Title  string `json:"title,omitempty"`
	Body   string `json:"body,omitempty"`
	Date   string `json:"date,omitempty"`
}

type BadActorMetadata struct {
	Score  float64 `json:"score,omitempty"`
	Reason string  `json:"reason,omitempty"`
}

type AuditMetadata struct {
	CreatedBy      string   `json:"created_by,omitempty"`
	LastModifiedBy string   `json:"last_modified_by,omitempty"`
	History        []string `json:"history,omitempty"`
}

// ServiceMetadataFromStruct converts a structpb.Struct to ServiceMetadata.
func ServiceMetadataFromStruct(s *structpb.Struct) (*ServiceMetadata, error) {
	if s == nil {
		return &ServiceMetadata{}, nil
	}
	b, err := json.Marshal(s.AsMap())
	if err != nil {
		return nil, err
	}
	var meta ServiceMetadata
	err = json.Unmarshal(b, &meta)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

// ServiceMetadataToStruct converts ServiceMetadata to structpb.Struct.
func ServiceMetadataToStruct(meta *ServiceMetadata) (*structpb.Struct, error) {
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

// BuildProductMetadata builds a canonical product metadata struct for storage, analytics, and extensibility.
func BuildProductMetadata(
	meta *ServiceMetadata,
	tags []string,
) (*commonpb.Metadata, error) {
	productMap := map[string]interface{}{}
	b, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal product metadata: %w", err)
	}
	err = json.Unmarshal(b, &productMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal product metadata: %w", err)
	}
	// Always require versioning for compliance
	if _, ok := productMap["versioning"]; !ok {
		productMap["versioning"] = map[string]interface{}{"system_version": "1.0.0"}
	}
	ss := map[string]interface{}{"product": productMap}
	ssStruct, err := structpb.NewStruct(ss)
	if err != nil {
		return nil, fmt.Errorf("failed to build service_specific struct: %w", err)
	}
	return &commonpb.Metadata{
		ServiceSpecific: ssStruct,
		Tags:            tags,
	}, nil
}
