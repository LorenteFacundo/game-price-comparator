package services

import "testing"

func TestFeaturedStoresOnlyIncludesGamePlatforms(t *testing.T) {
	for _, store := range []string{"Steam", "Epic Games Store", "Microsoft Store"} {
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
