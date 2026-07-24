package service

import (
	"context"
	"time"

	"MetalTracker/internal/domain"
	"MetalTracker/internal/price"
)

type valuationContext struct {
	settings domain.AppSettings
	quote    price.Quote
}

func (services *AppServices) loadValuationContext(ctx context.Context) (valuationContext, error) {
	settings, err := services.vaultDB.GetSettings()
	if err != nil {
		return valuationContext{}, err
	}
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
	return valuationContext{settings: settings, quote: quote}, nil
}

func valueUnitInDisplayCurrency(unit domain.HoldingUnit, quote price.Quote, displayCurrency domain.Currency) domain.UnitValuation {
	purchasePrice, fxRate, fxOK := price.ConvertAmount(unit.CostBasis(), unit.Currency, displayCurrency, quote)
	spotAtPurchase, _, _ := price.ConvertAmount(unit.SpotWorthAtPurchase, unit.Currency, displayCurrency, quote)

	converted := unit
	converted.PurchasePrice = purchasePrice
	converted.SpotWorthAtPurchase = spotAtPurchase
	if unit.SalePrice != nil {
		salePrice, _, saleOK := price.ConvertAmount(*unit.SalePrice, unit.Currency, displayCurrency, quote)
		converted.SalePrice = &salePrice
		if !saleOK {
			fxOK = false
		}
	}

	currentWorth := spotAtPurchase
	approximate := !fxOK
	if unit.Status != domain.UnitStatusSold {
		if spot, ok := price.MetalSpotPerKg(quote, unit.Metal); ok {
			currentWorth = price.SpotWorthForUnit(spot, unit.WeightGrams, unit.Purity)
		} else {
			approximate = true
		}
	}

	valuation := domain.ValueUnit(converted, currentWorth)
	valuation.FineWeightGrams = domain.FineWeightGrams(unit.WeightGrams, unit.Purity)
	valuation.ValuationApproximate = approximate || quote.IsStale || quote.IsPartial || quote.ErrorCode != ""
	valuation.DisplayCurrency = displayCurrency
	if fxOK && unit.Currency != displayCurrency {
		valuation.FxRateToDisplay = fxRate
	}
	// Keep original currency on the unit for reference; monetary fields are display-converted.
	valuation.Currency = unit.Currency
	valuation.IsGift = unit.IsGift

	endDate := time.Now().UTC().Format("2006-01-02")
	if unit.Status == domain.UnitStatusSold && unit.SoldAt != "" {
		endDate = unit.SoldAt
	} else if unit.Status != domain.UnitStatusSold && !unit.IsGift {
		valuation.BreakEvenSpotPerKg = domain.BreakEvenSpotPerKg(
			valuation.PurchasePrice,
			valuation.FineWeightGrams,
		)
	}
	valuation.DaysHeld = domain.DaysHeldBetween(unit.PurchasedAt, endDate)
	valuation.AnnualizedReturnPct = domain.AnnualizedReturnPct(valuation.TotalProfitPct, valuation.DaysHeld)
	return valuation
}

func applyQuoteMetaToSummary(summary *domain.PortfolioSummary, quote price.Quote) {
	asOf := quote.FetchedAt
	if asOf.IsZero() {
		asOf = quote.Timestamp
	}
	if !asOf.IsZero() {
		summary.QuoteAsOf = asOf.UTC().Format(time.RFC3339)
	}
	summary.QuoteIsStale = quote.IsStale
	summary.QuoteIsPartial = quote.IsPartial
	summary.QuoteCacheHit = quote.CacheHit
	summary.PriceErrorCode = quote.ErrorCode
	summary.ValuationApproximate = quote.IsStale || quote.IsPartial || quote.ErrorCode != "" ||
		(summary.GoldSpotPerKg == 0 && summary.SilverSpotPerKg == 0 && summary.HeldUnits > 0)
}
