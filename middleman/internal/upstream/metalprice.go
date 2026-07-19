package upstream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://api-eu.metalpriceapi.com/v1"

// Quote is currency-per-kg metal rates (after invert) plus fiat FX vs base.
type Quote struct {
	Base      string
	Date      string
	Timestamp time.Time
	Rates     map[string]float64
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (client *Client) Latest(ctx context.Context, base string, symbols []string) (Quote, error) {
	return client.fetch(ctx, "/latest", base, symbols)
}

func (client *Client) Historical(ctx context.Context, date time.Time, base string, symbols []string) (Quote, error) {
	path := "/" + date.UTC().Format("2006-01-02")
	return client.fetch(ctx, path, base, symbols)
}

// HourlyAt fetches the rate for one currency at the UTC hour start (minute 00).
// MetalpriceAPI /hourly accepts a single currency per request.
func (client *Client) HourlyAt(ctx context.Context, hour time.Time, base string, currency string) (Quote, error) {
	if client.apiKey == "" {
		return Quote{}, fmt.Errorf("metalprice API key is not configured")
	}

	hourUTC := hour.UTC().Truncate(time.Hour)
	query := url.Values{}
	query.Set("api_key", client.apiKey)
	query.Set("base", base)
	query.Set("currency", strings.ToUpper(currency))
	query.Set("start", fmt.Sprintf("%d", hourUTC.Unix()))
	query.Set("end", fmt.Sprintf("%d", hourUTC.Unix()))
	query.Set("unit", "kilogram")

	requestURL := client.baseURL + "/hourly?" + query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return Quote{}, err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return Quote{}, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Quote{}, err
	}
	if response.StatusCode >= 400 {
		return Quote{}, fmt.Errorf("metalpriceapi hourly error: %s", string(body))
	}

	var payload struct {
		Success bool   `json:"success"`
		Base    string `json:"base"`
		Rates   []struct {
			Timestamp int64              `json:"timestamp"`
			Rates     map[string]float64 `json:"rates"`
		} `json:"rates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Quote{}, err
	}
	if payload.Error != nil {
		return Quote{}, fmt.Errorf("metalpriceapi: %s", payload.Error.Message)
	}
	if !payload.Success {
		return Quote{}, fmt.Errorf("metalpriceapi hourly request failed")
	}

	targetUnix := hourUTC.Unix()
	var matchedRates map[string]float64
	for _, entry := range payload.Rates {
		entryHour := time.Unix(entry.Timestamp, 0).UTC().Truncate(time.Hour).Unix()
		if entryHour == targetUnix {
			matchedRates = entry.Rates
			break
		}
	}
	if matchedRates == nil && len(payload.Rates) == 1 {
		matchedRates = payload.Rates[0].Rates
	}
	if matchedRates == nil {
		return Quote{}, fmt.Errorf("metalpriceapi: no hourly rate for %s", hourUTC.Format(time.RFC3339))
	}

	return Quote{
		Base:      payload.Base,
		Date:      hourUTC.Format("2006-01-02"),
		Timestamp: hourUTC,
		Rates:     invertMetalRates(matchedRates, []string{currency}),
	}, nil
}

func (client *Client) Timeframe(ctx context.Context, from, to time.Time, base string, symbols []string) ([]Quote, error) {
	if client.apiKey == "" {
		return nil, fmt.Errorf("metalprice API key is not configured")
	}

	query := url.Values{}
	query.Set("api_key", client.apiKey)
	query.Set("base", base)
	query.Set("currencies", strings.Join(symbols, ","))
	query.Set("start_date", from.UTC().Format("2006-01-02"))
	query.Set("end_date", to.UTC().Format("2006-01-02"))
	query.Set("unit", "kilogram")

	requestURL := client.baseURL + "/timeframe?" + query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("metalpriceapi timeframe error: %s", string(body))
	}

	var payload struct {
		Success bool                          `json:"success"`
		Base    string                        `json:"base"`
		Rates   map[string]map[string]float64 `json:"rates"`
		Error   *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Error != nil {
		return nil, fmt.Errorf("metalpriceapi: %s", payload.Error.Message)
	}
	if !payload.Success {
		return nil, fmt.Errorf("metalpriceapi timeframe request failed")
	}

	quotes := make([]Quote, 0, len(payload.Rates))
	for dateKey, rates := range payload.Rates {
		parsed, _ := time.Parse("2006-01-02", dateKey)
		quotes = append(quotes, Quote{
			Base:      payload.Base,
			Date:      dateKey,
			Timestamp: parsed.UTC(),
			Rates:     invertMetalRates(rates, symbols),
		})
	}
	return quotes, nil
}

func (client *Client) fetch(ctx context.Context, path, base string, symbols []string) (Quote, error) {
	if client.apiKey == "" {
		return Quote{}, fmt.Errorf("metalprice API key is not configured")
	}

	query := url.Values{}
	query.Set("api_key", client.apiKey)
	query.Set("base", base)
	query.Set("currencies", strings.Join(symbols, ","))
	query.Set("unit", "kilogram")

	requestURL := client.baseURL + path + "?" + query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return Quote{}, err
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return Quote{}, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Quote{}, err
	}
	if response.StatusCode >= 400 {
		return Quote{}, fmt.Errorf("metalpriceapi error: %s", string(body))
	}

	var payload struct {
		Success   bool               `json:"success"`
		Base      string             `json:"base"`
		Timestamp int64              `json:"timestamp"`
		Rates     map[string]float64 `json:"rates"`
		Error     *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Quote{}, err
	}
	if payload.Error != nil {
		return Quote{}, fmt.Errorf("metalpriceapi: %s", payload.Error.Message)
	}
	if !payload.Success {
		return Quote{}, fmt.Errorf("metalpriceapi request failed")
	}

	timestamp := time.Unix(payload.Timestamp, 0).UTC()
	date := timestamp.Format("2006-01-02")
	if strings.HasPrefix(path, "/") && len(path) == 11 {
		date = strings.TrimPrefix(path, "/")
		if parsed, err := time.Parse("2006-01-02", date); err == nil {
			timestamp = parsed.UTC()
		}
	}

	return Quote{
		Base:      payload.Base,
		Date:      date,
		Timestamp: timestamp,
		Rates:     invertMetalRates(payload.Rates, symbols),
	}, nil
}

func invertMetalRates(rates map[string]float64, symbols []string) map[string]float64 {
	converted := make(map[string]float64, len(rates))
	for symbol, rate := range rates {
		if rate == 0 {
			continue
		}
		if isMetalSymbol(symbol) {
			converted[symbol] = 1 / rate
		} else {
			converted[symbol] = rate
		}
	}
	_ = symbols
	return converted
}

func isMetalSymbol(symbol string) bool {
	switch strings.ToUpper(symbol) {
	case "XAU", "XAG", "XPT", "XPD":
		return true
	default:
		return false
	}
}

// PollSymbols are metals + FX needed to fill the wide price table from a EUR base quote.
func PollSymbols() []string {
	return []string{"XAU", "XAG", "XPT", "XPD", "USD", "CHF", "GBP"}
}
