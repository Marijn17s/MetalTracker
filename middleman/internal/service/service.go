package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"MetalTracker/middleman/internal/store"
	"MetalTracker/middleman/internal/upstream"
)

// Service serves cached kg quotes and fills gaps from MetalpriceAPI.
type Service struct {
	store    *store.Store
	upstream *upstream.Client
	flights  sync.Map
}

func New(priceStore *store.Store, client *upstream.Client) *Service {
	return &Service{store: priceStore, upstream: client}
}

type QuoteResponse struct {
	Base      string             `json:"base"`
	Timestamp int64              `json:"timestamp"`
	Date      string             `json:"date"`
	Rates     map[string]float64 `json:"rates"`
}

type TimeframeResponse struct {
	Base  string                        `json:"base"`
	Rates map[string]map[string]float64 `json:"rates"`
}

func (service *Service) Latest(ctx context.Context, base string, currencies []string) (QuoteResponse, error) {
	metals := metalSymbols(currencies)
	if len(metals) == 0 {
		metals = store.DefaultMetals()
	}

	rows, err := service.store.Latest(metals)
	if err != nil {
		return QuoteResponse{}, err
	}
	if len(rows) == 0 {
		if err := service.PollLatest(ctx); err != nil {
			return QuoteResponse{}, err
		}
		rows, err = service.store.Latest(metals)
		if err != nil {
			return QuoteResponse{}, err
		}
	}
	if len(rows) == 0 {
		return QuoteResponse{}, fmt.Errorf("no prices available")
	}
	return buildResponse(base, currencies, rows), nil
}

func (service *Service) Historical(ctx context.Context, day time.Time, base string, currencies []string) (QuoteResponse, error) {
	metals := metalSymbols(currencies)
	if len(metals) == 0 {
		metals = store.DefaultMetals()
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	dayUTC := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)

	rows, err := service.store.LatestOnDay(dayUTC, metals)
	if err != nil {
		return QuoteResponse{}, err
	}

	missing := missingMetals(metals, rows)
	if len(missing) > 0 && dayUTC.Before(today) {
		if err := service.fillHistoricalDay(ctx, dayUTC, missing); err != nil {
			return QuoteResponse{}, err
		}
		rows, err = service.store.LatestOnDay(dayUTC, metals)
		if err != nil {
			return QuoteResponse{}, err
		}
	}

	if len(rows) == 0 {
		return QuoteResponse{}, fmt.Errorf("no prices for %s", dayUTC.Format("2006-01-02"))
	}
	return buildResponse(base, currencies, rows), nil
}

func (service *Service) Timeframe(ctx context.Context, from, to time.Time, base string, currencies []string) (TimeframeResponse, error) {
	metals := metalSymbols(currencies)
	if len(metals) == 0 {
		metals = store.DefaultMetals()
	}
	fromUTC := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	toUTC := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	if toUTC.Before(fromUTC) {
		return TimeframeResponse{}, fmt.Errorf("end_date before start_date")
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	existing, err := service.store.DaysInRange(fromUTC, toUTC, metals)
	if err != nil {
		return TimeframeResponse{}, err
	}

	var rangeFrom, rangeTo time.Time
	hasMissing := false
	for day := fromUTC; !day.After(toUTC); day = day.Add(24 * time.Hour) {
		if !day.Before(today) {
			continue
		}
		dayKey := day.Format("2006-01-02")
		dayRows := existing[dayKey]
		if len(missingMetals(metals, dayRows)) > 0 {
			if !hasMissing {
				rangeFrom = day
				hasMissing = true
			}
			rangeTo = day
		}
	}

	if hasMissing {
		if err := service.fillHistoricalRange(ctx, rangeFrom, rangeTo, metals); err != nil {
			return TimeframeResponse{}, err
		}
		existing, err = service.store.DaysInRange(fromUTC, toUTC, metals)
		if err != nil {
			return TimeframeResponse{}, err
		}
	}

	ratesByDate := make(map[string]map[string]float64)
	for day := fromUTC; !day.After(toUTC); day = day.Add(24 * time.Hour) {
		dayKey := day.Format("2006-01-02")
		dayRows := existing[dayKey]
		if len(dayRows) == 0 {
			continue
		}
		built := buildResponse(base, currencies, dayRows)
		ratesByDate[dayKey] = built.Rates
	}
	return TimeframeResponse{
		Base:  strings.ToUpper(base),
		Rates: ratesByDate,
	}, nil
}

// PollLatest stores a snapshot for the current UTC hour (:00), preferring /hourly.
func (service *Service) PollLatest(ctx context.Context) error {
	_, err := service.doSingle("poll-latest", func() (any, error) {
		stamp := time.Now().UTC().Truncate(time.Hour)
		quote, err := service.quoteAtHour(ctx, stamp)
		if err != nil {
			return nil, err
		}
		for _, metal := range store.DefaultMetals() {
			row, ok := rowFromEURQuote(quote, metal, stamp)
			if !ok {
				continue
			}
			if err := service.store.InsertHourly(row); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}

// quoteAtHour loads metals + FX for the UTC hour start via /hourly, falling back to /latest.
func (service *Service) quoteAtHour(ctx context.Context, hour time.Time) (upstream.Quote, error) {
	hourUTC := hour.UTC().Truncate(time.Hour)
	rates := map[string]float64{}
	for _, symbol := range upstream.PollSymbols() {
		hourlyQuote, err := service.upstream.HourlyAt(ctx, hourUTC, "EUR", symbol)
		if err != nil {
			// /hourly may be unavailable on some plans - fall back to live and stamp :00.
			latest, latestErr := service.upstream.Latest(ctx, "EUR", upstream.PollSymbols())
			if latestErr != nil {
				return upstream.Quote{}, err
			}
			latest.Timestamp = hourUTC
			latest.Date = hourUTC.Format("2006-01-02")
			return latest, nil
		}
		for key, value := range hourlyQuote.Rates {
			rates[key] = value
		}
	}
	return upstream.Quote{
		Base:      "EUR",
		Date:      hourUTC.Format("2006-01-02"),
		Timestamp: hourUTC,
		Rates:     rates,
	}, nil
}

func (service *Service) fillHistoricalDay(ctx context.Context, day time.Time, metals []string) error {
	key := "hist-" + day.Format("2006-01-02")
	_, err := service.doSingle(key, func() (any, error) {
		quote, err := service.upstream.Historical(ctx, day, "EUR", upstream.PollSymbols())
		if err != nil {
			return nil, err
		}
		stamp := time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, time.UTC)
		for _, metal := range metals {
			row, ok := rowFromEURQuote(quote, metal, stamp)
			if !ok {
				continue
			}
			if err := service.store.InsertHistoricalIfMissing(row); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}

func (service *Service) fillHistoricalRange(ctx context.Context, from, to time.Time, metals []string) error {
	key := fmt.Sprintf("tf-%s-%s", from.Format("2006-01-02"), to.Format("2006-01-02"))
	_, err := service.doSingle(key, func() (any, error) {
		quotes, err := service.upstream.Timeframe(ctx, from, to, "EUR", upstream.PollSymbols())
		if err != nil {
			return nil, err
		}
		for _, quote := range quotes {
			stamp := time.Date(quote.Timestamp.Year(), quote.Timestamp.Month(), quote.Timestamp.Day(), 12, 0, 0, 0, time.UTC)
			for _, metal := range metals {
				row, ok := rowFromEURQuote(quote, metal, stamp)
				if !ok {
					continue
				}
				if err := service.store.InsertHistoricalIfMissing(row); err != nil {
					return nil, err
				}
			}
		}
		return nil, nil
	})
	return err
}

type flightCall struct {
	wg sync.WaitGroup
	value any
	err   error
}

func (service *Service) doSingle(key string, fn func() (any, error)) (any, error) {
	call := &flightCall{}
	call.wg.Add(1)
	actual, loaded := service.flights.LoadOrStore(key, call)
	if loaded {
		existing := actual.(*flightCall)
		existing.wg.Wait()
		return existing.value, existing.err
	}
	defer func() {
		call.wg.Done()
		service.flights.Delete(key)
	}()
	call.value, call.err = fn()
	return call.value, call.err
}

func rowFromEURQuote(quote upstream.Quote, metal string, stamp time.Time) (store.Price, bool) {
	eur, ok := quote.Rates[metal]
	if !ok || eur <= 0 {
		return store.Price{}, false
	}
	usdRate := quote.Rates["USD"]
	chfRate := quote.Rates["CHF"]
	gbpRate := quote.Rates["GBP"]
	return store.Price{
		DateTime: stamp,
		Metal:    metal,
		EUR:      eur,
		USD:      multiplyPositive(eur, usdRate),
		CHF:      multiplyPositive(eur, chfRate),
		GBP:      multiplyPositive(eur, gbpRate),
	}, true
}

func multiplyPositive(eur, fx float64) float64 {
	if fx <= 0 {
		return 0
	}
	return eur * fx
}

func buildResponse(base string, currencies []string, rows map[string]store.Price) QuoteResponse {
	base = strings.ToUpper(strings.TrimSpace(base))
	if base == "" {
		base = "EUR"
	}
	var stamp time.Time
	var sample store.Price
	for _, row := range rows {
		if stamp.IsZero() || row.DateTime.After(stamp) {
			stamp = row.DateTime
			sample = row
		}
	}

	wanted := currencies
	if len(wanted) == 0 {
		for metal := range rows {
			wanted = append(wanted, metal)
		}
	}

	rates := make(map[string]float64)
	for _, symbol := range wanted {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if symbol == "" {
			continue
		}
		if isMetal(symbol) {
			row, ok := rows[symbol]
			if !ok {
				continue
			}
			if price := priceInCurrency(row, base); price > 0 {
				rates[symbol] = price
			}
			continue
		}
		if fx := fiatPerBase(sample, base, symbol); fx > 0 {
			rates[symbol] = fx
		}
	}

	return QuoteResponse{
		Base:      base,
		Timestamp: stamp.Unix(),
		Date:      stamp.Format("2006-01-02"),
		Rates:     rates,
	}
}

func priceInCurrency(row store.Price, currency string) float64 {
	switch strings.ToUpper(currency) {
	case "EUR":
		return row.EUR
	case "USD":
		return row.USD
	case "CHF":
		return row.CHF
	case "GBP":
		return row.GBP
	default:
		return 0
	}
}

func fiatPerBase(row store.Price, base, fiat string) float64 {
	basePrice := priceInCurrency(row, base)
	fiatPrice := priceInCurrency(row, fiat)
	if basePrice <= 0 || fiatPrice <= 0 {
		return 0
	}
	return fiatPrice / basePrice
}

func metalSymbols(currencies []string) []string {
	metals := make([]string, 0, len(currencies))
	seen := map[string]bool{}
	for _, symbol := range currencies {
		symbol = strings.ToUpper(strings.TrimSpace(symbol))
		if !isMetal(symbol) || seen[symbol] {
			continue
		}
		seen[symbol] = true
		metals = append(metals, symbol)
	}
	return metals
}

func missingMetals(wanted []string, have map[string]store.Price) []string {
	missing := make([]string, 0)
	for _, metal := range wanted {
		if _, ok := have[metal]; !ok {
			missing = append(missing, metal)
		}
	}
	return missing
}

func isMetal(symbol string) bool {
	switch symbol {
	case "XAU", "XAG", "XPT", "XPD":
		return true
	default:
		return false
	}
}
