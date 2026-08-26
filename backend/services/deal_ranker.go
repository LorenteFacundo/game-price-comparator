package services

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"game-price-comparator/models"
)

type SteamDealSignal struct {
	AppID       string
	ReviewPct   int
	ReviewCount int
	PopularRank int
	Players     int
}

var secondaryMarkers = []string{" soundtrack", " dlc", " level", " levels", " season pass", " upgrade", " theme", " bundle", " expansion", " cosmetic pack"}
var editionMarkers = []string{" deluxe edition", " ultimate edition", " complete edition", " gold edition", " collector's edition", " game of the year edition"}

func RankDeals(candidates []models.FeaturedDeal, signals map[string]SteamDealSignal, limit int) models.DealsResponse {
	if limit < 1 {
		limit = 12
	}
	ranked := append([]models.FeaturedDeal(nil), candidates...)
	for i := range ranked {
		signal := signals[ranked[i].ID]
		applyDealScore(&ranked[i], signal)
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Score > ranked[j].Score })

	free := make([]models.FeaturedDeal, 0)
	paid := make([]models.FeaturedDeal, 0)
	for _, deal := range ranked {
		if deal.Price == 0 {
			free = append(free, deal)
		} else {
			paid = append(paid, deal)
		}
	}
	featured := diversifyDeals(paid, limit)
	free = diversifyDeals(free, min(limit, 8))

	discounts := append([]models.FeaturedDeal(nil), paid...)
	sort.SliceStable(discounts, func(i, j int) bool {
		if discounts[i].Discount != discounts[j].Discount {
			return discounts[i].Discount > discounts[j].Discount
		}
		return discounts[i].Score > discounts[j].Score
	})
	discounts = diversifyDeals(discounts, limit)
	return models.DealsResponse{Featured: featured, Free: free, Discounts: discounts}
}

func applyDealScore(deal *models.FeaturedDeal, signal SteamDealSignal) {
	deal.SteamAppID = signal.AppID
	deal.ReviewPct = signal.ReviewPct
	deal.ReviewCount = signal.ReviewCount
	deal.PopularRank = signal.PopularRank
	deal.Players = signal.Players

	// Ya todas las candidatas están en oferta. El porcentaje exacto no define
	// si vale la pena mirarla: sólo aporta un bonus pequeño por estar rebajada.
	discount := 0.0
	if deal.Discount > 0 {
		discount = 5
	}
	history := 0.0
	if deal.IsNearLow {
		history = 8
	} else if deal.HistoryLow > 0 && deal.Price <= deal.HistoryLow*1.15 {
		history = 4
	}
	popularity := 0.0
	if signal.PopularRank > 0 {
		// Ventas recientes: 25% más peso que el modelo anterior (25 → 31,25).
		popularity = 31.25 * math.Max(0, 1-float64(signal.PopularRank-1)/30)
	}
	players := 0.0
	if signal.Players > 0 {
		players = 25 * math.Min(1, math.Log10(float64(signal.Players)+1)/6)
	}
	reviews := 0.0
	if signal.ReviewCount > 0 {
		adjusted := (float64(signal.ReviewPct)*float64(signal.ReviewCount) + 75*100) / float64(signal.ReviewCount+100)
		volume := math.Min(1, math.Log10(float64(signal.ReviewCount)+1)/5)
		reviews = 25 * (.75*adjusted/100 + .25*volume)
	}
	freshness := 0.5
	if deal.ExpiresAt != "" {
		if expiry, err := time.Parse(time.RFC3339, deal.ExpiresAt); err == nil {
			remaining := time.Until(expiry)
			if remaining > 0 && remaining <= 7*24*time.Hour {
				freshness = 3
			} else if remaining > 0 && remaining <= 30*24*time.Hour {
				freshness = 1.5
			}
		}
	}
	penalty := 0.0
	if isSecondaryContent(deal.Title) {
		penalty = 32
	}
	deal.Score = math.Round(math.Max(0, math.Min(100, discount+history+popularity+players+reviews+freshness-penalty))*10) / 10

	reasons := make([]string, 0, 2)
	if signal.PopularRank > 0 {
		reasons = append(reasons, fmt.Sprintf("#%d en ventas", signal.PopularRank))
	}
	if signal.Players > 0 && len(reasons) < 2 {
		reasons = append(reasons, compactPlayers(signal.Players)+" jugando")
	}
	if signal.ReviewCount >= 50 && len(reasons) < 2 {
		reasons = append(reasons, fmt.Sprintf("%d%% positivas", signal.ReviewPct))
	}
	if deal.IsNearLow && len(reasons) < 2 {
		reasons = append(reasons, "Mínimo histórico")
	}
	if deal.Price == 0 && len(reasons) < 2 {
		reasons = append(reasons, "Gratis ahora")
	}
	deal.Reasons = reasons
}

func diversifyDeals(deals []models.FeaturedDeal, limit int) []models.FeaturedDeal {
	selected := make([]models.FeaturedDeal, 0, limit)
	seen := make(map[string]int)
	for _, deal := range deals {
		key := canonicalDealTitle(deal.Title)
		if seen[key] >= 1 {
			continue
		}
		selected = append(selected, deal)
		seen[key]++
		if len(selected) == limit {
			break
		}
	}
	return selected
}

func canonicalDealTitle(title string) string {
	normal := normalizeGameTitle(title)
	for _, marker := range editionMarkers {
		normal = strings.TrimSuffix(normal, marker)
	}
	return strings.TrimSpace(normal)
}

func isSecondaryContent(title string) bool {
	normal := " " + normalizeGameTitle(title)
	for _, marker := range secondaryMarkers {
		if strings.Contains(normal, marker) {
			return true
		}
	}
	return false
}

func normalizeGameTitle(value string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			space = false
		} else if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}

func compactPlayers(players int) string {
	if players >= 1_000_000 {
		return fmt.Sprintf("%.1f M", float64(players)/1_000_000)
	}
	if players >= 1_000 {
		return fmt.Sprintf("%.0f mil", float64(players)/1_000)
	}
	return fmt.Sprintf("%d", players)
}
