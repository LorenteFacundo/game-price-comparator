package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
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
	// Pedimos más candidatos porque luego conservamos solamente tiendas de juegos
	// que el usuario puede abrir y comprar directamente desde Argentina.
	req, err := s.newRequest(ctx, http.MethodGet, fmt.Sprintf("%s/deals/v2?country=AR&limit=24&sort=-cut", itadBaseURL), nil)
	if err != nil {
		return nil, err
	}
	var response itadDealsResponse
	if err := s.doJSON(req, &response); err != nil {
		return nil, err
	}
	deals := make([]models.FeaturedDeal, 0, len(response.List))
	for _, item := range response.List {
		if !isFeaturedStore(item.Deal.Shop.Name) {
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
		deals = append(deals, deal)
		if len(deals) == limit {
			break
		}
	}
	return deals, nil
}

func (s *ITADService) GetRadarDeals(ctx context.Context, popular []SteamTopSeller, limit int) ([]models.FeaturedDeal, error) {
	if limit < 1 {
		limit = 12
	}
	if len(popular) == 0 {
		return []models.FeaturedDeal{}, nil
	}

	type searchResult struct {
		candidate SteamTopSeller
		game      itadSearchResult
		err       error
	}
	results := make([]searchResult, len(popular))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				games, err := s.searchGames(ctx, popular[index].Title)
				results[index] = searchResult{candidate: popular[index], game: exactGameMatch(games, popular[index].Title), err: err}
			}
		}()
	}
	for index := range popular {
		jobs <- index
	}
	close(jobs)
	wg.Wait()

	ids := make([]string, 0, len(results))
	matched := make(map[string]searchResult)
	for _, result := range results {
		if result.err == nil && result.game.ID != "" {
			ids = append(ids, result.game.ID)
			matched[result.game.ID] = result
		}
	}
	if len(ids) == 0 {
		return []models.FeaturedDeal{}, nil
	}
	prices, err := s.getPrices(ctx, ids)
	if err != nil {
		return nil, err
	}

	radar := make([]models.FeaturedDeal, 0, limit)
	for _, priceResult := range prices {
		match, ok := matched[priceResult.ID]
		if !ok {
			continue
		}
		deal, ok := bestRadarDeal(priceResult.Deals, match.candidate, match.game)
		if !ok {
			continue
		}
		radar = append(radar, deal)
	}
	sort.Slice(radar, func(i, j int) bool { return radar[i].PopularityRank < radar[j].PopularityRank })
	if len(radar) > limit {
		radar = radar[:limit]
	}
	return radar, nil
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

func exactGameMatch(games []itadSearchResult, title string) itadSearchResult {
	for _, game := range games {
		if strings.EqualFold(strings.TrimSpace(game.Title), strings.TrimSpace(title)) {
			return game
		}
	}
	return itadSearchResult{}
}

func bestRadarDeal(deals []itadDeal, candidate SteamTopSeller, game itadSearchResult) (models.FeaturedDeal, bool) {
	stores := make([]string, 0, len(deals))
	var best *itadDeal
	for index := range deals {
		deal := &deals[index]
		if !isFeaturedStore(deal.Shop.Name) || deal.Cut <= 0 {
			continue
		}
		stores = append(stores, deal.Shop.Name)
		if best == nil || deal.Cut > best.Cut || (deal.Cut == best.Cut && deal.Price.Amount < best.Price.Amount) {
			best = deal
		}
	}
	if best == nil {
		return models.FeaturedDeal{}, false
	}
	image := game.Assets.Banner300
	if image == "" {
		image = candidate.Image
	}
	if image == "" {
		image = game.Assets.BoxArt
	}
	deal := models.FeaturedDeal{ID: game.ID, Title: game.Title, ImageURL: image, StoreName: best.Shop.Name, Price: best.Price.Amount, Regular: best.Regular.Amount, Currency: strings.ToUpper(best.Price.Currency), Discount: best.Cut, URL: best.URL, PopularityRank: candidate.Rank, MatchedStores: uniqueStores(stores)}
	if best.HistoryLow != nil && strings.EqualFold(best.HistoryLow.Currency, best.Price.Currency) {
		deal.HistoryLow = best.HistoryLow.Amount
		deal.IsNearLow = best.Price.Amount <= best.HistoryLow.Amount*1.03
	}
	return deal, true
}

func uniqueStores(stores []string) []string {
	unique := make([]string, 0, len(stores))
	seen := make(map[string]struct{}, len(stores))
	for _, store := range stores {
		key := strings.ToLower(store)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			unique = append(unique, store)
		}
	}
	return unique
}
