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
	cached   float64
	cacheAge time.Time
}

type bluelyticsResponse struct {
	Blue struct {
		ValueSell float64 `json:"value_sell"`
	} `json:"blue"`
}

func NewCurrencyService() *CurrencyService {
	return &CurrencyService{client: &http.Client{Timeout: 5 * time.Second}}
}

func (c *CurrencyService) GetBlueRate(ctx context.Context) (float64, error) {
	c.mu.RLock()
	if c.cached > 0 && time.Since(c.cacheAge) < 10*time.Minute {
		rate := c.cached
		c.mu.RUnlock()
		return rate, nil
	}
	cached := c.cached
	c.mu.RUnlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.bluelytics.com.ar/v2/latest", nil)
	if err != nil {
		return 0, fmt.Errorf("creando consulta de tipo de cambio: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		if cached > 0 {
			return cached, nil
		}
		return 0, fmt.Errorf("obteniendo tipo de cambio: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("tipo de cambio respondió con status %d", resp.StatusCode)
	}
	var data bluelyticsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("parseando tipo de cambio: %w", err)
	}
	if data.Blue.ValueSell <= 0 {
		return 0, fmt.Errorf("tipo de cambio inválido")
	}
	c.mu.Lock()
	c.cached = data.Blue.ValueSell
	c.cacheAge = time.Now()
	c.mu.Unlock()
	return data.Blue.ValueSell, nil
}
