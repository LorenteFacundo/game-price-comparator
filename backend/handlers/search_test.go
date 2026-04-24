package handlers

import (
	"testing"

	"game-price-comparator/models"
)

func TestSortPricesAndPickBestPrefersLowestNormalizedPrice(t *testing.T) {
	prices := []models.StorePrice{
		{StoreName: "Steam", PriceUSD: 1.24},
		{StoreName: "Microsoft Store", PriceARS: 359},
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
