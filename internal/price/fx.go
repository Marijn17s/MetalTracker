package price

import (
	"MetalTracker/internal/domain"
)

// SupportedFiatCurrencies are currencies that can appear as purchase or display currency.
func SupportedFiatCurrencies() []domain.Currency {
	return []domain.Currency{
		domain.CurrencyEUR,
		domain.CurrencyUSD,
		domain.CurrencyCHF,
	}
}

// FiatSymbolsForBase returns FX symbols to request alongside metals for a display base.
func FiatSymbolsForBase(base domain.Currency) []string {
	symbols := make([]string, 0, 2)
	for _, currency := range SupportedFiatCurrencies() {
		if currency == base {
			continue
		}
		symbols = append(symbols, string(currency))
	}
	return symbols
}

// ValuationSymbols returns metals plus FX symbols for the display currency.
func ValuationSymbols(base domain.Currency) []string {
	symbols := []string{"XAU", "XAG"}
	symbols = append(symbols, FiatSymbolsForBase(base)...)
	return symbols
}

// ConvertAmount converts amount from fromCurrency into toCurrency using quote rates
// where rates[symbol] means units of symbol per 1 unit of quote.Base (toCurrency).
func ConvertAmount(amount float64, fromCurrency domain.Currency, toCurrency domain.Currency, quote Quote) (float64, float64, bool) {
	if fromCurrency == "" {
		fromCurrency = toCurrency
	}
	if fromCurrency == toCurrency || amount == 0 {
		return amount, 1, true
	}
	if quote.Base != "" && quote.Base != string(toCurrency) {
		return amount, 0, false
	}
	rate, ok := quote.Rates[string(fromCurrency)]
	if !ok || rate <= 0 {
		return amount, 0, false
	}
	// rate = fromCurrency units per 1 toCurrency -> amount_to = amount_from / rate
	return amount / rate, rate, true
}
