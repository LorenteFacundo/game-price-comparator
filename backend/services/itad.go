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
type itadPopularGame struct {
	Position int    `json:"position"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
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
		limit = 60
	}
	if limit > 100 {
		limit = 100
	}
	// Pedimos una muestra amplia porque ITAD ordena por descuento y las primeras
	// posiciones suelen ser juegos gratuitos con 100% de descuento.
	req, err := s.newRequest(ctx, http.MethodGet, fmt.Sprintf("%s/deals/v2?country=AR&limit=200&sort=-cut", itadBaseURL), nil)
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
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
}

func (s *ITADService) GetPopularDeals(ctx context.Context, limit int) ([]models.FeaturedDeal, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	req, err := s.newRequest(ctx, http.MethodGet, fmt.Sprintf("%s/stats/most-popular/v1?limit=%d", itadBaseURL, limit), nil)
	if err != nil {
		return nil, err
	}
	var games []itadPopularGame
	if err := s.doJSON(req, &games); err != nil {
		return nil, fmt.Errorf("populares ITAD: %w", err)
	}
	ids := make([]string, 0, len(games))
	byID := make(map[string]itadPopularGame, len(games))
	for _, game := range games {
		if game.ID == "" || !strings.EqualFold(game.Type, "game") {
			continue
		}
		ids = append(ids, game.ID)
		byID[game.ID] = game
	}
	if len(ids) == 0 {
		return []models.FeaturedDeal{}, nil
	}
	prices, err := s.getPrices(ctx, ids)
	if err != nil {
		return nil, err
	}
	return popularDealsFromPrices(prices, byID), nil
}

func popularDealsFromPrices(prices []itadPriceResult, games map[string]itadPopularGame) []models.FeaturedDeal {
	deals := make([]models.FeaturedDeal, 0)
	for _, result := range prices {
		game, ok := games[result.ID]
		if !ok {
			continue
		}
		for _, source := range result.Deals {
			if !isEligibleFeaturedDeal(source) {
				continue
			}
			deal := featuredDealFromITAD(result.ID, game.Title, itadAssets{}, source)
			deal.ID = fmt.Sprintf("popular-%s-%s", result.ID, strings.ToLower(strings.ReplaceAll(source.Shop.Name, " ", "-")))
			deal.ITADPopularRank = game.Position
			deals = append(deals, deal)
		}
	}
	return deals
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

func featuredDealFromITAD(id, title string, assets itadAssets, source itadDeal) models.FeaturedDeal {
	image := assets.Banner300
	if image == "" {
		image = assets.BoxArt
	}
	deal := models.FeaturedDeal{ID: id, Title: title, ImageURL: image, StoreName: source.Shop.Name, Price: source.Price.Amount, Regular: source.Regular.Amount, Currency: strings.ToUpper(source.Price.Currency), Discount: source.Cut, URL: source.URL}
	if source.Expiry != nil {
		deal.ExpiresAt = *source.Expiry
	}
	if source.HistoryLow != nil && strings.EqualFold(source.HistoryLow.Currency, source.Price.Currency) {
		deal.HistoryLow = source.HistoryLow.Amount
		deal.IsNearLow = source.Price.Amount <= source.HistoryLow.Amount*1.03
	}
	return deal
}

func isFeaturedStore(storeName string) bool {
	_, ok := featuredStores[strings.ToLower(strings.TrimSpace(storeName))]
	return ok
}

func isEligibleFeaturedDeal(deal itadDeal) bool {
	return isFeaturedStore(deal.Shop.Name) && deal.Cut > 0
}
