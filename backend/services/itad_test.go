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

func TestBestRadarDealKeepsTheBestDiscountAndAllMatchedStores(t *testing.T) {
	steam := itadDeal{}
	steam.Shop.Name = "Steam"
	steam.Price = itadMoney{Amount: 1000, Currency: "ARS"}
	steam.Regular = itadMoney{Amount: 2000, Currency: "ARS"}
	steam.Cut = 50
	steam.URL = "https://store.steampowered.com/app/1"
	epic := itadDeal{}
	epic.Shop.Name = "Epic Game Store"
	epic.Price = itadMoney{Amount: 800, Currency: "ARS"}
	epic.Regular = itadMoney{Amount: 2000, Currency: "ARS"}
	epic.Cut = 60
	epic.URL = "https://store.epicgames.com/"

	deal, ok := bestRadarDeal([]itadDeal{steam, epic}, SteamTopSeller{Title: "Hades", Rank: 2}, itadSearchResult{ID: "hades", Title: "Hades"})
	if !ok {
		t.Fatal("expected a radar deal")
	}
	if deal.StoreName != "Epic Game Store" || deal.PopularityRank != 2 {
		t.Fatalf("expected Epic as rank 2 best deal, got %+v", deal)
	}
	if len(deal.MatchedStores) != 2 {
		t.Fatalf("expected two matched stores, got %+v", deal.MatchedStores)
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
