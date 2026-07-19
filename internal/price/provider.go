package price

import (
	"context"
	"time"

	"MetalTracker/internal/domain"
)

type Quote struct {
	Base      string
	Date      string
	Timestamp time.Time
	FetchedAt time.Time
	// Rates maps symbol -> amount of currency per 1 kg of metal, or FX units of symbol per 1 base.
	Rates      map[string]float64
	CacheHit   bool
	IsStale    bool
	IsPartial  bool
	ErrorCode  string
}

type Provider interface {
	Latest(ctx context.Context, base string, symbols []string) (Quote, error)
	Historical(ctx context.Context, date time.Time, base string, symbols []string) (Quote, error)
	Timeframe(ctx context.Context, from, to time.Time, base string, symbols []string) ([]Quote, error)
}

func MetalSpotPerKg(quote Quote, metal domain.MetalSymbol) (float64, bool) {
	if priceValue, ok := quote.Rates[string(metal)]; ok && priceValue > 0 {
		return priceValue, true
	}
	return 0, false
}

func SpotWorthForUnit(pricePerKg float64, weightGrams float64, purity float64) float64 {
	fineGrams := domain.FineWeightGrams(weightGrams, purity)
	return (fineGrams / domain.GramsPerKilogram) * pricePerKg
}

func QuoteHasMetal(quote Quote, metal domain.MetalSymbol) bool {
	_, ok := MetalSpotPerKg(quote, metal)
	return ok
}

func ToSpotQuote(quote Quote) domain.SpotQuote {
	fetchedAt := quote.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = quote.Timestamp
	}
	timestamp := quote.Timestamp
	if timestamp.IsZero() {
		timestamp = fetchedAt
	}
	return domain.SpotQuote{
		Base:      quote.Base,
		Timestamp: timestamp.UTC().Format(time.RFC3339),
		FetchedAt: fetchedAt.UTC().Format(time.RFC3339),
		Rates:     quote.Rates,
		CacheHit:  quote.CacheHit,
		IsStale:   quote.IsStale,
		IsPartial: quote.IsPartial,
		ErrorCode: quote.ErrorCode,
	}
}
