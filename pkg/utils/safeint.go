package utils

import (
	"math"
	"math/big"
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

// ToBigInt safely converts an int to a *big.Int.
func ToBigInt(i int) *big.Int {
	return big.NewInt(int64(i))
}

// ToBigInt64 safely converts an int64 to a *big.Int.
func ToBigInt64(i int64) *big.Int {
	return big.NewInt(i)
}
