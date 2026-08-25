package services

import (
	"testing"

	"game-price-comparator/models"
)

func TestFeaturedStoresOnlyIncludesGamePlatforms(t *testing.T) {
	for _, store := range []string{"Steam", "Epic Games Store", "Epic Game Store", "Microsoft Store"} {
		if !isFeaturedStore(store) {
			t.Fatalf("expected %q to be a featured game platform", store)
		}
	}
	for _, store := range []string{"Humble Store", "Fanatical", "Random Software"} {
		if isFeaturedStore(store) {
			t.Fatalf("did not expect %q to be featured", store)
		}
	}
}

func TestStorePriceFromDealMarksOnlySteamARSAsRegional(t *testing.T) {
	deal := itadDeal{}
	deal.Shop.Name = "Steam"
	deal.Price.Currency = "ARS"
	deal.Price.Amount = 7549
	if !storePriceFromDeal(deal).IsRegional {
		t.Fatal("expected Steam ARS price to be regional")
	}

	deal.Price.Currency = "USD"
	if storePriceFromDeal(deal).IsRegional {
		t.Fatal("did not expect Steam USD price to be regional")
	}
}

func TestFreeDealsAreNotFeatured(t *testing.T) {
	deal := itadDeal{}
	deal.Shop.Name = "Steam"
	deal.Price.Amount = 0
	deal.Cut = 100
	if !isEligibleFeaturedDeal(deal) {
		t.Fatal("expected a free 100% deal to remain in featured offers")
	}

	deal.Price.Amount = 250
	deal.Cut = 30
	if !isEligibleFeaturedDeal(deal) {
		t.Fatal("expected a paid discounted game to be featured")
	}
}

func TestSelectFeaturedDealsMixesFreeAndPaidOffers(t *testing.T) {
	candidates := []models.FeaturedDeal{
		{ID: "free-1", Price: 0, Discount: 100},
		{ID: "free-2", Price: 0, Discount: 100},
		{ID: "paid-1", Price: 100, Discount: 30},
		{ID: "paid-2", Price: 200, Discount: 50},
		{ID: "paid-3", Price: 300, Discount: 70},
	}

	selected := selectFeaturedDeals(candidates, 4)
	if len(selected) != 4 {
		t.Fatalf("expected four selected deals, got %d", len(selected))
	}
	if selected[0].ID != "free-1" || selected[1].ID != "paid-1" || selected[2].ID != "paid-2" || selected[3].ID != "free-2" {
		t.Fatalf("expected free and paid offers interleaved, got %+v", selected)
	}
}
