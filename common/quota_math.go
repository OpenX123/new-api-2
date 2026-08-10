package common

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

const (
	MaxQuota = math.MaxInt32
	MinQuota = math.MinInt32
)

// QuotaFromFloat converts a computed quota value to int with saturation.
// Quota products can include user-controlled multipliers (image n, video
// seconds, resolution ratios); an oversized product must never wrap around
// and turn a charge into a credit. The bound is int32 because quota columns
// (user/token/log) are 32-bit integers in the database.
func QuotaFromFloat(value float64) int {
	if math.IsNaN(value) {
		return 0
	}
	if value >= math.MaxInt32 {
		return math.MaxInt32
	}
	if value <= math.MinInt32 {
		return math.MinInt32
	}
	return int(value)
}

// QuotaFromDecimalStrict converts an in-range decimal quota and rejects a
// value that would otherwise be saturated at the database's int32 boundary.
func QuotaFromDecimalStrict(d decimal.Decimal) (int, error) {
	value, _ := d.Round(0).Float64()
	if value >= math.MaxInt32 || value <= math.MinInt32 {
		return 0, fmt.Errorf("quota conversion overflow: %s", d.String())
	}
	return QuotaFromFloat(value), nil
}

func QuotaFromDecimal(d decimal.Decimal) int {
	value, _ := d.Round(0).Float64()
	return QuotaFromFloat(value)
}
