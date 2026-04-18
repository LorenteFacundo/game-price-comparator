package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"game-price-comparator/models"
)

type ITADService struct {
	apiKey string
	client *http.Client
}

type itadSearchResult struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Assets struct {
		BoxArt    string `json:"boxart"`
		Banner300 string `json:"banner300"`
	} `json:"assets"`
}

type itadPriceResult struct {
	ID    string `json:"id"`
	Deals []struct {
		Shop struct {
			Name string `json:"name"`
		} `json:"shop"`
		Price struct {
			Amount float64 `json:"amount"`
		} `json:"price"`
		Regular struct {
			Amount float64 `json:"amount"`
		} `json:"regular"`
		Cut int    `json:"cut"`
		URL string `json:"url"`
	} `json:"deals"`
}

func NewITADService(apiKey string) *ITADService {
	return &ITADService{
		apiKey: apiKey,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *ITADService) Search(query string) ([]models.GameResult, error) {
	games, err := s.searchGames(query)
	if err != nil {
		return nil, err
	}
	if len(games) == 0 {
		return []models.GameResult{}, nil
	}

	// Tomamos los primeros 5 resultados
	if len(games) > 5 {
		games = games[:5]
	}

	ids := make([]string, len(games))
	for i, g := range games {
		ids[i] = g.ID
	}

	prices, err := s.getPrices(ids)
	if err != nil {
		return nil, err
	}

	// Mapeamos precios por ID de juego
	priceMap := make(map[string][]models.StorePrice)
	for _, p := range prices {
		var storePrices []models.StorePrice
		for _, deal := range p.Deals {
			sp := models.StorePrice{
				StoreName:  deal.Shop.Name,
				PriceARS:   deal.Price.Amount,
				RegularARS: deal.Regular.Amount,
				Discount:   deal.Cut,
				URL:        deal.URL,
				OnSale:     deal.Cut > 0,
			}
			storePrices = append(storePrices, sp)
		}
		// Ordenamos de menor a mayor precio
		sort.Slice(storePrices, func(i, j int) bool {
			return storePrices[i].PriceARS < storePrices[j].PriceARS
		})
		priceMap[p.ID] = storePrices
	}

	var results []models.GameResult
	for _, g := range games {
		image := g.Assets.Banner300
		if image == "" {
			image = g.Assets.BoxArt
		}

		storePrices := priceMap[g.ID]
		result := models.GameResult{
			ID:       g.ID,
			Title:    g.Title,
			ImageURL: image,
			Prices:   storePrices,
		}
		if len(storePrices) > 0 {
			best := storePrices[0]
			result.BestDeal = &best
		}
		results = append(results, result)
	}

	return results, nil
}

func (s *ITADService) searchGames(query string) ([]itadSearchResult, error) {
	endpoint := fmt.Sprintf(
		"https://api.isthereanydeal.com/games/search/v1?title=%s&results=5&key=%s",
		url.QueryEscape(query), s.apiKey,
	)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en búsqueda: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ITAD respondió con status %d", resp.StatusCode)
	}

	var results []itadSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("error parseando búsqueda: %w", err)
	}
	return results, nil
}

func (s *ITADService) getPrices(ids []string) ([]itadPriceResult, error) {
	body, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api.isthereanydeal.com/games/prices/v3?country=AR&key=%s", s.apiKey),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo precios: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ITAD precios respondió con status %d", resp.StatusCode)
	}

	var results []itadPriceResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("error parseando precios: %w", err)
	}
	return results, nil
}
