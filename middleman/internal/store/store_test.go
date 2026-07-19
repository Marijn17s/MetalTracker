package store_test

import (
	"path/filepath"
	"testing"
	"time"

	"MetalTracker/middleman/internal/store"
)

func TestInsertHourlyAllowsMultiplePerDay(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "prices.db")
	priceStore, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer priceStore.Close()

	day := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	for hour := 10; hour <= 12; hour++ {
		err := priceStore.InsertHourly(store.Price{
			DateTime: time.Date(2026, 7, 21, hour, 0, 0, 0, time.UTC),
			Metal:    "XAU",
			EUR:      float64(1000 + hour),
			USD:      1100,
			CHF:      950,
			GBP:      900,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	latest, err := priceStore.Latest([]string{"XAU"})
	if err != nil {
		t.Fatal(err)
	}
	if latest["XAU"].EUR != 1012 {
		t.Fatalf("expected latest EUR 1012, got %v", latest["XAU"].EUR)
	}

	onDay, err := priceStore.LatestOnDay(day, []string{"XAU"})
	if err != nil {
		t.Fatal(err)
	}
	if onDay["XAU"].EUR != 1012 {
		t.Fatalf("expected day latest 1012, got %v", onDay["XAU"].EUR)
	}
}

func TestHistoricalInsertOncePerDay(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "prices.db")
	priceStore, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer priceStore.Close()

	day := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	row := store.Price{DateTime: day, Metal: "XAG", EUR: 800, USD: 900, CHF: 750, GBP: 700}
	if err := priceStore.InsertHistoricalIfMissing(row); err != nil {
		t.Fatal(err)
	}
	row.EUR = 999
	if err := priceStore.InsertHistoricalIfMissing(row); err != nil {
		t.Fatal(err)
	}
	onDay, err := priceStore.LatestOnDay(day, []string{"XAG"})
	if err != nil {
		t.Fatal(err)
	}
	if onDay["XAG"].EUR != 800 {
		t.Fatalf("historical should not overwrite, got %v", onDay["XAG"].EUR)
	}
}
