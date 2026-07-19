package price

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"MetalTracker/internal/storage"
)

type stubProvider struct {
	latest Quote
	err    error
}

func (stub *stubProvider) Latest(ctx context.Context, base string, symbols []string) (Quote, error) {
	if stub.err != nil {
		return Quote{}, stub.err
	}
	return stub.latest, nil
}

func (stub *stubProvider) Historical(ctx context.Context, date time.Time, base string, symbols []string) (Quote, error) {
	return stub.Latest(ctx, base, symbols)
}

func (stub *stubProvider) Timeframe(ctx context.Context, from, to time.Time, base string, symbols []string) ([]Quote, error) {
	if stub.err != nil {
		return nil, stub.err
	}
	return []Quote{stub.latest}, nil
}

func TestCachedProviderPartialOnUpstreamFailure(t *testing.T) {
	dir := t.TempDir()
	cacheDB, err := storage.OpenPriceCache(filepath.Join(dir, "prices.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer cacheDB.Close()

	now := time.Now().UTC()
	if err := cacheDB.UpsertCachedQuote(storage.CachedQuote{
		QuoteDate:    liveQuoteDateKey,
		BaseCurrency: "EUR",
		Symbol:       "XAU",
		Unit:         quoteUnitKilogram,
		PricePerUnit: 3000,
		FetchedAt:    now.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	provider := NewCachedProvider(&stubProvider{err: errors.New("network down")}, cacheDB)
	quote, err := provider.Latest(context.Background(), "EUR", []string{"XAU", "XAG"})
	if err != nil {
		t.Fatalf("expected partial quote, got error: %v", err)
	}
	if !quote.IsPartial || !quote.IsStale {
		t.Fatalf("expected partial stale quote, got %+v", quote)
	}
	if quote.Rates["XAU"] != 3000 {
		t.Fatalf("expected cached XAU, got %v", quote.Rates["XAU"])
	}
}
