package services

import (
	"fmt"
	"testing"

	"game-price-comparator/models"
)

func TestRankDealsSeparatesFreeAndRewardsRelevantGames(t *testing.T) {
	deals := []models.FeaturedDeal{
		{ID: "known", Title: "Great Game", Price: 20, Discount: 40, HistoryLow: 20, IsNearLow: true},
		{ID: "obscure", Title: "Unknown Levels Pack", Price: 1, Discount: 95},
		{ID: "free", Title: "Free Favorite", Price: 0, Discount: 100},
	}
	signals := map[string]SteamDealSignal{
		"known": {AppID: "10", ReviewPct: 94, ReviewCount: 50000, PopularRank: 3, Players: 80000},
		"free":  {AppID: "20", ReviewPct: 90, ReviewCount: 1000},
	}

	result := RankDeals(deals, signals, 12)
	if len(result.Free) != 1 || result.Free[0].ID != "free" {
		t.Fatalf("expected the free game in its own collection, got %+v", result.Free)
	}
	if len(result.Featured) < 2 || result.Featured[0].ID != "known" {
		t.Fatalf("expected the acclaimed popular game first, got %+v", result.Featured)
	}
	if result.Discounts[0].ID != "obscure" {
		t.Fatalf("expected raw discount view to preserve the biggest discount, got %+v", result.Discounts)
	}
	if len(result.Featured[0].Reasons) != 2 {
		t.Fatalf("expected at most two concise reasons, got %+v", result.Featured[0].Reasons)
	}
}

func TestRankDealsDeduplicatesEditions(t *testing.T) {
	deals := []models.FeaturedDeal{
		{ID: "base", Title: "Hades", Price: 10, Discount: 50},
		{ID: "deluxe", Title: "Hades Deluxe Edition", Price: 12, Discount: 60},
	}
	result := RankDeals(deals, nil, 12)
	if len(result.Featured) != 1 {
		t.Fatalf("expected one Hades edition in featured, got %+v", result.Featured)
	}
}

func TestSelectDealsForSignalsSamplesAcrossDiscountRange(t *testing.T) {
	deals := make([]models.FeaturedDeal, 0, 40)
	for index := 0; index < 40; index++ {
		deals = append(deals, models.FeaturedDeal{ID: fmt.Sprintf("deal-%d", index), Title: fmt.Sprintf("Game %d", index), Price: 10, Discount: 100 - index*2})
	}
	selected := selectDealsForSignals(deals, nil, nil, 8)
	if len(selected) != 8 {
		t.Fatalf("expected eight samples, got %d", len(selected))
	}
	if selected[len(selected)-1].Discount > 40 {
		t.Fatalf("expected lower discounts to be sampled, got %+v", selected)
	}
}
