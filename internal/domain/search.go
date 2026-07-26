package domain

import "strings"

func containsFold(haystack string, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func MetalSearchTerms(metal MetalSymbol) []string {
	terms := []string{string(metal), MetalDisplayName(metal)}
	switch metal {
	case MetalGold:
		terms = append(terms, "goud")
	case MetalSilver:
		terms = append(terms, "zilver")
	}
	return terms
}

func FormSearchTerms(form Form) []string {
	return []string{string(form), FormDisplayName(form)}
}

func termMatchesQuery(term string, query string) bool {
	lowerTerm := strings.ToLower(strings.TrimSpace(term))
	if lowerTerm == "" || query == "" {
		return false
	}
	if lowerTerm == query {
		return true
	}
	// Prefix match: "gol" -> gold, "xa" -> xau
	if strings.HasPrefix(lowerTerm, query) {
		return true
	}
	// Query contains a full alias: "gold maple" -> gold
	if len(lowerTerm) >= 3 && strings.Contains(query, lowerTerm) {
		return true
	}
	return false
}

func UnitMatchesSearch(unit HoldingUnit, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	for _, term := range MetalSearchTerms(unit.Metal) {
		if termMatchesQuery(term, query) {
			return true
		}
	}
	for _, term := range FormSearchTerms(unit.Form) {
		if termMatchesQuery(term, query) {
			return true
		}
	}

	fields := []string{
		unit.Brand,
		unit.ProductName,
		unit.Dealer,
		unit.Notes,
		unit.StorageLocation,
		unit.Condition,
		string(unit.Currency),
	}
	for _, field := range fields {
		if containsFold(field, query) {
			return true
		}
	}
	return false
}

func UnitMatchesHoldingsFilter(unit HoldingUnit, filter HoldingsFilter) bool {
	if !UnitMatchesSearch(unit, filter.Search) {
		return false
	}
	if len(filter.Metals) > 0 && !stringInFold(filter.Metals, string(unit.Metal)) {
		return false
	}
	if len(filter.Forms) > 0 && !stringInFold(filter.Forms, string(unit.Form)) {
		return false
	}
	if len(filter.Brands) > 0 {
		brand := strings.TrimSpace(unit.Brand)
		if brand == "" {
			brand = "Unknown"
		}
		if !stringInFold(filter.Brands, brand) {
			return false
		}
	}
	if len(filter.Locations) > 0 {
		location := strings.TrimSpace(unit.StorageLocation)
		if location == "" {
			location = "Unset"
		}
		if !stringInFold(filter.Locations, location) {
			return false
		}
	}
	if len(filter.Weights) > 0 && !floatIn(filter.Weights, unit.WeightGrams) {
		return false
	}
	return true
}

func stringInFold(values []string, candidate string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func floatIn(values []float64, candidate float64) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
