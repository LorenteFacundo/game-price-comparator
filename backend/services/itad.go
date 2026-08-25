package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"game-price-comparator/models"
)

const itadBaseURL = "https://api.isthereanydeal.com"

var featuredStores = map[string]struct{}{
	"steam":            {},
	"epic games store": {},
	"epic game store":  {},
	"microsoft store":  {},
}

type ITADService struct {
	apiKey string
	client *http.Client
}
type itadMoney struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}
type itadAssets struct {
	BoxArt    string `json:"boxart"`
	Banner300 string `json:"banner300"`
}
type itadSearchResult struct {
	ID     string     `json:"id"`
	Title  string     `json:"title"`
	Assets itadAssets `json:"assets"`
}
type itadDeal struct {
	Shop struct {
		Name string `json:"name"`
	} `json:"shop"`
	Price      itadMoney  `json:"price"`
	Regular    itadMoney  `json:"regular"`
	Cut        int        `json:"cut"`
	URL        string     `json:"url"`
	HistoryLow *itadMoney `json:"historyLow"`
	Expiry     *string    `json:"expiry"`
}
type itadPriceResult struct {
	ID    string     `json:"id"`
	Deals []itadDeal `json:"deals"`
}
type itadDealsResponse struct {
	List []struct {
		ID     string     `json:"id"`
		Title  string     `json:"title"`
		Assets itadAssets `json:"assets"`
		Deal   itadDeal   `json:"deal"`
	} `json:"list"`
}

func NewITADService(apiKey string) *ITADService {
	return &ITADService{apiKey: apiKey, client: &http.Client{Timeout: 12 * time.Second}}
}

func (s *ITADService) Search(ctx context.Context, query string) ([]models.GameResult, error) {
	games, err := s.searchGames(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(games) == 0 {
		return []models.GameResult{}, nil
	}
	ids := make([]string, len(games))
	for i, game := range games {
		ids[i] = game.ID
	}
	prices, err := s.getPrices(ctx, ids)
	if err != nil {
		return nil, err
	}
	priceMap := make(map[string][]models.StorePrice, len(prices))
	for _, result := range prices {
		storePrices := make([]models.StorePrice, 0, len(result.Deals))
		for _, deal := range result.Deals {
			storePrices = append(storePrices, storePriceFromDeal(deal))
		}
		priceMap[result.ID] = storePrices
	}
	results := make([]models.GameResult, 0, len(games))
	for _, game := range games {
		image := game.Assets.Banner300
		if image == "" {
			image = game.Assets.BoxArt
		}
		results = append(results, models.GameResult{ID: game.ID, Title: game.Title, ImageURL: image, Prices: priceMap[game.ID]})
	}
	return results, nil
}

func (s *ITADService) GetDeals(ctx context.Context, limit int) ([]models.FeaturedDeal, error) {
	if limit < 1 {
		limit = 12
	}
	if limit > 24 {
		limit = 24
	}
	// Pedimos una muestra amplia porque ITAD ordena por descuento y las primeras
	// posiciones suelen ser juegos gratuitos con 100% de descuento.
	req, err := s.newRequest(ctx, http.MethodGet, fmt.Sprintf("%s/deals/v2?country=AR&limit=100&sort=-cut", itadBaseURL), nil)
	if err != nil {
		return nil, err
	}
	var response itadDealsResponse
	if err := s.doJSON(req, &response); err != nil {
		return nil, err
	}
	candidates := make([]models.FeaturedDeal, 0, len(response.List))
	for _, item := range response.List {
		if !isEligibleFeaturedDeal(item.Deal) {
			continue
		}
		image := item.Assets.Banner300
		if image == "" {
			image = item.Assets.BoxArt
		}
		deal := models.FeaturedDeal{ID: item.ID, Title: item.Title, ImageURL: image, StoreName: item.Deal.Shop.Name, Price: item.Deal.Price.Amount, Regular: item.Deal.Regular.Amount, Currency: strings.ToUpper(item.Deal.Price.Currency), Discount: item.Deal.Cut, URL: item.Deal.URL}
		if item.Deal.Expiry != nil {
			deal.ExpiresAt = *item.Deal.Expiry
		}
		if item.Deal.HistoryLow != nil && strings.EqualFold(item.Deal.HistoryLow.Currency, item.Deal.Price.Currency) {
			deal.HistoryLow = item.Deal.HistoryLow.Amount
			deal.IsNearLow = item.Deal.Price.Amount <= item.Deal.HistoryLow.Amount*1.03
		}
		candidates = append(candidates, deal)
	}
	return selectFeaturedDeals(candidates, limit), nil
}

func (s *ITADService) searchGames(ctx context.Context, query string) ([]itadSearchResult, error) {
	req, err := s.newRequest(ctx, http.MethodGet, fmt.Sprintf("%s/games/search/v1?title=%s&results=5", itadBaseURL, url.QueryEscape(query)), nil)
	if err != nil {
		return nil, err
	}
	var results []itadSearchResult
	if err := s.doJSON(req, &results); err != nil {
		return nil, fmt.Errorf("búsqueda ITAD: %w", err)
	}
	return results, nil
}
func (s *ITADService) getPrices(ctx context.Context, ids []string) ([]itadPriceResult, error) {
	body, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	req, err := s.newRequest(ctx, http.MethodPost, itadBaseURL+"/games/prices/v3?country=AR", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	var results []itadPriceResult
	if err := s.doJSON(req, &results); err != nil {
		return nil, fmt.Errorf("precios ITAD: %w", err)
	}
	return results, nil
}
func (s *ITADService) newRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("ITAD-API-Key", s.apiKey)
	return req, nil
}
func (s *ITADService) doJSON(req *http.Request, target any) error {
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("respondió con status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
func storePriceFromDeal(deal itadDeal) models.StorePrice {
	currency := strings.ToUpper(deal.Price.Currency)
	return models.StorePrice{StoreName: deal.Shop.Name, Price: deal.Price.Amount, Regular: deal.Regular.Amount, Currency: currency, Discount: deal.Cut, URL: deal.URL, OnSale: deal.Cut > 0, IsRegional: strings.EqualFold(deal.Shop.Name, "Steam") && currency == "ARS"}
}

func isFeaturedStore(storeName string) bool {
	_, ok := featuredStores[strings.ToLower(strings.TrimSpace(storeName))]
	return ok
}

func isEligibleFeaturedDeal(deal itadDeal) bool {
	return isFeaturedStore(deal.Shop.Name) && deal.Cut > 0
}

func selectFeaturedDeals(candidates []models.FeaturedDeal, limit int) []models.FeaturedDeal {
	if limit < 1 {
		return []models.FeaturedDeal{}
	}
	free := make([]models.FeaturedDeal, 0, len(candidates))
	paid := make([]models.FeaturedDeal, 0, len(candidates))
	for _, deal := range candidates {
		if deal.Price == 0 {
			free = append(free, deal)
		} else {
			paid = append(paid, deal)
		}
	}

	selected := make([]models.FeaturedDeal, 0, limit)
	for len(selected) < limit && (len(free) > 0 || len(paid) > 0) {
		// Intercalamos un juego gratis y hasta dos ofertas pagas para que
		// convivan los regalos con descuentos del 10%, 30%, 50%, etc.
		if len(free) > 0 {
			selected = append(selected, free[0])
			free = free[1:]
		}
		for index := 0; index < 2 && len(selected) < limit && len(paid) > 0; index++ {
			selected = append(selected, paid[0])
			paid = paid[1:]
		}
		if len(free) == 0 {
			selected = append(selected, paid...)
			if len(selected) > limit {
				selected = selected[:limit]
			}
			break
		}
		if len(paid) == 0 {
			selected = append(selected, free...)
			if len(selected) > limit {
				selected = selected[:limit]
			}
			break
		}
	}
	return selected
}
