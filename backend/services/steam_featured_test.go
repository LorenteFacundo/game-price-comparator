package services

import "testing"

func TestSteamFeaturedDealKeepsSteamSaleMetadata(t *testing.T) {
	item := steamFeaturedItem{ID: 123, Type: 0, Name: "Top Tier Game", Currency: "USD", FinalPrice: 1599, OriginalPrice: 1999, DiscountPercent: 20, DiscountExpiration: 1_800_000_000}
	deal, ok := steamFeaturedDeal(item)
	if !ok {
		t.Fatal("expected a discounted Steam game to be accepted")
	}
	if !deal.SteamFeatured || deal.SteamAppID != "123" || deal.Price != 15.99 || deal.Discount != 20 {
		t.Fatalf("unexpected Steam featured deal: %+v", deal)
	}
}
