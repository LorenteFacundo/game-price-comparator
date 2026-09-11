package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"game-price-comparator/models"
	"game-price-comparator/services"
)

type fakeITAD struct {
	deals, popular       []models.FeaturedDeal
	dealsErr, popularErr error
}

func (f fakeITAD) Search(context.Context, string) ([]models.GameResult, error) { return nil, nil }
func (f fakeITAD) GetDeals(context.Context, int) ([]models.FeaturedDeal, error) {
	return f.deals, f.dealsErr
}
func (f fakeITAD) GetPopularDeals(context.Context, int) ([]models.FeaturedDeal, error) {
	return f.popular, f.popularErr
}

type fakeCurrency struct {
	rates services.ExchangeRates
	err   error
}

func (f fakeCurrency) GetRates(context.Context) (services.ExchangeRates, error) {
	return f.rates, f.err
}

type fakeSteam struct {
	featured                []models.FeaturedDeal
	featuredErr, popularErr error
}

func (f fakeSteam) GetPriceByTitle(context.Context, string, string) (*services.SteamPrice, error) {
	return nil, nil
}
func (f fakeSteam) GetTopSellers(context.Context, int) ([]services.SteamTopSeller, error) {
	return nil, f.popularErr
}
func (f fakeSteam) GetSteamFeaturedDeals(context.Context, int) ([]models.FeaturedDeal, error) {
	return f.featured, f.featuredErr
}
func (f fakeSteam) GetDealSignals(context.Context, []models.FeaturedDeal, []services.SteamTopSeller, []models.RankedGame) map[string]services.SteamDealSignal {
	return nil
}
func (f fakeSteam) GetMostPlayed(context.Context, int) ([]models.RankedGame, error) { return nil, nil }

func TestSortPricesAndPickBestPrefersLowestNormalizedPrice(t *testing.T) {
	prices := []models.StorePrice{
		{StoreName: "Steam", Price: 1.24, Currency: "USD"},
		{StoreName: "Microsoft Store", Price: 359, Currency: "ARS"},
		{StoreName: "Eneba", URL: "https://example.com"},
		{StoreName: "MundoSteam", URL: "https://example.com"},
	}

	sorted, best := sortPricesAndPickBest(prices, 1420)
	if best == nil {
		t.Fatal("expected a best deal")
	}

	if best.StoreName != "Microsoft Store" {
		t.Fatalf("expected Microsoft Store as best deal, got %s", best.StoreName)
	}

	if sorted[0].StoreName != "Microsoft Store" {
		t.Fatalf("expected Microsoft Store first after sorting, got %s", sorted[0].StoreName)
	}

	if sorted[len(sorted)-1].StoreName != "MundoSteam" {
		t.Fatalf("expected MundoSteam last, got %s", sorted[len(sorted)-1].StoreName)
	}
}

func TestSortPricesAndPickBestDoesNotCompareUnknownCurrenciesAsARS(t *testing.T) {
	prices := []models.StorePrice{
		{StoreName: "EUR Store", Price: 1, Currency: "EUR"},
		{StoreName: "Steam", Price: 2, Currency: "USD"},
	}

	_, best := sortPricesAndPickBest(prices, 1400)
	if best == nil || best.StoreName != "Steam" {
		t.Fatalf("expected Steam to be the only comparable offer, got %+v", best)
	}
}

func TestWithoutStoreRemovesEverySteamPrice(t *testing.T) {
	prices := []models.StorePrice{
		{StoreName: "Steam", Currency: "ARS", Price: 7549},
		{StoreName: "Epic Games Store", Currency: "ARS", Price: 7549},
		{StoreName: "steam", Currency: "USD", Price: 24.99},
	}

	filtered := withoutStore(prices, "Steam")
	if len(filtered) != 1 || filtered[0].StoreName != "Epic Games Store" {
		t.Fatalf("expected only Epic Games Store after removing Steam, got %+v", filtered)
	}
}

func TestSortPricesAndPickBestTreatsFreeAsBest(t *testing.T) {
	prices := []models.StorePrice{{StoreName: "Epic", Price: 10, Currency: "USD"}, {StoreName: "Steam", Price: 0, Currency: "USD"}}
	_, best := sortPricesAndPickBest(prices, 1400)
	if best == nil || best.StoreName != "Steam" || best.Price != 0 {
		t.Fatalf("expected free Steam offer as best, got %+v", best)
	}
}

func TestHandleDealsReturnsPartialSteamResultsWhenITADFails(t *testing.T) {
	now := time.Now().UTC()
	handler := NewSearchHandler(
		fakeITAD{dealsErr: errors.New("itad down"), popularErr: errors.New("itad down")},
		fakeCurrency{rates: services.ExchangeRates{Official: 1000, Blue: 1200, UpdatedAt: now}},
		fakeSteam{featured: []models.FeaturedDeal{{ID: "steam-1", Title: "Popular Game", StoreName: "Steam", Price: 10, Regular: 20, Currency: "USD", Discount: 50, SteamFeatured: true}}},
		0.21,
	)
	recorder := httptest.NewRecorder()
	handler.HandleDeals(recorder, httptest.NewRequest(http.MethodGet, "/api/deals?limit=12", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var response models.DealsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response.Discounts) != 1 || response.CardRate != 1210 {
		t.Fatalf("unexpected partial response: %+v", response)
	}
	if len(response.Warnings) == 0 {
		t.Fatal("expected provider warnings")
	}
}

func TestHandleDealsReturnsBadGatewayWhenNoSourceHasDeals(t *testing.T) {
	handler := NewSearchHandler(
		fakeITAD{dealsErr: errors.New("down"), popularErr: errors.New("down")},
		fakeCurrency{err: errors.New("down")},
		fakeSteam{featuredErr: errors.New("down"), popularErr: errors.New("down")},
		0.21,
	)
	recorder := httptest.NewRecorder()
	handler.HandleDeals(recorder, httptest.NewRequest(http.MethodGet, "/api/deals", nil))
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", recorder.Code)
	}
}

func TestSearchCachePurgesExpiredAndCapsEntries(t *testing.T) {
	handler := &SearchHandler{searchCache: make(map[string]cachedSearch)}
	handler.searchCache["expired"] = cachedSearch{expiresAt: time.Now().Add(-time.Minute)}
	for index := 0; index < maxSearchCacheEntries; index++ {
		handler.storeSearch(string(rune(index+1000)), models.SearchResponse{Query: "cached"})
	}
	handler.storeSearch("overflow", models.SearchResponse{Query: "new"})
	if len(handler.searchCache) != maxSearchCacheEntries {
		t.Fatalf("expected %d entries, got %d", maxSearchCacheEntries, len(handler.searchCache))
	}
	if _, exists := handler.searchCache["expired"]; exists {
		t.Fatal("expected expired entry to be purged")
	}
}
