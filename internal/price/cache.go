package price

import (
	"context"
	"strings"
	"sync"
	"time"

	"MetalTracker/internal/apperr"
	"MetalTracker/internal/storage"
)

const (
	liveQuoteDateKey = "latest"
	liveQuoteTTL     = 6 * time.Hour
	quoteUnitKilogram = "kilogram"
)

type CachedProvider struct {
	inner   Provider
	cache   *storage.DB
	mu      sync.Mutex
	liveTTL time.Duration
}

func NewCachedProvider(inner Provider, cacheDB *storage.DB) *CachedProvider {
	return &CachedProvider{
		inner:   inner,
		cache:   cacheDB,
		liveTTL: liveQuoteTTL,
	}
}

func (provider *CachedProvider) SetInner(inner Provider) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	provider.inner = inner
}

func (provider *CachedProvider) Latest(ctx context.Context, base string, symbols []string) (Quote, error) {
	provider.mu.Lock()
	inner := provider.inner
	provider.mu.Unlock()

	today := time.Now().UTC().Format("2006-01-02")
	rates := make(map[string]float64, len(symbols))
	missing := make([]string, 0)
	var oldestFetched time.Time
	cacheHits := 0

	for _, symbol := range symbols {
		cached, found, err := provider.cache.GetCachedQuote(liveQuoteDateKey, base, symbol, quoteUnitKilogram)
		if err != nil {
			return Quote{}, err
		}
		if found && time.Since(cached.FetchedAt) < provider.liveTTL {
			rates[symbol] = cached.PricePerUnit
			cacheHits++
			if oldestFetched.IsZero() || cached.FetchedAt.Before(oldestFetched) {
				oldestFetched = cached.FetchedAt
			}
			continue
		}
		dayCached, dayFound, dayErr := provider.cache.GetCachedQuote(today, base, symbol, quoteUnitKilogram)
		if dayErr != nil {
			return Quote{}, dayErr
		}
		if dayFound && time.Since(dayCached.FetchedAt) < provider.liveTTL {
			rates[symbol] = dayCached.PricePerUnit
			cacheHits++
			if oldestFetched.IsZero() || dayCached.FetchedAt.Before(oldestFetched) {
				oldestFetched = dayCached.FetchedAt
			}
			continue
		}
		// Keep stale cache as fallback while we try to refresh.
		if found {
			rates[symbol] = cached.PricePerUnit
			if oldestFetched.IsZero() || cached.FetchedAt.Before(oldestFetched) {
				oldestFetched = cached.FetchedAt
			}
		} else if dayFound {
			rates[symbol] = dayCached.PricePerUnit
			if oldestFetched.IsZero() || dayCached.FetchedAt.Before(oldestFetched) {
				oldestFetched = dayCached.FetchedAt
			}
		}
		missing = append(missing, symbol)
	}

	if len(missing) == 0 {
		fetchedAt := oldestFetched
		if fetchedAt.IsZero() {
			fetchedAt = time.Now().UTC()
		}
		return Quote{
			Base:      base,
			Date:      today,
			Timestamp: fetchedAt,
			FetchedAt: fetchedAt,
			Rates:     rates,
			CacheHit:  true,
			IsStale:   time.Since(fetchedAt) >= provider.liveTTL,
			IsPartial: false,
		}, nil
	}

	fetched, err := inner.Latest(ctx, base, missing)
	if err != nil {
		if len(rates) > 0 {
			fetchedAt := oldestFetched
			if fetchedAt.IsZero() {
				fetchedAt = time.Now().UTC()
			}
			errorCode := apperr.CodePriceUnavailable
			if isAPIKeyError(err) {
				errorCode = apperr.CodeInvalidAPIKey
			}
			return Quote{
				Base:      base,
				Date:      today,
				Timestamp: fetchedAt,
				FetchedAt: fetchedAt,
				Rates:     rates,
				CacheHit:  cacheHits > 0,
				IsStale:   true,
				IsPartial: true,
				ErrorCode: errorCode,
			}, nil
		}
		if isAPIKeyError(err) {
			return Quote{}, apperr.New(apperr.CodeInvalidAPIKey, err.Error())
		}
		return Quote{}, apperr.PriceUnavailable(err.Error())
	}

	now := time.Now().UTC()
	for symbol, priceValue := range fetched.Rates {
		rates[symbol] = priceValue
		_ = provider.cache.UpsertCachedQuote(storage.CachedQuote{
			QuoteDate:    liveQuoteDateKey,
			BaseCurrency: base,
			Symbol:       symbol,
			Unit:         quoteUnitKilogram,
			PricePerUnit: priceValue,
			FetchedAt:    now,
		})
		_ = provider.cache.UpsertCachedQuote(storage.CachedQuote{
			QuoteDate:    today,
			BaseCurrency: base,
			Symbol:       symbol,
			Unit:         quoteUnitKilogram,
			PricePerUnit: priceValue,
			FetchedAt:    now,
		})
	}

	stillMissing := false
	for _, symbol := range symbols {
		if _, ok := rates[symbol]; !ok {
			stillMissing = true
			break
		}
	}

	return Quote{
		Base:      base,
		Date:      today,
		Timestamp: now,
		FetchedAt: now,
		Rates:     rates,
		CacheHit:  cacheHits > 0 && len(missing) < len(symbols),
		IsStale:   false,
		IsPartial: stillMissing,
	}, nil
}

func (provider *CachedProvider) Historical(ctx context.Context, date time.Time, base string, symbols []string) (Quote, error) {
	provider.mu.Lock()
	inner := provider.inner
	provider.mu.Unlock()

	dateKey := date.Format("2006-01-02")
	rates := make(map[string]float64, len(symbols))
	missing := make([]string, 0)
	cacheHits := 0
	var fetchedAt time.Time

	for _, symbol := range symbols {
		cached, found, err := provider.cache.GetCachedQuote(dateKey, base, symbol, quoteUnitKilogram)
		if err != nil {
			return Quote{}, err
		}
		if found {
			rates[symbol] = cached.PricePerUnit
			cacheHits++
			if fetchedAt.IsZero() || cached.FetchedAt.Before(fetchedAt) {
				fetchedAt = cached.FetchedAt
			}
			continue
		}
		missing = append(missing, symbol)
	}

	if len(missing) == 0 {
		if fetchedAt.IsZero() {
			fetchedAt = date
		}
		return Quote{
			Base:      base,
			Date:      dateKey,
			Timestamp: date,
			FetchedAt: fetchedAt,
			Rates:     rates,
			CacheHit:  true,
		}, nil
	}

	fetched, err := inner.Historical(ctx, date, base, missing)
	if err != nil {
		if len(rates) > 0 {
			if fetchedAt.IsZero() {
				fetchedAt = date
			}
			return Quote{
				Base:      base,
				Date:      dateKey,
				Timestamp: date,
				FetchedAt: fetchedAt,
				Rates:     rates,
				CacheHit:  cacheHits > 0,
				IsPartial: true,
				IsStale:   true,
				ErrorCode: apperr.CodePriceUnavailable,
			}, nil
		}
		if isAPIKeyError(err) {
			return Quote{}, apperr.New(apperr.CodeInvalidAPIKey, err.Error())
		}
		return Quote{}, apperr.PriceUnavailable(err.Error())
	}

	now := time.Now().UTC()
	for symbol, priceValue := range fetched.Rates {
		rates[symbol] = priceValue
		_ = provider.cache.UpsertCachedQuote(storage.CachedQuote{
			QuoteDate:    dateKey,
			BaseCurrency: base,
			Symbol:       symbol,
			Unit:         quoteUnitKilogram,
			PricePerUnit: priceValue,
			FetchedAt:    now,
		})
	}

	stillMissing := false
	for _, symbol := range symbols {
		if _, ok := rates[symbol]; !ok {
			stillMissing = true
			break
		}
	}

	return Quote{
		Base:      fetched.Base,
		Date:      dateKey,
		Timestamp: fetched.Timestamp,
		FetchedAt: now,
		Rates:     rates,
		CacheHit:  cacheHits > 0,
		IsPartial: stillMissing,
	}, nil
}

func (provider *CachedProvider) Timeframe(ctx context.Context, from, to time.Time, base string, symbols []string) ([]Quote, error) {
	provider.mu.Lock()
	inner := provider.inner
	provider.mu.Unlock()

	// Prefer cache-assembled timeframe when every day is fully cached.
	if cachedQuotes, ok := provider.timeframeFromCache(from, to, base, symbols); ok {
		return cachedQuotes, nil
	}

	quotes, err := inner.Timeframe(ctx, from, to, base, symbols)
	if err != nil {
		if isAPIKeyError(err) {
			return nil, apperr.New(apperr.CodeInvalidAPIKey, err.Error())
		}
		return nil, apperr.PriceUnavailable(err.Error())
	}
	now := time.Now().UTC()
	for index := range quotes {
		quote := &quotes[index]
		quote.FetchedAt = now
		for symbol, priceValue := range quote.Rates {
			_ = provider.cache.UpsertCachedQuote(storage.CachedQuote{
				QuoteDate:    quote.Date,
				BaseCurrency: base,
				Symbol:       symbol,
				Unit:         quoteUnitKilogram,
				PricePerUnit: priceValue,
				FetchedAt:    now,
			})
		}
	}
	return quotes, nil
}

func (provider *CachedProvider) timeframeFromCache(from, to time.Time, base string, symbols []string) ([]Quote, bool) {
	if to.Before(from) {
		return nil, false
	}
	quotes := make([]Quote, 0)
	for cursor := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC); !cursor.After(to); cursor = cursor.AddDate(0, 0, 1) {
		dateKey := cursor.Format("2006-01-02")
		rates := make(map[string]float64, len(symbols))
		var fetchedAt time.Time
		for _, symbol := range symbols {
			cached, found, err := provider.cache.GetCachedQuote(dateKey, base, symbol, quoteUnitKilogram)
			if err != nil || !found {
				return nil, false
			}
			rates[symbol] = cached.PricePerUnit
			if fetchedAt.IsZero() || cached.FetchedAt.Before(fetchedAt) {
				fetchedAt = cached.FetchedAt
			}
		}
		quotes = append(quotes, Quote{
			Base:      base,
			Date:      dateKey,
			Timestamp: cursor,
			FetchedAt: fetchedAt,
			Rates:     rates,
			CacheHit:  true,
		})
	}
	return quotes, true
}

func isAPIKeyError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "api key") ||
		strings.Contains(text, "invalid_api_key") ||
		strings.Contains(text, "not configured")
}
