package services

import "testing"

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
