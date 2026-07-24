package service

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"MetalTracker/internal/apperr"
	"MetalTracker/internal/domain"
	"MetalTracker/internal/price"
)

func (services *AppServices) GetAllocationBreakdown(ctx context.Context) (domain.AllocationBreakdown, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return domain.AllocationBreakdown{}, mapError(err)
	}

	valuation, err := services.loadValuationContext(ctx)
	if err != nil {
		return domain.AllocationBreakdown{}, mapError(err)
	}
	units, err := services.vaultDB.ListUnits()
	if err != nil {
		return domain.AllocationBreakdown{}, mapError(err)
	}

	metalWorth := map[string]float64{}
	formWorth := map[string]float64{}
	brandWorth := map[string]float64{}
	locationWorth := map[string]float64{}
	var totalWorth float64

	for _, unit := range units {
		if unit.Status == domain.UnitStatusSold {
			continue
		}
		valued := valueUnitInDisplayCurrency(unit, valuation.quote, valuation.settings.DisplayCurrency)
		totalWorth += valued.CurrentSpotWorth
		metalWorth[string(unit.Metal)] += valued.CurrentSpotWorth
		formWorth[string(unit.Form)] += valued.CurrentSpotWorth
		brand := strings.TrimSpace(unit.Brand)
		if brand == "" {
			brand = "Unknown"
		}
		brandWorth[brand] += valued.CurrentSpotWorth
		location := strings.TrimSpace(unit.StorageLocation)
		if location == "" {
			location = "Unset"
		}
		locationWorth[location] += valued.CurrentSpotWorth
	}

	return domain.AllocationBreakdown{
		DisplayCurrency: valuation.settings.DisplayCurrency,
		TotalWorth:      totalWorth,
		ByMetal:         allocationSlices(metalWorth, totalWorth, metalAllocationLabel),
		ByForm:          allocationSlices(formWorth, totalWorth, formAllocationLabel),
		ByBrand:         allocationSlices(brandWorth, totalWorth, identityLabel),
		ByLocation:      allocationSlices(locationWorth, totalWorth, identityLabel),
	}, nil
}

func (services *AppServices) GetMetalAverageCosts(ctx context.Context) ([]domain.MetalAverageCost, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return nil, mapError(err)
	}

	valuation, err := services.loadValuationContext(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	units, err := services.vaultDB.ListUnits()
	if err != nil {
		return nil, mapError(err)
	}

	type accumulator struct {
		cost float64
		fine float64
		held int
	}
	byMetal := map[domain.MetalSymbol]*accumulator{}
	for _, unit := range units {
		if unit.Status == domain.UnitStatusSold {
			continue
		}
		valued := valueUnitInDisplayCurrency(unit, valuation.quote, valuation.settings.DisplayCurrency)
		entry, exists := byMetal[unit.Metal]
		if !exists {
			entry = &accumulator{}
			byMetal[unit.Metal] = entry
		}
		entry.held++
		if unit.IsGift {
			continue
		}
		entry.cost += valued.PurchasePrice
		entry.fine += valued.FineWeightGrams
	}

	result := make([]domain.MetalAverageCost, 0, len(byMetal))
	for metal, entry := range byMetal {
		result = append(result, domain.MetalAverageCost{
			Metal:                metal,
			TotalPurchaseCost:    entry.cost,
			TotalFineWeightGrams: entry.fine,
			AvgCostPerKgFine:     domain.AvgCostPerKgFine(entry.cost, entry.fine),
			HeldUnits:            entry.held,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Metal < result[j].Metal
	})
	return result, nil
}

func (services *AppServices) PreviewWhatIf(ctx context.Context, request domain.WhatIfRequest) (domain.WhatIfPreview, error) {
	services.mu.Lock()
	defer services.mu.Unlock()
	if err := services.requireUnlockedLocked(); err != nil {
		return domain.WhatIfPreview{}, mapError(err)
	}
	if request.GoldSpot < 0 || request.SilverSpot < 0 {
		return domain.WhatIfPreview{}, apperr.Validation("spot prices cannot be negative")
	}
	spotUnit := request.SpotUnit
	if spotUnit == "" {
		spotUnit = domain.SpotPriceUnitKilogram
	}

	valuation, err := services.loadValuationContext(ctx)
	if err != nil {
		return domain.WhatIfPreview{}, mapError(err)
	}
	units, err := services.vaultDB.ListUnits()
	if err != nil {
		return domain.WhatIfPreview{}, mapError(err)
	}

	goldPerKg := domain.SpotPriceToPerKg(request.GoldSpot, spotUnit)
	silverPerKg := domain.SpotPriceToPerKg(request.SilverSpot, spotUnit)

	var portfolioWorth float64
	var baselineWorth float64
	var purchaseCost float64
	for _, unit := range units {
		if unit.Status == domain.UnitStatusSold {
			continue
		}
		valued := valueUnitInDisplayCurrency(unit, valuation.quote, valuation.settings.DisplayCurrency)
		baselineWorth += valued.CurrentSpotWorth
		purchaseCost += valued.PurchasePrice
		hypotheticalSpot := goldPerKg
		if unit.Metal == domain.MetalSilver {
			hypotheticalSpot = silverPerKg
		}
		portfolioWorth += price.SpotWorthForUnit(hypotheticalSpot, unit.WeightGrams, unit.Purity)
	}

	unrealized := portfolioWorth - purchaseCost
	return domain.WhatIfPreview{
		DisplayCurrency:     valuation.settings.DisplayCurrency,
		SpotUnit:            spotUnit,
		PortfolioWorth:      portfolioWorth,
		BaselineWorth:       baselineWorth,
		TotalPurchaseCost:   purchaseCost,
		UnrealizedProfit:    unrealized,
		UnrealizedProfitPct: domain.Percentage(unrealized, purchaseCost),
		WorthDelta:          portfolioWorth - baselineWorth,
	}, nil
}

type contributionBucket struct {
	unrealized float64
	realized   float64
	label      string
}

func (services *AppServices) GetPnLContribution(ctx context.Context, fromDate string, toDate string) (domain.PnLContributionReport, error) {
	services.mu.Lock()
	if err := services.requireUnlockedLocked(); err != nil {
		services.mu.Unlock()
		return domain.PnLContributionReport{}, mapError(err)
	}

	start, end, err := resolveDateRange(fromDate, toDate, 12)
	if err != nil {
		services.mu.Unlock()
		return domain.PnLContributionReport{}, apperr.Validation(err.Error())
	}

	settings, err := services.vaultDB.GetSettings()
	if err != nil {
		services.mu.Unlock()
		return domain.PnLContributionReport{}, mapError(err)
	}
	units, err := services.vaultDB.ListUnits()
	if err != nil {
		services.mu.Unlock()
		return domain.PnLContributionReport{}, mapError(err)
	}

	symbols := price.ValuationSymbols(settings.DisplayCurrency)
	services.mu.Unlock()

	quotesByDate := services.loadMonthBoundaryQuotes(ctx, start.AddDate(0, -1, 0), end, string(settings.DisplayCurrency), symbols)
	return buildPnLContribution(start, end, settings, units, quotesByDate), nil
}

func buildPnLContribution(
	start time.Time,
	end time.Time,
	settings domain.AppSettings,
	units []domain.HoldingUnit,
	quotesByDate map[string]price.Quote,
) domain.PnLContributionReport {
	startQuote := quoteOnOrBefore(quotesByDate, start)
	endQuote := quoteOnOrBefore(quotesByDate, end)

	byMetal := map[string]*contributionBucket{}
	byGroup := map[string]*contributionBucket{}

	ensure := func(store map[string]*contributionBucket, key string, label string) *contributionBucket {
		entry, exists := store[key]
		if !exists {
			entry = &contributionBucket{label: label}
			store[key] = entry
		}
		return entry
	}

	for _, unit := range units {
		purchasedAt, parseErr := domain.ParseDate(unit.PurchasedAt)
		if parseErr != nil || purchasedAt.IsZero() {
			continue
		}
		var soldAt time.Time
		if unit.SoldAt != "" {
			soldAt, _ = domain.ParseDate(unit.SoldAt)
		}

		metalKey := string(unit.Metal)
		groupKey := unit.ProductKey
		groupLabel := strings.TrimSpace(unit.ProductName)
		if groupLabel == "" {
			groupLabel = domain.MetalDisplayName(unit.Metal) + " " + domain.FormDisplayName(unit.Form)
		}
		metalBucket := ensure(byMetal, metalKey, domain.MetalDisplayName(unit.Metal))
		groupBucket := ensure(byGroup, groupKey, groupLabel)

		soldInRange := unit.Status == domain.UnitStatusSold && !soldAt.IsZero() &&
			!soldAt.Before(start) && !soldAt.After(end)
		if soldInRange && unit.SalePrice != nil {
			salePrice, _, _ := price.ConvertAmount(*unit.SalePrice, unit.Currency, settings.DisplayCurrency, endQuote)
			purchasePrice, _, _ := price.ConvertAmount(unit.CostBasis(), unit.Currency, settings.DisplayCurrency, endQuote)
			realized := salePrice - purchasePrice
			metalBucket.realized += realized
			groupBucket.realized += realized
			continue
		}

		heldAtStart := !purchasedAt.After(start) && (soldAt.IsZero() || soldAt.After(start))
		heldAtEnd := !purchasedAt.After(end) && (soldAt.IsZero() || soldAt.After(end))
		if !heldAtEnd {
			continue
		}

		endWorth := unitWorthAtQuote(unit, endQuote, settings.DisplayCurrency)
		var startWorth float64
		if heldAtStart {
			startWorth = unitWorthAtQuote(unit, startQuote, settings.DisplayCurrency)
		} else {
			startWorth, _, _ = price.ConvertAmount(unit.SpotWorthAtPurchase, unit.Currency, settings.DisplayCurrency, endQuote)
		}
		unrealized := endWorth - startWorth
		metalBucket.unrealized += unrealized
		groupBucket.unrealized += unrealized
	}

	report := domain.PnLContributionReport{
		DisplayCurrency: settings.DisplayCurrency,
		ByMetal:         contributionList(byMetal, "metal"),
		ByGroup:         contributionList(byGroup, "group"),
	}
	for _, item := range report.ByMetal {
		report.TotalUnrealized += item.UnrealizedChange
		report.TotalRealized += item.RealizedProfit
		report.TotalNet += item.NetChange
	}
	applyContributionPercents(report.ByMetal, report.TotalNet)
	applyContributionPercents(report.ByGroup, report.TotalNet)
	if len(report.ByGroup) > 10 {
		report.ByGroup = report.ByGroup[:10]
	}
	return report
}

func unitWorthAtQuote(unit domain.HoldingUnit, quote price.Quote, displayCurrency domain.Currency) float64 {
	if spot, ok := price.MetalSpotPerKg(quote, unit.Metal); ok {
		return price.SpotWorthForUnit(spot, unit.WeightGrams, unit.Purity)
	}
	worth, _, _ := price.ConvertAmount(unit.SpotWorthAtPurchase, unit.Currency, displayCurrency, quote)
	return worth
}

func allocationSlices(
	values map[string]float64,
	total float64,
	labelFor func(key string) string,
) []domain.AllocationSlice {
	slices := make([]domain.AllocationSlice, 0, len(values))
	for key, worth := range values {
		percent := 0.0
		if total > 0 {
			percent = (worth / total) * 100
		}
		slices = append(slices, domain.AllocationSlice{
			Key:     key,
			Label:   labelFor(key),
			Worth:   worth,
			Percent: percent,
		})
	}
	sort.Slice(slices, func(i, j int) bool {
		return slices[i].Worth > slices[j].Worth
	})
	return slices
}

func contributionList(values map[string]*contributionBucket, dimension string) []domain.PnLContribution {
	items := make([]domain.PnLContribution, 0, len(values))
	for key, entry := range values {
		net := entry.unrealized + entry.realized
		if math.Abs(net) < 0.0000001 && math.Abs(entry.unrealized) < 0.0000001 && math.Abs(entry.realized) < 0.0000001 {
			continue
		}
		items = append(items, domain.PnLContribution{
			Dimension:        dimension,
			Key:              key,
			Label:            entry.label,
			UnrealizedChange: entry.unrealized,
			RealizedProfit:   entry.realized,
			NetChange:        net,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return math.Abs(items[i].NetChange) > math.Abs(items[j].NetChange)
	})
	return items
}

func applyContributionPercents(items []domain.PnLContribution, totalNet float64) {
	for index := range items {
		if totalNet == 0 {
			items[index].PercentOfTotalNet = 0
			continue
		}
		items[index].PercentOfTotalNet = (items[index].NetChange / totalNet) * 100
	}
}

func metalAllocationLabel(key string) string {
	return domain.MetalDisplayName(domain.MetalSymbol(key))
}

func formAllocationLabel(key string) string {
	return domain.FormDisplayName(domain.Form(key))
}

func identityLabel(key string) string {
	return key
}
