package storage

import (
	"database/sql"
	"time"
)

type CachedQuote struct {
	QuoteDate    string
	BaseCurrency string
	Symbol       string
	Unit         string
	PricePerUnit float64
	FetchedAt    time.Time
}

func (database *DB) GetCachedQuote(quoteDate, baseCurrency, symbol, unit string) (CachedQuote, bool, error) {
	var quote CachedQuote
	var fetchedAt string
	err := database.conn.QueryRow(`
		SELECT quote_date, base_currency, symbol, unit, price_per_unit, fetched_at
		FROM price_quotes
		WHERE quote_date = ? AND base_currency = ? AND symbol = ? AND unit = ?`,
		quoteDate, baseCurrency, symbol, unit,
	).Scan(
		&quote.QuoteDate,
		&quote.BaseCurrency,
		&quote.Symbol,
		&quote.Unit,
		&quote.PricePerUnit,
		&fetchedAt,
	)
	if err == sql.ErrNoRows {
		return quote, false, nil
	}
	if err != nil {
		return quote, false, err
	}
	parsed, err := time.Parse(time.RFC3339, fetchedAt)
	if err != nil {
		return quote, false, err
	}
	quote.FetchedAt = parsed
	return quote, true, nil
}

func (database *DB) UpsertCachedQuote(quote CachedQuote) error {
	_, err := database.conn.Exec(`
		INSERT INTO price_quotes(quote_date, base_currency, symbol, unit, price_per_unit, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(quote_date, base_currency, symbol, unit)
		DO UPDATE SET price_per_unit = excluded.price_per_unit, fetched_at = excluded.fetched_at`,
		quote.QuoteDate,
		quote.BaseCurrency,
		quote.Symbol,
		quote.Unit,
		quote.PricePerUnit,
		quote.FetchedAt.UTC().Format(time.RFC3339),
	)
	return err
}
