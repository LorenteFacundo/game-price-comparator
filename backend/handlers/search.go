package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"game-price-comparator/models"
	"game-price-comparator/services"
)

const (
	cacheTTL              = 5 * time.Minute
	maxSearchCacheEntries = 500
	dealsRequestTimeout   = 15 * time.Second
	rankingPricesTimeout  = 8 * time.Second
)

type itadProvider interface {
	Search(context.Context, string) ([]models.GameResult, error)
	GetDeals(context.Context, int) ([]models.FeaturedDeal, error)
	GetPopularDeals(context.Context, int) ([]models.FeaturedDeal, error)
	PricesForSteamApps(context.Context, []string) (map[string][]models.StorePrice, error)
}
type currencyProvider interface {
	GetRates(context.Context) (services.ExchangeRates, error)
}
type steamProvider interface {
	GetPriceByTitle(context.Context, string, string) (*services.SteamPrice, error)
	GetTopSellers(context.Context, int) ([]services.SteamTopSeller, error)
	GetSteamFeaturedDeals(context.Context, int) ([]models.FeaturedDeal, error)
	GetDealSignals(context.Context, []models.FeaturedDeal, []services.SteamTopSeller, []models.RankedGame) map[string]services.SteamDealSignal
	GetMostPlayed(context.Context, int) ([]models.RankedGame, error)
}

type cachedSearch struct {
	response  models.SearchResponse
	expiresAt time.Time
}
type cachedDeals struct {
	response  models.DealsResponse
	expiresAt time.Time
}
type cachedDiscover struct {
	response  models.DiscoverResponse
	expiresAt time.Time
}
type SearchHandler struct {
	itad          itadProvider
	currency      currencyProvider
	steam         steamProvider
	taxRate       float64
	mu            sync.RWMutex
	searchCache   map[string]cachedSearch
	dealsCache    map[int]cachedDeals
	discoverCache cachedDiscover
}

func NewSearchHandler(itad itadProvider, currency currencyProvider, steam steamProvider, taxRate float64) *SearchHandler {
	if taxRate < 0 || taxRate > 1 {
		taxRate = 0.21
	}
	return &SearchHandler{itad: itad, currency: currency, steam: steam, taxRate: taxRate, searchCache: make(map[string]cachedSearch), dealsCache: make(map[int]cachedDeals)}
}

func (h *SearchHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, models.SearchResponse{Error: "método no permitido"})
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusBadRequest, models.SearchResponse{Error: "Ingresá el nombre de un juego."})
		return
	}
	if len([]rune(query)) > 100 {
		writeJSON(w, http.StatusBadRequest, models.SearchResponse{Error: "La búsqueda no puede superar 100 caracteres."})
		return
	}
	steamMode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("steam_mode")))
	if steamMode == "" {
		steamMode = "regional"
	}
	if steamMode != "regional" && steamMode != "global" {
		writeJSON(w, http.StatusBadRequest, models.SearchResponse{Error: "steam_mode inválido."})
		return
	}
	cacheKey := strings.ToLower(query) + "|" + steamMode
	if response, ok := h.cachedSearch(cacheKey); ok {
		w.Header().Set("X-Cache", "HIT")
		writeJSON(w, http.StatusOK, response)
		return
	}
	results, err := h.itad.Search(r.Context(), query)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, models.SearchResponse{Error: "No pudimos consultar las tiendas ahora. Probá de nuevo en unos segundos."})
		return
	}
	response := models.SearchResponse{Query: query, Results: results, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	comparisonRate := 0.0
	if rates, err := h.currency.GetRates(r.Context()); err == nil {
		response.USDRate = rates.Blue
		response.OfficialRate = rates.Official
		response.TaxRate = h.taxRate
		response.CardRate = rates.Official * (1 + response.TaxRate)
		if !rates.UpdatedAt.IsZero() {
			response.RatesUpdatedAt = rates.UpdatedAt.UTC().Format(time.RFC3339)
		}
		response.RatesStale = rates.Stale
		comparisonRate = response.CardRate
		if rates.Stale {
			response.Warnings = append(response.Warnings, "La cotización está temporalmente desactualizada; verificá el total antes de comprar.")
		}
	} else {
		response.Warnings = append(response.Warnings, "No se pudo actualizar la cotización; mostramos el precio base.")
	}
	if steamMode == "global" {
		response.Warnings = append(response.Warnings, h.replaceWithGlobalSteam(r.Context(), response.Results)...)
	}
	for index := range response.Results {
		response.Results[index].Prices, response.Results[index].BestDeal = sortPricesAndPickBest(response.Results[index].Prices, comparisonRate)
	}
	h.storeSearch(cacheKey, response)
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, response)
}

func (h *SearchHandler) replaceWithGlobalSteam(ctx context.Context, results []models.GameResult) []string {
	type lookup struct {
		index int
		price *services.SteamPrice
		err   error
	}
	lookups := make([]lookup, len(results))
	jobs := make(chan int)
	var wg sync.WaitGroup
	for worker := 0; worker < min(4, len(results)); worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				price, err := h.steam.GetPriceByTitle(ctx, results[index].Title, "US")
				lookups[index] = lookup{index: index, price: price, err: err}
			}
		}()
	}
	for index := range results {
		jobs <- index
	}
	close(jobs)
	wg.Wait()

	failed := false
	for _, item := range lookups {
		if item.err != nil || item.price == nil || !item.price.Found {
			failed = true
			continue
		}
		results[item.index].Prices = withoutStore(results[item.index].Prices, "Steam")
		results[item.index].Prices = append(results[item.index].Prices, models.StorePrice{StoreName: "Steam", Price: item.price.Price, Regular: item.price.Regular, Currency: item.price.Currency, Discount: item.price.Discount, URL: item.price.URL, OnSale: item.price.Discount > 0})
	}
	if failed {
		return []string{"No pudimos cargar Steam global para algunos resultados."}
	}
	return nil
}

func withoutStore(prices []models.StorePrice, storeName string) []models.StorePrice {
	filtered := make([]models.StorePrice, 0, len(prices))
	for _, price := range prices {
		if !strings.EqualFold(price.StoreName, storeName) {
			filtered = append(filtered, price)
		}
	}
	return filtered
}
func (h *SearchHandler) HandleDeals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, models.DealsResponse{Error: "método no permitido"})
		return
	}
	limit := 12
	if value := r.URL.Query().Get("limit"); value != "" {
		if _, err := fmt.Sscanf(value, "%d", &limit); err != nil || limit < 1 || limit > 24 {
			writeJSON(w, http.StatusBadRequest, models.DealsResponse{Error: "limit debe estar entre 1 y 24."})
			return
		}
	}
	if response, ok := h.cachedDeals(limit); ok {
		w.Header().Set("X-Cache", "HIT")
		writeJSON(w, http.StatusOK, response)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dealsRequestTimeout)
	defer cancel()
	type dealsResult struct {
		deals []models.FeaturedDeal
		err   error
	}
	type popularResult struct {
		games []services.SteamTopSeller
		err   error
	}
	type highlightedResult struct {
		deals []models.FeaturedDeal
		err   error
	}
	type catalogPopularResult struct {
		deals []models.FeaturedDeal
		err   error
	}
	type ratesResult struct {
		rates services.ExchangeRates
		err   error
	}
	dealsChannel := make(chan dealsResult, 1)
	popularChannel := make(chan popularResult, 1)
	highlightedChannel := make(chan highlightedResult, 1)
	catalogPopularChannel := make(chan catalogPopularResult, 1)
	ratesChannel := make(chan ratesResult, 1)
	go func() {
		deals, err := h.itad.GetDeals(ctx, 80)
		dealsChannel <- dealsResult{deals: deals, err: err}
	}()
	go func() {
		games, err := h.steam.GetTopSellers(ctx, 30)
		popularChannel <- popularResult{games: games, err: err}
	}()
	go func() {
		deals, err := h.steam.GetSteamFeaturedDeals(ctx, 24)
		highlightedChannel <- highlightedResult{deals: deals, err: err}
	}()
	go func() {
		deals, err := h.itad.GetPopularDeals(ctx, 50)
		catalogPopularChannel <- catalogPopularResult{deals: deals, err: err}
	}()
	go func() { rates, err := h.currency.GetRates(ctx); ratesChannel <- ratesResult{rates: rates, err: err} }()
	loadedDeals := <-dealsChannel
	loadedPopular := <-popularChannel
	loadedHighlighted := <-highlightedChannel
	loadedCatalogPopular := <-catalogPopularChannel
	loadedRates := <-ratesChannel
	deals := append([]models.FeaturedDeal(nil), loadedDeals.deals...)
	popular, popularErr := loadedPopular.games, loadedPopular.err
	if loadedHighlighted.err == nil {
		deals = append(deals, loadedHighlighted.deals...)
	}
	if loadedCatalogPopular.err == nil {
		deals = append(deals, loadedCatalogPopular.deals...)
	}
	mostPlayed := []models.RankedGame{}
	if discover, ok := h.cachedDiscoverResponse(); ok {
		mostPlayed = discover.MostPlayed
	}
	if len(deals) == 0 {
		writeJSON(w, http.StatusBadGateway, models.DealsResponse{Error: "No pudimos cargar las ofertas ahora."})
		return
	}
	signals := h.steam.GetDealSignals(ctx, deals, popular, mostPlayed)
	response := services.RankDeals(deals, signals, limit*3)
	response.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	response.TaxRate = h.taxRate
	if loadedRates.err == nil {
		response.OfficialRate = loadedRates.rates.Official
		response.CardRate = loadedRates.rates.Official * (1 + h.taxRate)
		if !loadedRates.rates.UpdatedAt.IsZero() {
			response.RatesUpdatedAt = loadedRates.rates.UpdatedAt.UTC().Format(time.RFC3339)
		}
		response.RatesStale = loadedRates.rates.Stale
		if loadedRates.rates.Stale {
			response.Warnings = append(response.Warnings, "La cotización está temporalmente desactualizada; verificá el total antes de comprar.")
		}
	} else {
		response.Warnings = append(response.Warnings, "No pudimos actualizar la cotización; la conversión a ARS no está disponible.")
	}
	if loadedDeals.err != nil {
		response.Warnings = append(response.Warnings, "No pudimos sumar las ofertas generales del catálogo.")
	}
	if popularErr != nil {
		response.Warnings = append(response.Warnings, "No pudimos sumar el ranking de ventas de Steam.")
	}
	if loadedHighlighted.err != nil {
		response.Warnings = append(response.Warnings, "No pudimos sumar los destacados de Steam.")
	}
	if loadedCatalogPopular.err != nil {
		response.Warnings = append(response.Warnings, "No pudimos sumar el catálogo popular.")
	}
	h.storeDeals(limit, response)
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, response)
}

func (h *SearchHandler) HandleDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, models.DiscoverResponse{Error: "método no permitido"})
		return
	}
	if response, ok := h.cachedDiscoverResponse(); ok {
		w.Header().Set("X-Cache", "HIT")
		writeJSON(w, http.StatusOK, response)
		return
	}

	type result struct {
		popular    []services.SteamTopSeller
		mostPlayed []models.RankedGame
		err        error
	}
	popularResult := make(chan result, 1)
	mostPlayedResult := make(chan result, 1)
	go func() {
		games, err := h.steam.GetTopSellers(r.Context(), 8)
		popularResult <- result{popular: games, err: err}
	}()
	go func() {
		games, err := h.steam.GetMostPlayed(r.Context(), 8)
		mostPlayedResult <- result{mostPlayed: games, err: err}
	}()

	popular := <-popularResult
	mostPlayed := <-mostPlayedResult
	response := models.DiscoverResponse{Popular: rankedTopSellers(popular.popular), MostPlayed: mostPlayed.mostPlayed, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	if popular.err != nil {
		response.Warnings = append(response.Warnings, "No pudimos actualizar los populares de Steam.")
	}
	if mostPlayed.err != nil {
		response.Warnings = append(response.Warnings, "No pudimos actualizar los más jugados de Steam.")
	}
	if len(response.Popular) == 0 && len(response.MostPlayed) == 0 {
		writeJSON(w, http.StatusBadGateway, models.DiscoverResponse{Error: "No pudimos cargar los rankings de Steam ahora."})
		return
	}
	h.attachRankingPrices(r.Context(), &response)
	h.storeDiscover(response)
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, response)
}

// attachRankingPrices suma a cada juego de los rankings su precio en Steam y
// el más barato entre todas las tiendas. Si falla, los rankings se muestran
// igual, sin precios, y se avisa.
func (h *SearchHandler) attachRankingPrices(ctx context.Context, response *models.DiscoverResponse) {
	ctx, cancel := context.WithTimeout(ctx, rankingPricesTimeout)
	defer cancel()
	seen := make(map[string]struct{})
	appIDs := make([]string, 0, len(response.Popular)+len(response.MostPlayed))
	for _, games := range [][]models.RankedGame{response.Popular, response.MostPlayed} {
		for _, game := range games {
			if _, ok := seen[game.ID]; ok || game.ID == "" {
				continue
			}
			seen[game.ID] = struct{}{}
			appIDs = append(appIDs, game.ID)
		}
	}
	type pricesResult struct {
		prices map[string][]models.StorePrice
		err    error
	}
	pricesChannel := make(chan pricesResult, 1)
	go func() {
		prices, err := h.itad.PricesForSteamApps(ctx, appIDs)
		pricesChannel <- pricesResult{prices: prices, err: err}
	}()
	comparisonRate := 0.0
	if rates, err := h.currency.GetRates(ctx); err == nil {
		response.OfficialRate = rates.Official
		response.TaxRate = h.taxRate
		response.CardRate = rates.Official * (1 + h.taxRate)
		comparisonRate = response.CardRate
	}
	loaded := <-pricesChannel
	if loaded.err != nil {
		response.Warnings = append(response.Warnings, "No pudimos cargar los precios de los rankings.")
		return
	}
	for _, games := range [][]models.RankedGame{response.Popular, response.MostPlayed} {
		for index := range games {
			prices := loaded.prices[games[index].ID]
			if len(prices) == 0 {
				continue
			}
			sorted, best := sortPricesAndPickBest(prices, comparisonRate)
			games[index].BestPrice = best
			for _, price := range sorted {
				if strings.EqualFold(price.StoreName, "Steam") {
					steam := price
					games[index].SteamPrice = &steam
					break
				}
			}
		}
	}
}

func rankedTopSellers(games []services.SteamTopSeller) []models.RankedGame {
	ranked := make([]models.RankedGame, 0, len(games))
	for _, game := range games {
		ranked = append(ranked, models.RankedGame{ID: game.AppID, Title: game.Title, ImageURL: game.Image, SteamURL: fmt.Sprintf("https://store.steampowered.com/app/%s/", game.AppID), Rank: game.Rank})
	}
	return ranked
}
func (h *SearchHandler) cachedSearch(key string) (models.SearchResponse, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	entry, ok := h.searchCache[key]
	if ok && !time.Now().Before(entry.expiresAt) {
		delete(h.searchCache, key)
		return models.SearchResponse{}, false
	}
	return entry.response, ok
}
func (h *SearchHandler) storeSearch(key string, response models.SearchResponse) {
	h.mu.Lock()
	now := time.Now()
	for cacheKey, entry := range h.searchCache {
		if !now.Before(entry.expiresAt) {
			delete(h.searchCache, cacheKey)
		}
	}
	if _, exists := h.searchCache[key]; !exists && len(h.searchCache) >= maxSearchCacheEntries {
		var oldestKey string
		var oldestExpiry time.Time
		for cacheKey, entry := range h.searchCache {
			if oldestExpiry.IsZero() || entry.expiresAt.Before(oldestExpiry) {
				oldestKey, oldestExpiry = cacheKey, entry.expiresAt
			}
		}
		delete(h.searchCache, oldestKey)
	}
	h.searchCache[key] = cachedSearch{response: response, expiresAt: now.Add(cacheTTL)}
	h.mu.Unlock()
}
func (h *SearchHandler) cachedDeals(limit int) (models.DealsResponse, bool) {
	h.mu.RLock()
	entry, ok := h.dealsCache[limit]
	h.mu.RUnlock()
	return entry.response, ok && time.Now().Before(entry.expiresAt)
}
func (h *SearchHandler) storeDeals(limit int, response models.DealsResponse) {
	h.mu.Lock()
	h.dealsCache[limit] = cachedDeals{response: response, expiresAt: time.Now().Add(cacheTTL)}
	h.mu.Unlock()
}
func (h *SearchHandler) cachedDiscoverResponse() (models.DiscoverResponse, bool) {
	h.mu.RLock()
	entry := h.discoverCache
	h.mu.RUnlock()
	return entry.response, !entry.expiresAt.IsZero() && time.Now().Before(entry.expiresAt)
}
func (h *SearchHandler) storeDiscover(response models.DiscoverResponse) {
	h.mu.Lock()
	h.discoverCache = cachedDiscover{response: response, expiresAt: time.Now().Add(cacheTTL)}
	h.mu.Unlock()
}
func sortPricesAndPickBest(prices []models.StorePrice, usdRate float64) ([]models.StorePrice, *models.StorePrice) {
	sorted := append([]models.StorePrice(nil), prices...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left, leftOK := normalizedARS(sorted[i], usdRate)
		right, rightOK := normalizedARS(sorted[j], usdRate)
		if leftOK != rightOK {
			return leftOK
		}
		if leftOK && left != right {
			return left < right
		}
		return strings.ToLower(sorted[i].StoreName) < strings.ToLower(sorted[j].StoreName)
	})
	for i := range sorted {
		if _, ok := normalizedARS(sorted[i], usdRate); ok {
			best := sorted[i]
			return sorted, &best
		}
	}
	return sorted, nil
}
func normalizedARS(price models.StorePrice, usdRate float64) (float64, bool) {
	switch strings.ToUpper(price.Currency) {
	case "ARS":
		return price.Price, price.Price >= 0
	case "USD":
		if price.Price == 0 {
			return 0, true
		}
		return price.Price * usdRate, price.Price > 0 && usdRate > 0
	default:
		return 0, false
	}
}
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
