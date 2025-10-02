package product

import (
	"encoding/json"
	"time"

	commonpb "github.com/nmxmxh/master-ovasabi/api/protos/common/v1"
	"github.com/nmxmxh/master-ovasabi/pkg/metadata"
	"github.com/nmxmxh/master-ovasabi/pkg/utils"
)

// ExtractAndEnrichProductMetadata extracts, validates, and enriches product metadata.
func ExtractAndEnrichProductMetadata(meta *commonpb.Metadata, userID string, isCreate bool) (*commonpb.Metadata, error) {
	if meta == nil {
		meta = &commonpb.Metadata{}
	}
	// Extract service-specific.product metadata
	var prodMeta ServiceMetadata
	ss := meta.GetServiceSpecific()
	if ss != nil {
		if m, ok := ss.AsMap()["product"]; ok {
			b, err := json.Marshal(m)
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(b, &prodMeta); err != nil {
				return nil, err
			}
		}
	}
	// Ensure versioning
	if prodMeta.Versioning == nil {
		prodMeta.Versioning = map[string]interface{}{
			"system_version":   "1.0.0",
			"service_version":  "1.0.0",
			"product_version":  "1.0.0",
			"environment":      "prod",
			"last_migrated_at": time.Now().Format(time.RFC3339),
		}
	}
	// Enrich audit
	if prodMeta.Audit == nil {
		prodMeta.Audit = &AuditMetadata{
			CreatedBy: userID,
			History:   []string{"created"},
		}
	} else {
		prodMeta.Audit.LastModifiedBy = userID
		if isCreate {
			prodMeta.Audit.History = append(prodMeta.Audit.History, "created")
		} else {
			prodMeta.Audit.History = append(prodMeta.Audit.History, "updated")
		}
	}
	// Auto-calculate discount (float math is safe, but clamp to int if needed elsewhere)
	if prodMeta.Pricing != nil && prodMeta.Pricing.MSRP > 0 && prodMeta.Pricing.CurrentPrice > 0 {
		discount := 100 * (prodMeta.Pricing.MSRP - prodMeta.Pricing.CurrentPrice) / prodMeta.Pricing.MSRP
		prodMeta.Pricing.Discount = discount
		prodMeta.Pricing.DiscountType = "percentage"
	}
	// Use safeint for integer fields
	if prodMeta.Availability != nil {
		prodMeta.Availability.StockLevel = int(utils.ToInt32(prodMeta.Availability.StockLevel))
	}
	if prodMeta.Reviews != nil {
		prodMeta.Reviews.ReviewCount = int(utils.ToInt32(prodMeta.Reviews.ReviewCount))
	}
	// Compliance enrichment
	if prodMeta.Compliance == nil {
		prodMeta.Compliance = &ComplianceMetadata{
			Certifications:  []string{},
			CountryOfOrigin: "Unknown",
		}
	}
	// Example: If certifications are missing, set a default
	if len(prodMeta.Compliance.Certifications) == 0 {
		prodMeta.Compliance.Certifications = []string{"CE"}
	}
	// Example: Mark as compliant if certifications include 'CE' or 'FCC'
	isCompliant := false
	for _, cert := range prodMeta.Compliance.Certifications {
		if cert == "CE" || cert == "FCC" {
			isCompliant = true
			break
		}
	}
	// Add a compliance flag to Specifications if not present
	if prodMeta.Specifications == nil {
		prodMeta.Specifications = map[string]interface{}{}
	}
	prodMeta.Specifications["compliant"] = isCompliant

	// Review enrichment
	if prodMeta.Reviews != nil {
		// If only one review, set average_rating to top_review.rating
		if prodMeta.Reviews.ReviewCount == 1 && prodMeta.Reviews.TopReview != nil {
			prodMeta.Reviews.AverageRating = float64(prodMeta.Reviews.TopReview.Rating)
		}
		// If average_rating is missing or zero, set a default (e.g., 5.0)
		if prodMeta.Reviews.AverageRating == 0 {
			prodMeta.Reviews.AverageRating = 5.0
		}
	} else {
		// If no reviews, set a default reviews struct
		prodMeta.Reviews = &ReviewsMetadata{
			AverageRating: 5.0,
			ReviewCount:   0,
		}
	}
	// Build and normalize metadata for persistence/emission
	// NOTE: Enforce system-wide normalization and calculation before returning.
	// If you have a product ID, use it for prev/next/related; otherwise, leave as empty.
	metaMap := metadata.ProtoToMap(meta)
	// TODO: Replace "" with actual prev/next/related IDs if available
	metaProto := metadata.MapToProto(metaMap)
	metadata.Handler{}.NormalizeAndCalculate(metaProto, "", "", []string{}, "success", "enrich product metadata")
	return metaProto, nil
}

// ExtractProductServiceMetadata extracts product metadata as ServiceMetadata from a Metadata proto.
func ExtractProductServiceMetadata(meta *commonpb.Metadata) (*ServiceMetadata, error) {
	if meta == nil || meta.ServiceSpecific == nil {
		return &ServiceMetadata{}, nil
	}
	ss := meta.ServiceSpecific.AsMap()
	m, ok := ss["product"]
	if !ok {
		return &ServiceMetadata{}, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var prodMeta ServiceMetadata
	if err := json.Unmarshal(b, &prodMeta); err != nil {
		return nil, err
	}
	return &prodMeta, nil
}

// [CANONICAL] All metadata must be normalized and calculated via metadata.NormalizeAndCalculate before persistence or emission.
// Ensure required fields (versioning, audit, etc.) are present under the correct namespace.
