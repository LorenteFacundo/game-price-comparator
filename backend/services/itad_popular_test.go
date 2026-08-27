package services

import "testing"

func TestPopularDealsFromPricesKeepsPopularityRank(t *testing.T) {
	price := itadPriceResult{ID: "game-1"}
	price.Deals = append(price.Deals, discountedFeaturedITADDeal("Steam", 20))
	deals := popularDealsFromPrices([]itadPriceResult{price}, map[string]itadPopularGame{"game-1": {ID: "game-1", Title: "Top Game", Position: 7}})
	if len(deals) != 1 || deals[0].ITADPopularRank != 7 || deals[0].StoreName != "Steam" {
		t.Fatalf("unexpected popular deals: %+v", deals)
	}
}

func discountedFeaturedITADDeal(store string, discount int) itadDeal {
	deal := itadDeal{}
	deal.Shop.Name = store
	deal.Price = itadMoney{Amount: 10, Currency: "USD"}
	deal.Regular = itadMoney{Amount: 20, Currency: "USD"}
	deal.Cut = discount
	return deal
}
