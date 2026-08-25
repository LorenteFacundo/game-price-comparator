package services

import "testing"

func TestStorePriceFromDealPreservesCurrency(t *testing.T) {
	deal := itadDeal{Cut: 50, URL: "https://example.com"}
	deal.Shop.Name = "Example Store"
	deal.Price.Amount = 9.99
	deal.Price.Currency = "USD"
	deal.Regular.Amount = 19.99
	deal.Regular.Currency = "USD"

	price := storePriceFromDeal(deal)
	if price.Currency != "USD" || price.Price != 9.99 || price.Regular != 19.99 {
		t.Fatalf("currency or amounts were not preserved: %+v", price)
	}
}
