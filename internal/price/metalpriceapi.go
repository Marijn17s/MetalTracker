package price

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

const metalpriceAPIBaseURL = "https://api-eu.metalpriceapi.com/v1"

type MetalpriceAPIClient struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

func NewMetalpriceAPIClient(apiKey string) *MetalpriceAPIClient {
	return &MetalpriceAPIClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: metalpriceAPIBaseURL,
	}
}

func (client *MetalpriceAPIClient) SetAPIKey(apiKey string) {
	client.apiKey = apiKey
}

func (client *MetalpriceAPIClient) Latest(ctx context.Context, base string, symbols []string) (Quote, error) {
	return client.fetch(ctx, "/latest", base, symbols, nil)
}

func (client *MetalpriceAPIClient) Historical(ctx context.Context, date time.Time, base string, symbols []string) (Quote, error) {
	path := "/" + date.Format("2006-01-02")
	return client.fetch(ctx, path, base, symbols, nil)
}

func (client *MetalpriceAPIClient) Timeframe(ctx context.Context, from, to time.Time, base string, symbols []string) ([]Quote, error) {
	if client.apiKey == "" {
		return nil, fmt.Errorf("metalprice API key is not configured")
	}

	query := url.Values{}
	query.Set("api_key", client.apiKey)
	query.Set("base", base)
	query.Set("currencies", strings.Join(symbols, ","))
	query.Set("start_date", from.Format("2006-01-02"))
	query.Set("end_date", to.Format("2006-01-02"))
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
		quotes = append(quotes, Quote{
			Base:      payload.Base,
			Date:      dateKey,
			Timestamp: parseAPIDate(dateKey),
			Rates:     invertMetalRates(rates, symbols),
		})
	}
	return quotes, nil
}

func (client *MetalpriceAPIClient) fetch(ctx context.Context, path, base string, symbols []string, extra url.Values) (Quote, error) {
	if client.apiKey == "" {
		return Quote{}, fmt.Errorf("metalprice API key is not configured")
	}

	query := url.Values{}
	query.Set("api_key", client.apiKey)
	query.Set("base", base)
	query.Set("currencies", strings.Join(symbols, ","))
	query.Set("unit", "kilogram")
	for key, values := range extra {
		for _, value := range values {
			query.Add(key, value)
		}
	}

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
	}

	return Quote{
		Base:      payload.Base,
		Date:      date,
		Timestamp: timestamp,
		Rates:     invertMetalRates(payload.Rates, symbols),
	}, nil
}

// invertMetalRates converts API metal rates (metal per 1 base) into base currency per kg.
func invertMetalRates(rates map[string]float64, symbols []string) map[string]float64 {
	converted := make(map[string]float64, len(rates))
	metalSet := make(map[string]bool, len(symbols))
	for _, symbol := range symbols {
		metalSet[symbol] = isMetalSymbol(symbol)
	}
	for symbol, rate := range rates {
		if rate == 0 {
			continue
		}
		if metalSet[symbol] || isMetalSymbol(symbol) {
			converted[symbol] = 1 / rate
		} else {
			converted[symbol] = rate
		}
	}
	return converted
}

func isMetalSymbol(symbol string) bool {
	switch symbol {
	case "XAU", "XAG", "XPT", "XPD":
		return true
	default:
		return false
	}
}

func parseAPIDate(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Now().UTC()
	}
	return parsed
}
