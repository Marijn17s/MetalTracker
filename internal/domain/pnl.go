package domain

import "math"

func ValueUnit(unit HoldingUnit, currentSpotWorth float64) UnitValuation {
	costBasis := unit.CostBasis()
	valuation := UnitValuation{
		HoldingUnit:      unit,
		CurrentSpotWorth: currentSpotWorth,
	}
	if !unit.IsGift {
		valuation.PremiumPaid = unit.PurchasePrice - unit.SpotWorthAtPurchase
	}

	if unit.Status == UnitStatusSold && unit.SalePrice != nil {
		valuation.IsRealized = true
		valuation.TotalProfit = *unit.SalePrice - costBasis
		valuation.MetalDelta = *unit.SalePrice - unit.SpotWorthAtPurchase
		valuation.CurrentSpotWorth = *unit.SalePrice
	} else {
		valuation.IsRealized = false
		valuation.TotalProfit = currentSpotWorth - costBasis
		valuation.MetalDelta = currentSpotWorth - unit.SpotWorthAtPurchase
	}

	if costBasis != 0 {
		valuation.TotalProfitPct = (valuation.TotalProfit / costBasis) * 100
	}
	return valuation
}

func Percentage(profit float64, cost float64) float64 {
	if cost == 0 {
		return 0
	}
	return (profit / cost) * 100
}

// BreakEvenSpotPerKg is the spot (per kg fine) where worth equals purchase cost.
func BreakEvenSpotPerKg(purchasePriceDisplay float64, fineWeightGrams float64) float64 {
	kilograms := fineWeightGrams / GramsPerKilogram
	if kilograms <= 0 {
		return 0
	}
	return purchasePriceDisplay / kilograms
}

// AvgCostPerKgFine is total purchase cost divided by fine kilograms.
func AvgCostPerKgFine(totalPurchaseCost float64, totalFineWeightGrams float64) float64 {
	kilograms := totalFineWeightGrams / GramsPerKilogram
	if kilograms <= 0 {
		return 0
	}
	return totalPurchaseCost / kilograms
}

// SpotPriceToPerKg converts a price in the user's spot unit into per-kilogram.
func SpotPriceToPerKg(price float64, unit SpotPriceUnit) float64 {
	switch unit {
	case SpotPriceUnitGram:
		return price * GramsPerKilogram
	case SpotPriceUnitTroyOz:
		return price * (GramsPerKilogram / GramsPerTroyOunce)
	default:
		return price
	}
}

// DaysHeldBetween returns whole days from purchase to end (sale or as-of). Minimum 1 when dates are valid.
func DaysHeldBetween(purchasedAt string, endAt string) int {
	start, startErr := ParseDate(purchasedAt)
	end, endErr := ParseDate(endAt)
	if startErr != nil || endErr != nil || start.IsZero() || end.IsZero() {
		return 0
	}
	if end.Before(start) {
		return 0
	}
	days := int(math.Floor(end.Sub(start).Hours() / 24))
	if days < 1 {
		return 1
	}
	return days
}

// AnnualizedReturnPct is a simple annualization of total return percent over days held.
func AnnualizedReturnPct(totalProfitPct float64, daysHeld int) float64 {
	if daysHeld <= 0 {
		return 0
	}
	return totalProfitPct * (365.0 / float64(daysHeld))
}
