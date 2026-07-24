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

func TestValueUnitGift(t *testing.T) {
	gift := HoldingUnit{
		SpotWorthAtPurchase: 100,
		IsGift:              true,
		Status:              UnitStatusHeld,
	}
	held := ValueUnit(gift, 250)
	if held.TotalProfit != 250 {
		t.Fatalf("gift unrealized want 250, got %v", held.TotalProfit)
	}
	if held.PremiumPaid != 0 {
		t.Fatalf("gift premium want 0, got %v", held.PremiumPaid)
	}

	salePrice := 300.0
	gift.Status = UnitStatusSold
	gift.SalePrice = &salePrice
	sold := ValueUnit(gift, 0)
	if sold.TotalProfit != 300 {
		t.Fatalf("gift realized want 300, got %v", sold.TotalProfit)
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
