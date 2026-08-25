package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
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
