package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	GramsPerTroyOunce = 31.1034768
	GramsPerKilogram  = 1000.0
)

func ToGrams(weight float64, unit WeightUnit) (float64, error) {
	switch unit {
	case WeightUnitGram, "":
		return weight, nil
	case WeightUnitTroyOz:
		return weight * GramsPerTroyOunce, nil
	case WeightUnitKilogram:
		return weight * GramsPerKilogram, nil
	default:
		return 0, fmt.Errorf("unsupported weight unit: %s", unit)
	}
}

func GramsToTroyOunces(grams float64) float64 {
	return grams / GramsPerTroyOunce
}

// KilogramToTroyOzPrice converts a per-kilogram spot into a per-troy-ounce spot.
func KilogramToTroyOzPrice(pricePerKg float64) float64 {
	return pricePerKg * (GramsPerTroyOunce / GramsPerKilogram)
}

// TroyOzToKilogramPrice converts a per-troy-ounce spot into a per-kilogram spot.
func TroyOzToKilogramPrice(pricePerTroyOz float64) float64 {
	return pricePerTroyOz * (GramsPerKilogram / GramsPerTroyOunce)
}

func FineWeightGrams(weightGrams float64, purity float64) float64 {
	if purity <= 0 {
		return weightGrams
	}
	return weightGrams * NormalizePurity(purity)
}

func NormalizePurity(purity float64) float64 {
	if purity <= 0 {
		return 1
	}
	if purity > 1 && purity <= 100 {
		return purity / 100.0
	}
	if purity > 100 {
		return purity / 1000.0
	}
	return purity
}

// IsUnusualPurity reports soft-warning purities outside common bullion/coin bands.
func IsUnusualPurity(purity float64) bool {
	normalized := NormalizePurity(purity)
	if normalized <= 0 || normalized > 1 {
		return true
	}
	if normalized >= 0.999 {
		return false
	}
	commonValues := []float64{0.995, 0.99, 0.925, 0.9, 0.835, 0.8}
	for _, value := range commonValues {
		if math.Abs(normalized-value) < 0.001 {
			return false
		}
	}
	return true
}

func GramsToSpotUnit(grams float64, unit SpotPriceUnit) float64 {
	switch unit {
	case SpotPriceUnitKilogram:
		return grams / GramsPerKilogram
	case SpotPriceUnitTroyOz:
		return GramsToTroyOunces(grams)
	case SpotPriceUnitGram, "":
		return grams
	default:
		return grams
	}
}

// ConvertSpotPricePerKg converts a per-kilogram price into the display spot unit.
func ConvertSpotPricePerKg(pricePerKg float64, unit SpotPriceUnit) float64 {
	switch unit {
	case SpotPriceUnitKilogram:
		return pricePerKg
	case SpotPriceUnitGram:
		return pricePerKg / GramsPerKilogram
	case SpotPriceUnitTroyOz:
		return KilogramToTroyOzPrice(pricePerKg)
	default:
		return pricePerKg
	}
}

func trimFloat(value float64) string {
	text := strconv.FormatFloat(value, 'f', 8, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" || text == "-" {
		return "0"
	}
	return text
}

func MetalDisplayName(metal MetalSymbol) string {
	switch metal {
	case MetalGold:
		return "Gold"
	case MetalSilver:
		return "Silver"
	default:
		return string(metal)
	}
}

func FormDisplayName(form Form) string {
	switch form {
	case FormCoin:
		return "Coin"
	case FormBar:
		return "Bar"
	case FormOther:
		return "Other"
	default:
		return string(form)
	}
}
