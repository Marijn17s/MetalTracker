package service

import (
	"context"
	"sort"
	"sync"
	"time"

	"MetalTracker/internal/apperr"
	"MetalTracker/internal/domain"
	"MetalTracker/internal/price"
)

func (services *AppServices) GetPortfolioSummary(ctx context.Context) (domain.PortfolioSummary, error) {
	services.mu.Lock()
	if err := services.requireUnlockedLocked(); err != nil {
		services.mu.Unlock()
		return domain.PortfolioSummary{}, mapError(err)
	}

	settings, err := services.vaultDB.GetSettings()
	if err != nil {
		services.mu.Unlock()
		return domain.PortfolioSummary{}, mapError(err)
	}
	units, err := services.vaultDB.ListUnits()
	if err != nil {
		services.mu.Unlock()
		return domain.PortfolioSummary{}, mapError(err)
	}
	services.mu.Unlock()

	quote, err := services.priceProvider().Latest(
		ctx,
		string(settings.DisplayCurrency),
		price.ValuationSymbols(settings.DisplayCurrency),
	)
	if err != nil {
		quote = price.Quote{
			Base:      string(settings.DisplayCurrency),
			Rates:     map[string]float64{},
			IsStale:   true,
			IsPartial: true,
			ErrorCode: "price_unavailable",
		}
	}

	summary := domain.PortfolioSummary{
		DisplayCurrency: settings.DisplayCurrency,
	}
	if gold, ok := price.MetalSpotPerKg(quote, domain.MetalGold); ok {
		summary.GoldSpotPerKg = gold
	}
	if silver, ok := price.MetalSpotPerKg(quote, domain.MetalSilver); ok {
		summary.SilverSpotPerKg = silver
	}
	applyQuoteMetaToSummary(&summary, quote)

	for _, unit := range units {
		valued := valueUnitInDisplayCurrency(unit, quote, settings.DisplayCurrency)
		summary.TotalPurchaseCost += valued.PurchasePrice
		if valued.ValuationApproximate {
			summary.ValuationApproximate = true
		}
		if unit.Status == domain.UnitStatusSold && unit.SalePrice != nil {
			summary.SoldUnits++
			summary.TotalRealizedProfit += valued.TotalProfit
			summary.TotalCurrentWorth += valued.CurrentSpotWorth
		} else {
			summary.HeldUnits++
			summary.TotalCurrentWorth += valued.CurrentSpotWorth
			summary.TotalUnrealizedProfit += valued.TotalProfit
			switch unit.Metal {
			case domain.MetalGold:
				summary.HeldGoldFineWeightGrams += valued.FineWeightGrams
			case domain.MetalSilver:
				summary.HeldSilverFineWeightGrams += valued.FineWeightGrams
			}
		}
	}

	summary.TotalProfit = summary.TotalRealizedProfit + summary.TotalUnrealizedProfit
	summary.TotalProfitPct = domain.Percentage(summary.TotalProfit, summary.TotalPurchaseCost)
	return summary, nil
}

func (services *AppServices) GetPortfolioHistory(ctx context.Context, fromDate string, toDate string) ([]domain.PortfolioHistoryPoint, error) {
	services.mu.Lock()
	if err := services.requireUnlockedLocked(); err != nil {
		services.mu.Unlock()
		return nil, mapError(err)
	}

	start, end, err := resolveDateRange(fromDate, toDate, 12)
	if err != nil {
		services.mu.Unlock()
		return nil, apperr.Validation(err.Error())
	}

	settings, err := services.vaultDB.GetSettings()
	if err != nil {
		services.mu.Unlock()
		return nil, mapError(err)
	}
	units, err := services.vaultDB.ListUnits()
	if err != nil {
		services.mu.Unlock()
		return nil, mapError(err)
	}

	symbols := price.ValuationSymbols(settings.DisplayCurrency)
	services.mu.Unlock()

	// History chart only samples month ends - avoid full daily timeframe under the app lock.
	quotesByDate := services.loadMonthBoundaryQuotes(ctx, start, end, string(settings.DisplayCurrency), symbols)

	points := make([]domain.PortfolioHistoryPoint, 0)
	cursor := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cursor.After(end) {
		monthEnd := time.Date(cursor.Year(), cursor.Month()+1, 0, 23, 59, 59, 0, time.UTC)
		if monthEnd.After(end) {
			monthEnd = end
		}
		if monthEnd.Before(start) {
			cursor = cursor.AddDate(0, 1, 0)
			continue
		}

		quote := quoteOnOrBefore(quotesByDate, monthEnd)
		var worth float64
		var cost float64
		var goldWorth float64
		var goldCost float64
		var silverWorth float64
		var silverCost float64
		for _, unit := range units {
			purchasedAt, parseErr := domain.ParseDate(unit.PurchasedAt)
			if parseErr != nil || purchasedAt.After(monthEnd) {
				continue
			}
			if unit.Status == domain.UnitStatusSold && unit.SoldAt != "" {
				soldAt, soldErr := domain.ParseDate(unit.SoldAt)
				if soldErr == nil && !soldAt.After(monthEnd) {
					continue
				}
			}
			purchasePrice, _, _ := price.ConvertAmount(unit.CostBasis(), unit.Currency, settings.DisplayCurrency, quote)
			cost += purchasePrice
			var unitWorth float64
			if spot, ok := price.MetalSpotPerKg(quote, unit.Metal); ok {
				unitWorth = price.SpotWorthForUnit(spot, unit.WeightGrams, unit.Purity)
			} else {
				spotAtPurchase, _, _ := price.ConvertAmount(unit.SpotWorthAtPurchase, unit.Currency, settings.DisplayCurrency, quote)
				unitWorth = spotAtPurchase
			}
			worth += unitWorth
			switch unit.Metal {
			case domain.MetalGold:
				goldWorth += unitWorth
				goldCost += purchasePrice
			case domain.MetalSilver:
				silverWorth += unitWorth
				silverCost += purchasePrice
			}
		}

		points = append(points, domain.PortfolioHistoryPoint{
			Date:            monthEnd.Format("2006-01-02"),
			PortfolioWorth:  worth,
			CostBasis:       cost,
			GoldWorth:       goldWorth,
			GoldCostBasis:   goldCost,
			SilverWorth:     silverWorth,
			SilverCostBasis: silverCost,
		})
		cursor = cursor.AddDate(0, 1, 0)
	}

	return points, nil
}

func (services *AppServices) GetPortfolioValueAt(ctx context.Context, date string) (domain.PortfolioValueAt, error) {
	services.mu.Lock()
	if err := services.requireUnlockedLocked(); err != nil {
		services.mu.Unlock()
		return domain.PortfolioValueAt{}, mapError(err)
	}

	if date == "" {
		services.mu.Unlock()
		return domain.PortfolioValueAt{}, apperr.Validation("date is required")
	}
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		services.mu.Unlock()
		return domain.PortfolioValueAt{}, apperr.Validation("invalid date")
	}
	asOfDay := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if asOfDay.After(today) {
		services.mu.Unlock()
		return domain.PortfolioValueAt{}, apperr.Validation("date cannot be in the future")
	}
	asOf := time.Date(asOfDay.Year(), asOfDay.Month(), asOfDay.Day(), 23, 59, 59, 0, time.UTC)

	settings, err := services.vaultDB.GetSettings()
	if err != nil {
		services.mu.Unlock()
		return domain.PortfolioValueAt{}, mapError(err)
	}
	units, err := services.vaultDB.ListUnits()
	if err != nil {
		services.mu.Unlock()
		return domain.PortfolioValueAt{}, mapError(err)
	}

	symbols := price.ValuationSymbols(settings.DisplayCurrency)
	services.mu.Unlock()

	var quote price.Quote
	if asOfDay.Equal(today) {
		quote, err = services.priceProvider().Latest(ctx, string(settings.DisplayCurrency), symbols)
	} else {
		quote, err = services.priceProvider().Historical(ctx, asOfDay, string(settings.DisplayCurrency), symbols)
	}
	if err != nil {
		quote = price.Quote{
			Base:      string(settings.DisplayCurrency),
			Date:      asOfDay.Format("2006-01-02"),
			Rates:     map[string]float64{},
			IsStale:   true,
			IsPartial: true,
			ErrorCode: "price_unavailable",
		}
	}

	result := domain.PortfolioValueAt{
		Date:            asOfDay.Format("2006-01-02"),
		DisplayCurrency: settings.DisplayCurrency,
	}
	applyQuoteMetaToValueAt(&result, quote)

	for _, unit := range units {
		purchasedAt, parseErr := domain.ParseDate(unit.PurchasedAt)
		if parseErr != nil || purchasedAt.After(asOf) {
			continue
		}
		if unit.Status == domain.UnitStatusSold && unit.SoldAt != "" {
			soldAt, soldErr := domain.ParseDate(unit.SoldAt)
			if soldErr == nil && !soldAt.After(asOf) {
				continue
			}
		}

		purchasePrice, _, _ := price.ConvertAmount(unit.CostBasis(), unit.Currency, settings.DisplayCurrency, quote)
		result.CostBasis += purchasePrice
		result.HeldUnits++

		var unitWorth float64
		if spot, ok := price.MetalSpotPerKg(quote, unit.Metal); ok {
			unitWorth = price.SpotWorthForUnit(spot, unit.WeightGrams, unit.Purity)
		} else {
			spotAtPurchase, _, _ := price.ConvertAmount(unit.SpotWorthAtPurchase, unit.Currency, settings.DisplayCurrency, quote)
			unitWorth = spotAtPurchase
			result.ValuationApproximate = true
		}
		result.PortfolioWorth += unitWorth
		switch unit.Metal {
		case domain.MetalGold:
			result.GoldWorth += unitWorth
		case domain.MetalSilver:
			result.SilverWorth += unitWorth
		}
	}

	result.UnrealizedProfit = result.PortfolioWorth - result.CostBasis
	result.UnrealizedProfitPct = domain.Percentage(result.UnrealizedProfit, result.CostBasis)
	return result, nil
}

func applyQuoteMetaToValueAt(result *domain.PortfolioValueAt, quote price.Quote) {
	asOf := quote.FetchedAt
	if asOf.IsZero() {
		asOf = quote.Timestamp
	}
	if !asOf.IsZero() {
		result.QuoteAsOf = asOf.UTC().Format(time.RFC3339)
	} else if quote.Date != "" {
		result.QuoteAsOf = quote.Date
	}
	result.QuoteIsStale = quote.IsStale
	result.QuoteIsPartial = quote.IsPartial
	result.QuoteCacheHit = quote.CacheHit
	result.PriceErrorCode = quote.ErrorCode
	if quote.IsStale || quote.IsPartial || quote.ErrorCode != "" {
		result.ValuationApproximate = true
	}
}

func (services *AppServices) GetMonthlyBreakdown(ctx context.Context, fromDate string, toDate string) ([]domain.MonthlyMetalBreakdown, error) {
	services.mu.Lock()
	if err := services.requireUnlockedLocked(); err != nil {
		services.mu.Unlock()
		return nil, mapError(err)
	}

	start, end, err := resolveDateRange(fromDate, toDate, 12)
	if err != nil {
		services.mu.Unlock()
		return nil, apperr.Validation(err.Error())
	}
	start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)

	settings, err := services.vaultDB.GetSettings()
	if err != nil {
		services.mu.Unlock()
		return nil, mapError(err)
	}
	units, err := services.vaultDB.ListUnits()
	if err != nil {
		services.mu.Unlock()
		return nil, mapError(err)
	}

	metalsPresent := map[domain.MetalSymbol]bool{}
	for _, unit := range units {
		metalsPresent[unit.Metal] = true
	}
	metals := make([]domain.MetalSymbol, 0, 2)
	for _, metal := range []domain.MetalSymbol{domain.MetalGold, domain.MetalSilver} {
		if metalsPresent[metal] {
			metals = append(metals, metal)
		}
	}
	if len(metals) == 0 {
		services.mu.Unlock()
		return []domain.MonthlyMetalBreakdown{}, nil
	}

	symbols := price.ValuationSymbols(settings.DisplayCurrency)
	rangeStart := start.AddDate(0, 0, -1)
	services.mu.Unlock()

	quotesByDate := services.loadMonthBoundaryQuotes(ctx, rangeStart, end, string(settings.DisplayCurrency), symbols)
	return buildMonthlyBreakdown(start, end, settings, units, metals, quotesByDate), nil
}

func (services *AppServices) GetMonthlyPage(ctx context.Context, fromDate string, toDate string) (domain.MonthlyPage, error) {
	services.mu.Lock()
	if err := services.requireUnlockedLocked(); err != nil {
		services.mu.Unlock()
		return domain.MonthlyPage{}, mapError(err)
	}

	start, end, err := resolveDateRange(fromDate, toDate, 12)
	if err != nil {
		services.mu.Unlock()
		return domain.MonthlyPage{}, apperr.Validation(err.Error())
	}
	monthStart := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)

	settings, err := services.vaultDB.GetSettings()
	if err != nil {
		services.mu.Unlock()
		return domain.MonthlyPage{}, mapError(err)
	}
	units, err := services.vaultDB.ListUnits()
	if err != nil {
		services.mu.Unlock()
		return domain.MonthlyPage{}, mapError(err)
	}

	metalsPresent := map[domain.MetalSymbol]bool{}
	for _, unit := range units {
		metalsPresent[unit.Metal] = true
	}
	metals := make([]domain.MetalSymbol, 0, 2)
	for _, metal := range []domain.MetalSymbol{domain.MetalGold, domain.MetalSilver} {
		if metalsPresent[metal] {
			metals = append(metals, metal)
		}
	}

	symbols := price.ValuationSymbols(settings.DisplayCurrency)
	quoteFrom := monthStart.AddDate(0, 0, -1)
	if earlier := start.AddDate(0, -1, 0); earlier.Before(quoteFrom) {
		quoteFrom = earlier
	}
	services.mu.Unlock()

	quotesByDate := services.loadMonthBoundaryQuotes(ctx, quoteFrom, end, string(settings.DisplayCurrency), symbols)

	page := domain.MonthlyPage{
		Breakdown:    []domain.MonthlyMetalBreakdown{},
		Contribution: buildPnLContribution(start, end, settings, units, quotesByDate),
	}
	if len(metals) > 0 {
		page.Breakdown = buildMonthlyBreakdown(monthStart, end, settings, units, metals, quotesByDate)
	}
	return page, nil
}

func buildMonthlyBreakdown(
	start time.Time,
	end time.Time,
	settings domain.AppSettings,
	units []domain.HoldingUnit,
	metals []domain.MetalSymbol,
	quotesByDate map[string]price.Quote,
) []domain.MonthlyMetalBreakdown {
	type monthKey struct {
		yearMonth string
		metal     domain.MetalSymbol
	}
	results := map[monthKey]*domain.MonthlyMetalBreakdown{}

	for cursor := start; !cursor.After(end); cursor = cursor.AddDate(0, 1, 0) {
		monthStart := time.Date(cursor.Year(), cursor.Month(), 1, 0, 0, 0, 0, time.UTC)
		monthEnd := monthStart.AddDate(0, 1, -1)
		if monthEnd.After(end) {
			monthEnd = end
		}
		previousEnd := monthStart.Add(-time.Second)
		yearMonth := monthStart.Format("2006-01")

		startQuote := quoteOnOrBefore(quotesByDate, previousEnd)
		endQuote := quoteOnOrBefore(quotesByDate, monthEnd)

		for _, metal := range metals {
			key := monthKey{yearMonth: yearMonth, metal: metal}
			entry := &domain.MonthlyMetalBreakdown{
				YearMonth: yearMonth,
				Metal:     metal,
			}

			var soldCostBasis float64
			for _, unit := range units {
				if unit.Metal != metal {
					continue
				}
				purchasedAt, parseErr := domain.ParseDate(unit.PurchasedAt)
				if parseErr != nil {
					continue
				}
				var soldAt time.Time
				hasSale := false
				if unit.Status == domain.UnitStatusSold && unit.SoldAt != "" {
					if parsedSold, soldErr := domain.ParseDate(unit.SoldAt); soldErr == nil {
						soldAt = parsedSold
						hasSale = true
					}
				}

				boughtThisMonth := purchasedAt.Year() == monthStart.Year() && purchasedAt.Month() == monthStart.Month()
				soldThisMonth := hasSale && soldAt.Year() == monthStart.Year() && soldAt.Month() == monthStart.Month()
				ownedAtStart := !purchasedAt.After(previousEnd) && (!hasSale || soldAt.After(previousEnd))
				ownedAtEnd := !purchasedAt.After(monthEnd) && (!hasSale || soldAt.After(monthEnd))

				purchaseSpot, _, _ := price.ConvertAmount(unit.SpotWorthAtPurchase, unit.Currency, settings.DisplayCurrency, endQuote)
				unitStartWorth := purchaseSpot
				if spot, ok := price.MetalSpotPerKg(startQuote, metal); ok {
					unitStartWorth = price.SpotWorthForUnit(spot, unit.WeightGrams, unit.Purity)
				}
				unitEndWorth := purchaseSpot
				if spot, ok := price.MetalSpotPerKg(endQuote, metal); ok {
					unitEndWorth = price.SpotWorthForUnit(spot, unit.WeightGrams, unit.Purity)
				}

				if ownedAtStart {
					entry.OpeningWorth += unitStartWorth
				}

				if ownedAtStart && ownedAtEnd {
					entry.UnrealizedChange += unitEndWorth - unitStartWorth
				} else if boughtThisMonth && ownedAtEnd {
					entry.UnrealizedChange += unitEndWorth - purchaseSpot
				}

				if soldThisMonth && unit.SalePrice != nil {
					salePrice, _, _ := price.ConvertAmount(*unit.SalePrice, unit.Currency, settings.DisplayCurrency, endQuote)
					purchasePrice, _, _ := price.ConvertAmount(unit.CostBasis(), unit.Currency, settings.DisplayCurrency, endQuote)
					entry.RealizedProfit += salePrice - purchasePrice
					soldCostBasis += purchasePrice
				}
			}

			entry.NetChange = entry.UnrealizedChange + entry.RealizedProfit
			entry.UnrealizedPct = domain.Percentage(entry.UnrealizedChange, entry.OpeningWorth)
			entry.RealizedPct = domain.Percentage(entry.RealizedProfit, soldCostBasis)
			baseForNet := entry.OpeningWorth
			if baseForNet == 0 {
				baseForNet = soldCostBasis
			}
			entry.NetPct = domain.Percentage(entry.NetChange, baseForNet)
			results[key] = entry
		}
	}

	list := make([]domain.MonthlyMetalBreakdown, 0, len(results))
	for _, entry := range results {
		list = append(list, *entry)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].YearMonth == list[j].YearMonth {
			return list[i].Metal < list[j].Metal
		}
		return list[i].YearMonth > list[j].YearMonth
	})
	return list
}

func (services *AppServices) loadQuotesByDate(
	ctx context.Context,
	from time.Time,
	to time.Time,
	base string,
	symbols []string,
) map[string]price.Quote {
	quotesByDate := map[string]price.Quote{}

	timeframeQuotes, err := services.priceProvider().Timeframe(ctx, from, to, base, symbols)
	if err == nil {
		for _, quote := range timeframeQuotes {
			quotesByDate[quote.Date] = quote
		}
		return quotesByDate
	}

	// Fallback for daily history charts: month-boundary samples only.
	return services.loadMonthBoundaryQuotes(ctx, from, to, base, symbols)
}

// loadMonthBoundaryQuotes loads only month-end (and range edge) quotes.
// Avoids Timeframe fills of every calendar day, which hang on cold middleman/upstream.
func (services *AppServices) loadMonthBoundaryQuotes(
	ctx context.Context,
	from time.Time,
	to time.Time,
	base string,
	symbols []string,
) map[string]price.Quote {
	quotesByDate := map[string]price.Quote{}
	if to.Before(from) {
		return quotesByDate
	}

	sampleDates := monthBoundarySampleDates(from, to)
	if len(sampleDates) == 0 {
		return quotesByDate
	}

	provider := services.priceProvider()
	const maxParallel = 6
	jobs := make(chan time.Time)
	var waitGroup sync.WaitGroup
	var quotesMutex sync.Mutex

	workerCount := maxParallel
	if len(sampleDates) < workerCount {
		workerCount = len(sampleDates)
	}
	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for sampleDate := range jobs {
				if ctx.Err() != nil {
					return
				}
				quote, quoteErr := provider.Historical(ctx, sampleDate, base, symbols)
				if quoteErr != nil {
					continue
				}
				quotesMutex.Lock()
				quotesByDate[sampleDate.Format("2006-01-02")] = quote
				quotesMutex.Unlock()
			}
		}()
	}
	for _, sampleDate := range sampleDates {
		jobs <- sampleDate
	}
	close(jobs)
	waitGroup.Wait()
	return quotesByDate
}

func monthBoundarySampleDates(from time.Time, to time.Time) []time.Time {
	seen := map[string]bool{}
	dates := make([]time.Time, 0, 16)
	add := func(moment time.Time) {
		key := moment.UTC().Format("2006-01-02")
		if seen[key] {
			return
		}
		seen[key] = true
		dates = append(dates, moment.UTC())
	}

	add(from.UTC())
	cursor := time.Date(from.UTC().Year(), from.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cursor.After(to.UTC()) {
		monthEnd := time.Date(cursor.Year(), cursor.Month()+1, 0, 0, 0, 0, 0, time.UTC)
		if monthEnd.After(to.UTC()) {
			monthEnd = to.UTC()
		}
		add(monthEnd)
		cursor = cursor.AddDate(0, 1, 0)
	}
	add(to.UTC())
	return dates
}

func quoteOnOrBefore(quotesByDate map[string]price.Quote, moment time.Time) price.Quote {
	if len(quotesByDate) == 0 {
		return price.Quote{Rates: map[string]float64{}}
	}
	dateKey := moment.Format("2006-01-02")
	if quote, ok := quotesByDate[dateKey]; ok {
		return quote
	}
	var bestDate string
	var bestQuote price.Quote
	for key, quote := range quotesByDate {
		if key <= dateKey && (bestDate == "" || key > bestDate) {
			bestDate = key
			bestQuote = quote
		}
	}
	if bestDate != "" {
		return bestQuote
	}
	// Fall forward to earliest available.
	for key, quote := range quotesByDate {
		if bestDate == "" || key < bestDate {
			bestDate = key
			bestQuote = quote
		}
	}
	return bestQuote
}
