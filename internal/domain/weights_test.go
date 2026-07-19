package domain

import "testing"

func TestFineWeightAndNormalizePurity(t *testing.T) {
	if got := FineWeightGrams(31.1034768, 999.9); got < 31.1 || got > 31.11 {
		t.Fatalf("fine weight unexpected: %v", got)
	}
	if NormalizePurity(999.9) != 0.9999 {
		t.Fatalf("normalize 999.9 = %v", NormalizePurity(999.9))
	}
	if NormalizePurity(92.5) != 0.925 {
		t.Fatalf("normalize 92.5 = %v", NormalizePurity(92.5))
	}
}

func TestIsUnusualPurity(t *testing.T) {
	if IsUnusualPurity(999.9) {
		t.Fatal("999.9 should be common")
	}
	if IsUnusualPurity(0.925) {
		t.Fatal("sterling should be common")
	}
	if !IsUnusualPurity(0.75) {
		t.Fatal("0.75 should be unusual")
	}
}

func TestConvertSpotPricePerKg(t *testing.T) {
	price := 100000.0
	perGram := ConvertSpotPricePerKg(price, SpotPriceUnitGram)
	if perGram < 99.9 || perGram > 100.1 {
		t.Fatalf("per gram unexpected: %v", perGram)
	}
	perOz := ConvertSpotPricePerKg(price, SpotPriceUnitTroyOz)
	if perOz <= perGram {
		t.Fatalf("per troy oz should be larger than per gram: %v", perOz)
	}
	if got := ConvertSpotPricePerKg(price, SpotPriceUnitKilogram); got != price {
		t.Fatalf("kg passthrough: %v", got)
	}
}
