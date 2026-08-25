package handlers

import (
	"testing"

	"game-price-comparator/models"
)

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
