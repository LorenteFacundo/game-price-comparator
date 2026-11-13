package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeITADAndSteam simula ITAD (precios de AR convertidos a ARS con 1543,21)
// y la tienda de Steam (precios en USD) para calibrar y buscar.
func fakeITADAndSteam(t *testing.T) *httptest.Server {
	t.Helper()
	const rate = 1543.21
	steamUSD := map[string]float64{"1145360": 4.99, "8930": 29.99, "489830": 23.99}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/lookup/id/shop/61/v1":
			var keys []string
			_ = json.NewDecoder(r.Body).Decode(&keys)
			result := map[string]*string{}
			for _, key := range keys {
				id := "itad-" + strings.TrimPrefix(key, "app/")
				result[key] = &id
			}
			_ = json.NewEncoder(w).Encode(result)
		case r.URL.Path == "/games/search/v1":
			_, _ = io.WriteString(w, `[{"id":"itad-1145360","title":"Hades","assets":{}}]`)
		case r.URL.Path == "/games/prices/v3":
			var ids []string
			_ = json.NewDecoder(r.Body).Decode(&ids)
			type money struct {
				Amount   float64 `json:"amount"`
				Currency string  `json:"currency"`
			}
			type deal struct {
				Shop struct {
					Name string `json:"name"`
				} `json:"shop"`
				Price   money  `json:"price"`
				Regular money  `json:"regular"`
				Cut     int    `json:"cut"`
				URL     string `json:"url"`
			}
			newDeal := func(shop string, amount float64) deal {
				d := deal{Price: money{amount, "ARS"}, Regular: money{amount, "ARS"}, URL: "https://itad.link/" + shop}
				d.Shop.Name = shop
				return d
			}
			type result struct {
				ID    string `json:"id"`
				Deals []deal `json:"deals"`
			}
			results := []result{}
			for _, id := range ids {
				usd := steamUSD[strings.TrimPrefix(id, "itad-")]
				if usd == 0 {
					continue
				}
				deals := []deal{newDeal("Steam", round2(usd*rate)), newDeal("Epic Game Store", round2(usd*rate))}
				if id == "itad-1145360" {
					deals = append(deals, newDeal("Microsoft Store", 6999))
				}
				results = append(results, result{ID: id, Deals: deals})
			}
			_ = json.NewEncoder(w).Encode(results)
		case r.URL.Path == "/api/appdetails":
			payload := map[string]any{}
			for _, appID := range strings.Split(r.URL.Query().Get("appids"), ",") {
				if usd, ok := steamUSD[appID]; ok {
					payload[appID] = map[string]any{"success": true, "data": map[string]any{"price_overview": map[string]any{"currency": "USD", "initial": int(usd * 100), "final": int(usd * 100)}}}
				} else {
					payload[appID] = map[string]any{"success": false}
				}
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			t.Fatalf("unexpected request %s", r.URL.Path)
		}
	}))
}

func TestSearchRestoresDollarPricesConvertedByITAD(t *testing.T) {
	server := fakeITADAndSteam(t)
	defer server.Close()
	service := &ITADService{apiKey: "key", client: server.Client(), baseURL: server.URL, steamStoreURL: server.URL}

	results, err := service.Search(context.Background(), "hades")
	if err != nil {
		t.Fatal(err)
	}
	prices := map[string]string{}
	for _, price := range results[0].Prices {
		prices[price.StoreName] = price.Currency
		if price.StoreName == "Steam" && (price.Price != 4.99 || price.Currency != "USD" || price.IsRegional) {
			t.Fatalf("expected Steam US$ 4.99, got %+v", price)
		}
	}
	if prices["Epic Game Store"] != "USD" {
		t.Fatalf("Epic charges in dollars and must be restored to USD: %v", prices)
	}
	if prices["Microsoft Store"] != "ARS" {
		t.Fatalf("Microsoft Store charges in pesos and must stay in ARS: %v", prices)
	}
	if rate := service.arsPerUSD(context.Background()); rate < 1543 || rate > 1543.5 {
		t.Fatalf("unexpected calibrated rate %.2f", rate)
	}
}

func TestPricesStayUntouchedWhenCalibrationFails(t *testing.T) {
	deal := itadDeal{Price: itadMoney{Amount: 7700.62, Currency: "ARS"}, Regular: itadMoney{Amount: 7700.62, Currency: "ARS"}}
	deal.Shop.Name = "Steam"
	if got := restoreDollarDeal(deal, 0); got.Price.Currency != "ARS" || got.Price.Amount != 7700.62 {
		t.Fatalf("without a rate prices must not change: %+v", got)
	}
	if got := restoreDollarDeal(deal, 1543.21); got.Price.Amount != 4.99 || got.Regular.Currency != "USD" {
		t.Fatalf("unexpected restored price %+v", got)
	}
}
