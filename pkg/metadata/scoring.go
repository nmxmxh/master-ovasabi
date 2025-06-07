package metadata

// OneValueScore returns the score as the count of items (each worth 1 point).
func OneValueScore(count int) float64 {
	return float64(count)
}

type TAS struct {
	Trust    float64 // 0-1
	Activity float64 // 0-1
	Strength float64 // 0-1
}

// TASScore returns the weighted sum of trust, activity, and strength for all relationships.
func TASScore(tasList []TAS, trustWeight, activityWeight, strengthWeight float64) float64 {
	var total float64
	for _, t := range tasList {
		total += t.Trust*trustWeight + t.Activity*activityWeight + t.Strength*strengthWeight
	}
	return total
}

// CalculateTotalTax returns the sum of all connector percentages (never negative).
func CalculateTotalTax(connectors []map[string]interface{}) float64 {
	var total float64
	for _, c := range connectors {
		if pct, ok := c["percentage"].(float64); ok && pct > 0 {
			total += pct
		}
	}
	return total
}
