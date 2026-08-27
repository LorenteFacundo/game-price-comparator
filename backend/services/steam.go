package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"game-price-comparator/models"
)

type SteamService struct{ client *http.Client }
type SteamPrice struct {
	Currency string
	Price    float64
	Regular  float64
	Discount int
	URL      string
	Found    bool
}

type SteamTopSeller struct {
	AppID string
	Title string
	Image string
	Rank  int
}

type steamFeaturedItem struct {
	ID                 int    `json:"id"`
	Type               int    `json:"type"`
	Name               string `json:"name"`
	HeaderImage        string `json:"header_image"`
	LargeCapsuleImage  string `json:"large_capsule_image"`
	Currency           string `json:"currency"`
	FinalPrice         int    `json:"final_price"`
	OriginalPrice      int    `json:"original_price"`
	DiscountPercent    int    `json:"discount_percent"`
	DiscountExpiration int64  `json:"discount_expiration"`
}

type steamFeaturedCategory struct {
	Items []steamFeaturedItem `json:"items"`
}

type steamFeaturedCategories struct {
	Specials   steamFeaturedCategory `json:"specials"`
	TopSellers steamFeaturedCategory `json:"top_sellers"`
}

func (s *SteamService) GetDealSignals(ctx context.Context, deals []models.FeaturedDeal, popular []SteamTopSeller, mostPlayed []models.RankedGame) map[string]SteamDealSignal {
	popularByTitle := make(map[string]SteamTopSeller, len(popular))
	playedByTitle := make(map[string]models.RankedGame, len(mostPlayed))
	for _, game := range popular {
		popularByTitle[normalizeGameTitle(game.Title)] = game
	}
	for _, game := range mostPlayed {
		playedByTitle[normalizeGameTitle(game.Title)] = game
	}

	deals = selectDealsForSignals(deals, popularByTitle, playedByTitle, 24)
	result := make(map[string]SteamDealSignal, len(deals))
	var mu sync.Mutex
	jobs := make(chan models.FeaturedDeal)
	var wg sync.WaitGroup
	for worker := 0; worker < 6; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for deal := range jobs {
				key := normalizeGameTitle(deal.Title)
				signal := SteamDealSignal{AppID: deal.SteamAppID}
				if game, ok := popularByTitle[key]; ok {
					signal.AppID, signal.PopularRank = game.AppID, game.Rank
				}
				if game, ok := playedByTitle[key]; ok {
					signal.AppID, signal.Players = game.ID, game.Players
				}
				if signal.AppID == "" {
					signal.AppID, _ = s.searchExactAppID(ctx, deal.Title)
				}
				if signal.AppID != "" {
					signal.ReviewPct, signal.ReviewCount, _ = s.getReviewSummary(ctx, signal.AppID)
					if deal.ImageURL == "" {
						if appID, err := strconv.Atoi(signal.AppID); err == nil {
							if game, ok := s.getBasicGame(ctx, appID); ok {
								signal.ImageURL = game.ImageURL
							}
						}
					}
				}
				mu.Lock()
				result[deal.ID] = signal
				mu.Unlock()
			}
		}()
	}
	for _, deal := range deals {
		jobs <- deal
	}
	close(jobs)
	wg.Wait()
	return result
}

func selectDealsForSignals(deals []models.FeaturedDeal, popular map[string]SteamTopSeller, played map[string]models.RankedGame, limit int) []models.FeaturedDeal {
	if len(deals) <= limit {
		return deals
	}
	selected := make([]models.FeaturedDeal, 0, limit)
	seen := make(map[string]struct{}, limit)
	add := func(deal models.FeaturedDeal) {
		if len(selected) >= limit {
			return
		}
		if _, ok := seen[deal.ID]; ok {
			return
		}
		selected = append(selected, deal)
		seen[deal.ID] = struct{}{}
	}
	// No dejamos que el carrusel de Steam consuma toda la muestra: los juegos
	// que ITAD marca como populares suelen aportar ofertas grandes con rebajas
	// más moderadas (por ejemplo, lanzamientos aclamados).
	steamFeaturedBudget := min(limit, 12)
	for _, deal := range deals {
		if len(selected) >= steamFeaturedBudget {
			break
		}
		if deal.SteamFeatured {
			add(deal)
		}
	}
	catalogPopularBudget := min(limit, steamFeaturedBudget+12)
	for _, deal := range deals {
		if len(selected) >= catalogPopularBudget {
			break
		}
		if deal.ITADPopularRank > 0 {
			add(deal)
		}
	}
	for _, deal := range deals {
		key := normalizeGameTitle(deal.Title)
		if _, ok := popular[key]; ok {
			add(deal)
			continue
		}
		if _, ok := played[key]; ok {
			add(deal)
		}
	}
	for _, deal := range deals {
		if deal.Price == 0 && len(selected) < 6 {
			add(deal)
		}
	}
	paid := make([]models.FeaturedDeal, 0, len(deals))
	for _, deal := range deals {
		if deal.Price > 0 {
			paid = append(paid, deal)
		}
	}
	remaining := limit - len(selected)
	if remaining > 0 && len(paid) > 0 {
		step := math.Max(1, float64(len(paid))/float64(remaining))
		for cursor := 0.0; int(cursor) < len(paid) && len(selected) < limit; cursor += step {
			add(paid[int(cursor)])
		}
	}
	for _, deal := range deals {
		add(deal)
	}
	return selected
}

type steamResponse map[string]struct {
	Success bool `json:"success"`
	Data    struct {
		PriceOverview struct {
			Currency        string `json:"currency"`
			Initial         int    `json:"initial"`
			Final           int    `json:"final"`
			DiscountPercent int    `json:"discount_percent"`
		} `json:"price_overview"`
	} `json:"data"`
}

func NewSteamService() *SteamService {
	return &SteamService{client: &http.Client{Timeout: 8 * time.Second}}
}
func (s *SteamService) GetPriceByTitle(ctx context.Context, title, country string) (*SteamPrice, error) {
	appID, err := s.searchAppID(ctx, title)
	if err != nil || appID == "" {
		return &SteamPrice{Found: false}, err
	}
	return s.GetPriceByAppID(ctx, appID, country)
}

func (s *SteamService) GetTopSellers(ctx context.Context, limit int) ([]SteamTopSeller, error) {
	if limit < 1 {
		limit = 24
	}
	payload, err := s.getFeaturedCategories(ctx)
	if err != nil {
		return nil, err
	}

	popular := make([]SteamTopSeller, 0, limit)
	seen := make(map[int]struct{})
	for _, item := range payload.TopSellers.Items {
		if item.ID == 0 || item.Name == "" || item.Type != 0 {
			continue
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		popular = append(popular, SteamTopSeller{AppID: fmt.Sprintf("%d", item.ID), Title: item.Name, Image: item.HeaderImage, Rank: len(popular) + 1})
		if len(popular) == limit {
			break
		}
	}
	return popular, nil
}

func (s *SteamService) GetSteamFeaturedDeals(ctx context.Context, limit int) ([]models.FeaturedDeal, error) {
	if limit < 1 {
		limit = 24
	}
	payload, err := s.getFeaturedCategories(ctx)
	if err != nil {
		return nil, err
	}
	deals := make([]models.FeaturedDeal, 0, limit)
	seen := make(map[int]struct{})
	appendDeals := func(items []steamFeaturedItem) {
		for _, item := range items {
			if len(deals) == limit {
				return
			}
			if _, ok := seen[item.ID]; ok {
				continue
			}
			deal, ok := steamFeaturedDeal(item)
			if !ok {
				continue
			}
			seen[item.ID] = struct{}{}
			deals = append(deals, deal)
		}
	}
	// Specials es la fuente que refleja "Descuentos y eventos" de Steam.
	appendDeals(payload.Specials.Items)
	appendDeals(payload.TopSellers.Items)
	return deals, nil
}

func (s *SteamService) getFeaturedCategories(ctx context.Context) (steamFeaturedCategories, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://store.steampowered.com/api/featuredcategories?cc=AR&l=spanish", nil)
	if err != nil {
		return steamFeaturedCategories{}, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return steamFeaturedCategories{}, fmt.Errorf("consultando destacados de Steam: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return steamFeaturedCategories{}, fmt.Errorf("destacados de Steam respondió con status %d", resp.StatusCode)
	}
	var payload steamFeaturedCategories
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return steamFeaturedCategories{}, fmt.Errorf("parseando destacados de Steam: %w", err)
	}
	return payload, nil
}

func steamFeaturedDeal(item steamFeaturedItem) (models.FeaturedDeal, bool) {
	if item.ID == 0 || item.Name == "" || item.Type != 0 || item.DiscountPercent <= 0 {
		return models.FeaturedDeal{}, false
	}
	image := item.LargeCapsuleImage
	if image == "" {
		image = item.HeaderImage
	}
	deal := models.FeaturedDeal{
		ID: "steam-featured-" + fmt.Sprintf("%d", item.ID), Title: item.Name, ImageURL: image,
		StoreName: "Steam", Price: float64(item.FinalPrice) / 100, Regular: float64(item.OriginalPrice) / 100,
		Currency: strings.ToUpper(item.Currency), Discount: item.DiscountPercent,
		URL: fmt.Sprintf("https://store.steampowered.com/app/%d/", item.ID), SteamAppID: fmt.Sprintf("%d", item.ID), SteamFeatured: true,
	}
	if item.DiscountExpiration > 0 {
		deal.ExpiresAt = time.Unix(item.DiscountExpiration, 0).UTC().Format(time.RFC3339)
	}
	return deal, true
}

func (s *SteamService) GetMostPlayed(ctx context.Context, limit int) ([]models.RankedGame, error) {
	if limit < 1 {
		limit = 8
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.steampowered.com/ISteamChartsService/GetMostPlayedGames/v1/", nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("consultando más jugados de Steam: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("más jugados de Steam respondió con status %d", resp.StatusCode)
	}
	var payload struct {
		Response struct {
			Ranks []struct {
				Rank  int `json:"rank"`
				AppID int `json:"appid"`
				Peak  int `json:"peak_in_game"`
			} `json:"ranks"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("parseando más jugados de Steam: %w", err)
	}
	if len(payload.Response.Ranks) > limit {
		payload.Response.Ranks = payload.Response.Ranks[:limit]
	}

	type resolvedGame struct {
		index int
		game  models.RankedGame
		ok    bool
	}
	resolved := make([]resolvedGame, len(payload.Response.Ranks))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				rank := payload.Response.Ranks[index]
				game, ok := s.getBasicGame(ctx, rank.AppID)
				if ok {
					game.Rank = rank.Rank
					game.Players = rank.Peak
				}
				resolved[index] = resolvedGame{index: index, game: game, ok: ok}
			}
		}()
	}
	for index := range payload.Response.Ranks {
		jobs <- index
	}
	close(jobs)
	wg.Wait()

	games := make([]models.RankedGame, 0, len(resolved))
	for _, result := range resolved {
		if result.ok {
			games = append(games, result.game)
		}
	}
	return games, nil
}

func (s *SteamService) getBasicGame(ctx context.Context, appID int) (models.RankedGame, bool) {
	appIDText := fmt.Sprintf("%d", appID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%s&country=AR&l=spanish", url.QueryEscape(appIDText)), nil)
	if err != nil {
		return models.RankedGame{}, false
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return models.RankedGame{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return models.RankedGame{}, false
	}
	var payload map[string]struct {
		Success bool `json:"success"`
		Data    struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			HeaderImage string `json:"header_image"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return models.RankedGame{}, false
	}
	entry, ok := payload[appIDText]
	if !ok || !entry.Success || entry.Data.Type != "game" || entry.Data.Name == "" {
		return models.RankedGame{}, false
	}
	return models.RankedGame{ID: appIDText, Title: entry.Data.Name, ImageURL: entry.Data.HeaderImage, SteamURL: fmt.Sprintf("https://store.steampowered.com/app/%s/", appIDText)}, true
}
func (s *SteamService) GetPriceByAppID(ctx context.Context, appID, country string) (*SteamPrice, error) {
	if country == "" {
		country = "AR"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%s&country=%s", url.QueryEscape(appID), url.QueryEscape(strings.ToUpper(country))), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return &SteamPrice{Found: false}, fmt.Errorf("consultando Steam: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &SteamPrice{Found: false}, fmt.Errorf("Steam respondió con status %d", resp.StatusCode)
	}
	var data steamResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return &SteamPrice{Found: false}, fmt.Errorf("parseando respuesta de Steam: %w", err)
	}
	entry, ok := data[appID]
	if !ok || !entry.Success || (entry.Data.PriceOverview.Final == 0 && entry.Data.PriceOverview.Initial == 0) {
		return &SteamPrice{Found: false}, nil
	}
	po := entry.Data.PriceOverview
	return &SteamPrice{Currency: strings.ToUpper(po.Currency), Price: float64(po.Final) / 100, Regular: float64(po.Initial) / 100, Discount: po.DiscountPercent, URL: fmt.Sprintf("https://store.steampowered.com/app/%s/", appID), Found: true}, nil
}
func (s *SteamService) searchAppID(ctx context.Context, title string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://store.steampowered.com/api/storesearch?term=%s&l=spanish&cc=AR", url.QueryEscape(title)), nil)
	if err != nil {
		return "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("buscando en Steam: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("búsqueda Steam respondió con status %d", resp.StatusCode)
	}
	var result struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parseando búsqueda Steam: %w", err)
	}
	if len(result.Items) == 0 {
		return "", nil
	}
	for _, item := range result.Items {
		if strings.EqualFold(item.Name, title) {
			return fmt.Sprintf("%d", item.ID), nil
		}
	}
	return fmt.Sprintf("%d", result.Items[0].ID), nil
}

func (s *SteamService) searchExactAppID(ctx context.Context, title string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://store.steampowered.com/api/storesearch?term=%s&l=spanish&cc=AR", url.QueryEscape(title)), nil)
	if err != nil {
		return "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("búsqueda Steam respondió con status %d", resp.StatusCode)
	}
	var result struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	target := canonicalDealTitle(title)
	for _, item := range result.Items {
		if item.ID != 0 && canonicalDealTitle(item.Name) == target {
			return fmt.Sprintf("%d", item.ID), nil
		}
	}
	return "", nil
}

func (s *SteamService) getReviewSummary(ctx context.Context, appID string) (int, int, error) {
	endpoint := fmt.Sprintf("https://store.steampowered.com/appreviews/%s?json=1&language=all&purchase_type=all&num_per_page=0", url.QueryEscape(appID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, 0, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("reseñas Steam respondió con status %d", resp.StatusCode)
	}
	var payload struct {
		QuerySummary struct {
			TotalPositive int `json:"total_positive"`
			TotalReviews  int `json:"total_reviews"`
		} `json:"query_summary"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, 0, err
	}
	if payload.QuerySummary.TotalReviews == 0 {
		return 0, 0, nil
	}
	pct := int(math.Round(float64(payload.QuerySummary.TotalPositive) / float64(payload.QuerySummary.TotalReviews) * 100))
	return pct, payload.QuerySummary.TotalReviews, nil
}
