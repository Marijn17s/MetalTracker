package domain

import "testing"

func TestUnitMatchesSearchMetalNames(t *testing.T) {
	gold := HoldingUnit{Metal: MetalGold, Form: FormBar, Brand: "PAMP", ProductName: "Fortuna"}
	silver := HoldingUnit{Metal: MetalSilver, Form: FormCoin, Brand: "Maple", ProductName: "Leaf"}

	cases := []struct {
		unit    HoldingUnit
		query   string
		want    bool
		message string
	}{
		{gold, "gold", true, "gold should match gold metal"},
		{gold, "goud", true, "goud should match gold metal"},
		{gold, "xau", true, "xau should match gold metal"},
		{gold, "silver", false, "silver must not match gold"},
		{silver, "silver", true, "silver should match silver metal"},
		{silver, "zilver", true, "zilver should match silver metal"},
		{silver, "xag", true, "xag should match silver metal"},
		{silver, "gold", false, "gold must not match silver"},
		{gold, "pamp", true, "brand search should work"},
		{silver, "leaf", true, "product name search should work"},
		{gold, "coin", false, "form mismatch"},
		{silver, "coin", true, "form match"},
	}

	for _, testCase := range cases {
		got := UnitMatchesSearch(testCase.unit, testCase.query)
		if got != testCase.want {
			t.Fatalf("%s: query=%q got=%v want=%v", testCase.message, testCase.query, got, testCase.want)
		}
	}
}

func TestUnitMatchesHoldingsFilter(t *testing.T) {
	unit := HoldingUnit{
		Metal:    MetalGold,
		Form:     FormBar,
		Brand:    "Argor",
		Currency: CurrencyEUR,
	}
	if !UnitMatchesHoldingsFilter(unit, HoldingsFilter{Metals: []string{"XAU"}}) {
		t.Fatal("expected metal filter to match")
	}
	if UnitMatchesHoldingsFilter(unit, HoldingsFilter{Metals: []string{"XAG"}}) {
		t.Fatal("expected metal filter to reject silver")
	}
	if !UnitMatchesHoldingsFilter(unit, HoldingsFilter{Brands: []string{"Argor"}, Forms: []string{"bar"}}) {
		t.Fatal("expected brand/form filters to match")
	}
	unit.StorageLocation = "Home safe"
	if !UnitMatchesHoldingsFilter(unit, HoldingsFilter{Locations: []string{"Home safe"}}) {
		t.Fatal("expected location filter to match")
	}
	if UnitMatchesHoldingsFilter(unit, HoldingsFilter{Locations: []string{"Bank"}}) {
		t.Fatal("expected location filter to reject other location")
	}
}
