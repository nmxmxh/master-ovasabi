package utils

import (
	"math"
)

// ToInt32 safely converts an int to int32, clamping to the int32 range.
func ToInt32(i int) int32 {
	if i > math.MaxInt32 {
		return math.MaxInt32
	}
	if i < math.MinInt32 {
		return math.MinInt32
	}
	return int32(i)
}
