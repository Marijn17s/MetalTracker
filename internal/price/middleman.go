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

// MiddlemanProvider talks to a shared caching API with the same semantics
// as MetalpriceAPI (rates as currency per kg after invert), reducing duplicate upstream requests.
type MiddlemanProvider struct {
	baseURL    string
	httpClient *http.Client
}

func NewMiddlemanProvider(baseURL string) *MiddlemanProvider {
	return &MiddlemanProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (provider *MiddlemanProvider) SetBaseURL(baseURL string) {
	provider.baseURL = strings.TrimRight(baseURL, "/")
}

func (provider *MiddlemanProvider) Latest(ctx context.Context, base string, symbols []string) (Quote, error) {
	return provider.get(ctx, "/v1/latest", base, symbols, nil)
}

func (provider *MiddlemanProvider) Historical(ctx context.Context, date time.Time, base string, symbols []string) (Quote, error) {
	path := "/v1/" + date.Format("2006-01-02")
	return provider.get(ctx, path, base, symbols, nil)
}

func (provider *MiddlemanProvider) Timeframe(ctx context.Context, from, to time.Time, base string, symbols []string) ([]Quote, error) {
	if provider.baseURL == "" {
		return nil, fmt.Errorf("middleman base URL is not configured")
	}

	query := url.Values{}
	query.Set("base", base)
	query.Set("currencies", strings.Join(symbols, ","))
	query.Set("start_date", from.Format("2006-01-02"))
	query.Set("end_date", to.Format("2006-01-02"))

	requestURL := provider.baseURL + "/v1/timeframe?" + query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	response, err := provider.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("middleman timeframe error: %s", string(body))
	}

	var payload struct {
		Base  string                        `json:"base"`
		Rates map[string]map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	quotes := make([]Quote, 0, len(payload.Rates))
	for dateKey, rates := range payload.Rates {
		quotes = append(quotes, Quote{
			Base:      payload.Base,
			Date:      dateKey,
			Timestamp: parseAPIDate(dateKey),
			Rates:     rates,
		})
	}
	return quotes, nil
}

func (provider *MiddlemanProvider) get(ctx context.Context, path, base string, symbols []string, extra url.Values) (Quote, error) {
	if provider.baseURL == "" {
		return Quote{}, fmt.Errorf("middleman base URL is not configured")
	}

	query := url.Values{}
	query.Set("base", base)
	query.Set("currencies", strings.Join(symbols, ","))
	for key, values := range extra {
		for _, value := range values {
			query.Add(key, value)
		}
	}

	requestURL := provider.baseURL + path + "?" + query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return Quote{}, err
	}

	response, err := provider.httpClient.Do(request)
	if err != nil {
		return Quote{}, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Quote{}, err
	}
	if response.StatusCode >= 400 {
		return Quote{}, fmt.Errorf("middleman error: %s", string(body))
	}

	var payload struct {
		Base      string             `json:"base"`
		Timestamp int64              `json:"timestamp"`
		Date      string             `json:"date"`
		Rates     map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Quote{}, err
	}

	timestamp := time.Now().UTC()
	if payload.Timestamp > 0 {
		timestamp = time.Unix(payload.Timestamp, 0).UTC()
	}
	date := payload.Date
	if date == "" {
		date = timestamp.Format("2006-01-02")
	}

	return Quote{
		Base:      payload.Base,
		Date:      date,
		Timestamp: timestamp,
		Rates:     payload.Rates,
	}, nil
}
