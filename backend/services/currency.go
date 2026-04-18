package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CurrencyService struct {
	client   *http.Client
	cached   float64
	cacheAge time.Time
}

type bluelyticsResponse struct {
	Blue struct {
		ValueSell float64 `json:"value_sell"`
	} `json:"blue"`
}

func NewCurrencyService() *CurrencyService {
	return &CurrencyService{
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *CurrencyService) GetBlueRate() (float64, error) {
	if c.cached > 0 && time.Since(c.cacheAge) < 10*time.Minute {
		return c.cached, nil
	}

	resp, err := c.client.Get("https://api.bluelytics.com.ar/v2/latest")
	if err != nil {
		if c.cached > 0 {
			return c.cached, nil
		}
		return 0, fmt.Errorf("error obteniendo tipo de cambio: %w", err)
	}
	defer resp.Body.Close()

	var data bluelyticsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("error parseando respuesta: %w", err)
	}

	c.cached = data.Blue.ValueSell
	c.cacheAge = time.Now()
	return c.cached, nil
}
