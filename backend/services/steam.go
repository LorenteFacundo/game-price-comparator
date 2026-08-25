package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://store.steampowered.com/api/featuredcategories?cc=AR&l=spanish", nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("consultando populares de Steam: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("populares de Steam respondió con status %d", resp.StatusCode)
	}
	var payload struct {
		TopSellers struct {
			Items []struct {
				ID    int    `json:"id"`
				Type  int    `json:"type"`
				Name  string `json:"name"`
				Image string `json:"header_image"`
			} `json:"items"`
		} `json:"top_sellers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("parseando populares de Steam: %w", err)
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
		popular = append(popular, SteamTopSeller{AppID: fmt.Sprintf("%d", item.ID), Title: item.Name, Image: item.Image, Rank: len(popular) + 1})
		if len(popular) == limit {
			break
		}
	}
	return popular, nil
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
