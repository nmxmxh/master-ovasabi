package babel

import (
	"context"
	"database/sql"
	"time"
)

type PricingRule struct {
	ID            int64
	CountryCode   string
	Region        string
	City          string
	CurrencyCode  string
	AffluenceTier string
	DemandLevel   string
	Multiplier    float64
	BasePrice     float64
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	KGEntityID    string
	Notes         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Repository struct {
	DB *sql.DB
}

// FindBestRule returns the most specific pricing rule for the given location and time.
func (r *Repository) FindBestRule(ctx context.Context, country, region, city string, now time.Time) (*PricingRule, error) {
	query := `
		SELECT id, country_code, region, city, currency_code, affluence_tier, demand_level, multiplier, base_price, effective_from, effective_to, kg_entity_id, notes, created_at, updated_at
		FROM babel_pricing_rules
		WHERE (country_code = $1 OR country_code IS NULL)
		  AND (region = $2 OR region IS NULL)
		  AND (city = $3 OR city IS NULL)
		  AND effective_from <= $4
		  AND (effective_to IS NULL OR effective_to >= $4)
		ORDER BY (city IS NOT NULL) DESC, (region IS NOT NULL) DESC, (country_code IS NOT NULL) DESC, effective_from DESC
		LIMIT 1
	`
	row := r.DB.QueryRowContext(ctx, query, country, region, city, now)
	var rule PricingRule
	var effectiveTo sql.NullTime
	if err := row.Scan(&rule.ID, &rule.CountryCode, &rule.Region, &rule.City, &rule.CurrencyCode, &rule.AffluenceTier, &rule.DemandLevel, &rule.Multiplier, &rule.BasePrice, &rule.EffectiveFrom, &effectiveTo, &rule.KGEntityID, &rule.Notes, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
		return nil, err
	}
	if effectiveTo.Valid {
		rule.EffectiveTo = &effectiveTo.Time
	}
	return &rule, nil
}

// Initial sample data for seeding or testing
var InitialPricingRules = []PricingRule{
	{
		CountryCode:   "US",
		Region:        "CA",
		City:          "San Francisco",
		CurrencyCode:  "USD",
		AffluenceTier: "Very High",
		DemandLevel:   "Peak",
		Multiplier:    1.8,
		BasePrice:     120.0,
		EffectiveFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		KGEntityID:    "KG:LOC:SF",
		Notes:         "Tech hub, high demand",
	},
	{
		CountryCode:   "NG",
		Region:        "Lagos",
		City:          "Lagos",
		CurrencyCode:  "NGN",
		AffluenceTier: "Medium",
		DemandLevel:   "Normal",
		Multiplier:    1.0,
		BasePrice:     50.0,
		EffectiveFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		KGEntityID:    "KG:LOC:LAG",
		Notes:         "Commercial center",
	},
	{
		CountryCode:   "CH",
		Region:        "ZH",
		City:          "Zurich",
		CurrencyCode:  "CHF",
		AffluenceTier: "Very High",
		DemandLevel:   "High",
		Multiplier:    1.7,
		BasePrice:     200.0,
		EffectiveFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		KGEntityID:    "KG:LOC:ZUR",
		Notes:         "Banking capital",
	},
	{
		CountryCode:   "IN",
		Region:        "MH",
		City:          "Mumbai",
		CurrencyCode:  "INR",
		AffluenceTier: "Low",
		DemandLevel:   "Normal",
		Multiplier:    0.8,
		BasePrice:     30.0,
		EffectiveFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		KGEntityID:    "KG:LOC:MUM",
		Notes:         "Emerging market",
	},
	{
		CountryCode:   "US",
		Region:        "",
		City:          "",
		CurrencyCode:  "USD",
		AffluenceTier: "High",
		DemandLevel:   "Normal",
		Multiplier:    1.3,
		BasePrice:     100.0,
		EffectiveFrom: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		KGEntityID:    "KG:LOC:US",
		Notes:         "Default US pricing",
	},
}
