package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBluelyticsResponseReadsOficialField(t *testing.T) {
	var response bluelyticsResponse
	err := json.Unmarshal([]byte(`{"blue":{"value_sell":1555},"oficial":{"value_sell":1533}}`), &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.Official.ValueSell != 1533 {
		t.Fatalf("expected oficial value 1533, got %.2f", response.Official.ValueSell)
	}
}

func TestCurrencyServiceUsesMarkedStaleCacheForAtMostSixHours(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "down", http.StatusBadGateway) }))
	defer server.Close()
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	service := NewCurrencyService()
	service.client = server.Client()
	service.endpoint = server.URL
	service.now = func() time.Time { return now }
	service.cached = ExchangeRates{Blue: 1300, Official: 1000, UpdatedAt: now.Add(-5 * time.Hour)}
	service.cacheAge = now.Add(-5 * time.Hour)
	rates, err := service.GetRates(context.Background())
	if err != nil || !rates.Stale || !rates.UpdatedAt.Equal(service.cacheAge) {
		t.Fatalf("expected marked stale cache, got %+v, %v", rates, err)
	}
	service.cacheAge = now.Add(-7 * time.Hour)
	if _, err := service.GetRates(context.Background()); err == nil {
		t.Fatal("expected cache older than six hours to be rejected")
	}
}
