package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type CurrencyService struct {
	client   *http.Client
	mu       sync.RWMutex
	cached   ExchangeRates
	cacheAge time.Time
}
type ExchangeRates struct {
	Blue     float64
	Official float64
}

type bluelyticsResponse struct {
	Blue struct {
		ValueSell float64 `json:"value_sell"`
	} `json:"blue"`
	Official struct {
		ValueSell float64 `json:"value_sell"`
	} `json:"official"`
}

func NewCurrencyService() *CurrencyService {
	return &CurrencyService{client: &http.Client{Timeout: 5 * time.Second}}
}

func (c *CurrencyService) GetBlueRate(ctx context.Context) (float64, error) {
	rates, err := c.GetRates(ctx)
	return rates.Blue, err
}

func (c *CurrencyService) GetRates(ctx context.Context) (ExchangeRates, error) {
	c.mu.RLock()
	if c.cached.Blue > 0 && c.cached.Official > 0 && time.Since(c.cacheAge) < 10*time.Minute {
		rates := c.cached
		c.mu.RUnlock()
		return rates, nil
	}
	cached := c.cached
	c.mu.RUnlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.bluelytics.com.ar/v2/latest", nil)
	if err != nil {
		return ExchangeRates{}, fmt.Errorf("creando consulta de tipo de cambio: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		if cached.Blue > 0 && cached.Official > 0 {
			return cached, nil
		}
		return ExchangeRates{}, fmt.Errorf("obteniendo tipo de cambio: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ExchangeRates{}, fmt.Errorf("tipo de cambio respondió con status %d", resp.StatusCode)
	}
	var data bluelyticsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ExchangeRates{}, fmt.Errorf("parseando tipo de cambio: %w", err)
	}
	if data.Blue.ValueSell <= 0 || data.Official.ValueSell <= 0 {
		return ExchangeRates{}, fmt.Errorf("tipos de cambio inválidos")
	}
	rates := ExchangeRates{Blue: data.Blue.ValueSell, Official: data.Official.ValueSell}
	c.mu.Lock()
	c.cached = rates
	c.cacheAge = time.Now()
	c.mu.Unlock()
	return rates, nil
}
