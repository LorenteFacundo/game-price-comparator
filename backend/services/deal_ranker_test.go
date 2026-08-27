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
	if len(result.Featured) == 0 || result.Featured[0].ID != "known" {
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
		{ID: "base", Title: "Hades", Price: 10, Discount: 50, SteamFeatured: true},
		{ID: "deluxe", Title: "Hades Deluxe Edition", Price: 12, Discount: 60, SteamFeatured: true},
	}
	result := RankDeals(deals, map[string]SteamDealSignal{
		"base":   {ReviewPct: 98, ReviewCount: 10000},
		"deluxe": {ReviewPct: 98, ReviewCount: 10000},
	}, 12)
	if len(result.Featured) != 1 {
		t.Fatalf("expected one Hades edition in featured, got %+v", result.Featured)
	}
}

func TestRankDealsUsesITADPopularityAsATopTierSignal(t *testing.T) {
	result := RankDeals([]models.FeaturedDeal{{ID: "catalog", Title: "Top Catalog Game", Price: 20, Discount: 20, ITADPopularRank: 4}}, nil, 12)
	if len(result.Featured) != 1 || result.Featured[0].ID != "catalog" {
		t.Fatalf("expected popular catalog game in featured, got %+v", result.Featured)
	}
}

func TestRankDealsExcludesObscureHighDiscountFromFeatured(t *testing.T) {
	deals := []models.FeaturedDeal{
		{ID: "top", Title: "Clair Obscur: Expedition 33", Price: 30, Discount: 20},
		{ID: "random", Title: "Random Word Game 2", Price: 1, Discount: 95, IsNearLow: true},
		{ID: "poorly-rated", Title: "Popular but Poorly Rated Game", Price: 10, Discount: 75},
	}
	signals := map[string]SteamDealSignal{
		"top":          {ReviewPct: 96, ReviewCount: 70000},
		"random":       {ReviewPct: 59, ReviewCount: 40},
		"poorly-rated": {ReviewPct: 64, ReviewCount: 10000},
	}

	result := RankDeals(deals, signals, 12)
	if len(result.Featured) != 1 || result.Featured[0].ID != "top" {
		t.Fatalf("expected only the high-signal game in featured, got %+v", result.Featured)
	}
	if len(result.Discounts) != 3 || result.Discounts[0].ID != "random" {
		t.Fatalf("expected the raw discount list to retain the 95%% game, got %+v", result.Discounts)
	}
}

func TestRankDealsPrefersPopularityOverDiscountPercentage(t *testing.T) {
	deals := []models.FeaturedDeal{
		{ID: "popular", Title: "Popular Game", Price: 40, Discount: 20},
		{ID: "obscure", Title: "Obscure Game", Price: 1, Discount: 95},
	}
	signals := map[string]SteamDealSignal{
		"popular": {PopularRank: 2, Players: 150000, ReviewPct: 92, ReviewCount: 40000},
		"obscure": {ReviewPct: 72, ReviewCount: 20},
	}

	result := RankDeals(deals, signals, 12)
	if result.Featured[0].ID != "popular" {
		t.Fatalf("expected popularity to outrank discount percentage, got %+v", result.Featured)
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

func TestSelectDealsForSignalsReservesChecksForCatalogPopularGames(t *testing.T) {
	deals := make([]models.FeaturedDeal, 0, 27)
	for index := 0; index < 24; index++ {
		deals = append(deals, models.FeaturedDeal{ID: fmt.Sprintf("steam-%d", index), Title: fmt.Sprintf("Steam deal %d", index), SteamFeatured: true})
	}
	for index := 0; index < 3; index++ {
		deals = append(deals, models.FeaturedDeal{ID: fmt.Sprintf("catalog-%d", index), Title: fmt.Sprintf("Catalog deal %d", index), ITADPopularRank: index + 1})
	}

	selected := selectDealsForSignals(deals, nil, nil, 24)
	selectedIDs := make(map[string]bool, len(selected))
	for _, deal := range selected {
		selectedIDs[deal.ID] = true
	}
	for index := 0; index < 3; index++ {
		if !selectedIDs[fmt.Sprintf("catalog-%d", index)] {
			t.Fatalf("expected popular catalog game %d to be included, got %+v", index, selected)
		}
	}
}
