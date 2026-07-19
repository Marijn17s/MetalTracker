package domain

import "testing"

func TestBreakEvenAndAvgCostPerKg(t *testing.T) {
	// 1 kg fine at €2000 purchase -> break-even €2000/kg
	fine := GramsPerKilogram
	if got := BreakEvenSpotPerKg(2000, fine); got < 1999.9 || got > 2000.1 {
		t.Fatalf("break-even unexpected: %v", got)
	}
	if got := AvgCostPerKgFine(4000, fine*2); got < 1999.9 || got > 2000.1 {
		t.Fatalf("avg cost unexpected: %v", got)
	}
}

func TestSpotPriceToPerKg(t *testing.T) {
	perGram := 100.0
	perKg := SpotPriceToPerKg(perGram, SpotPriceUnitGram)
	if perKg < 99999 || perKg > 100001 {
		t.Fatalf("from gram unexpected: %v", perKg)
	}
	if got := SpotPriceToPerKg(3100, SpotPriceUnitKilogram); got != 3100 {
		t.Fatalf("kg passthrough: %v", got)
	}
}
