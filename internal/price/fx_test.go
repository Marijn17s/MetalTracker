package price

import (
	"testing"

	"MetalTracker/internal/domain"
)

func TestConvertAmountSameCurrency(t *testing.T) {
	amount, rate, ok := ConvertAmount(100, domain.CurrencyEUR, domain.CurrencyEUR, Quote{})
	if !ok || amount != 100 || rate != 1 {
		t.Fatalf("same currency failed: %v %v %v", amount, rate, ok)
	}
}

func TestConvertAmountWithFX(t *testing.T) {
	quote := Quote{
		Base: "EUR",
		Rates: map[string]float64{
			"USD": 1.10,
		},
	}
	amount, rate, ok := ConvertAmount(110, domain.CurrencyUSD, domain.CurrencyEUR, quote)
	if !ok {
		t.Fatal("expected conversion ok")
	}
	if rate != 1.10 {
		t.Fatalf("rate = %v", rate)
	}
	if amount < 99.9 || amount > 100.1 {
		t.Fatalf("converted amount = %v", amount)
	}
}

func TestValuationSymbolsIncludeFiat(t *testing.T) {
	symbols := ValuationSymbols(domain.CurrencyEUR)
	hasUSD := false
	hasXAU := false
	for _, symbol := range symbols {
		if symbol == "USD" {
			hasUSD = true
		}
		if symbol == "XAU" {
			hasXAU = true
		}
		if symbol == "EUR" {
			t.Fatal("base currency should not be requested as FX symbol")
		}
	}
	if !hasUSD || !hasXAU {
		t.Fatalf("symbols incomplete: %v", symbols)
	}
}
