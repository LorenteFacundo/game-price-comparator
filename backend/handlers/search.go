package handlers

import (
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

const cacheTTL = 5 * time.Minute

type cachedSearch struct {
	response  models.SearchResponse
	expiresAt time.Time
}
type cachedDeals struct {
	response  models.DealsResponse
	expiresAt time.Time
}
type SearchHandler struct {
	itad        *services.ITADService
	currency    *services.CurrencyService
	steam       *services.SteamService
	mu          sync.RWMutex
	searchCache map[string]cachedSearch
	dealsCache  map[int]cachedDeals
}

func NewSearchHandler(itad *services.ITADService, currency *services.CurrencyService, steam *services.SteamService) *SearchHandler {
	return &SearchHandler{itad: itad, currency: currency, steam: steam, searchCache: make(map[string]cachedSearch), dealsCache: make(map[int]cachedDeals)}
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
	response := models.SearchResponse{Query: query, Results: results}
	if rate, err := h.currency.GetBlueRate(r.Context()); err == nil {
		response.USDRate = rate
	} else {
		response.Warnings = append(response.Warnings, "No se pudo actualizar la conversión USD/ARS; mostramos la moneda original.")
	}
	for index := range response.Results {
		// ITAD ya consulta Argentina y entrega la cotización regional de Steam en ARS.
		// Sólo consultamos Steam de forma directa cuando se pidió explícitamente el precio global.
		if index == 0 && steamMode == "global" {
			response.Results[index].Prices = withoutStore(response.Results[index].Prices, "Steam")
			steamPrice, steamErr := h.steam.GetPriceByTitle(r.Context(), response.Results[index].Title, "US")
			if steamErr != nil {
				response.Warnings = append(response.Warnings, "No pudimos cargar Steam global.")
			}
			if steamPrice != nil && steamPrice.Found {
				response.Results[index].Prices = append(response.Results[index].Prices, models.StorePrice{StoreName: "Steam", Price: steamPrice.Price, Regular: steamPrice.Regular, Currency: steamPrice.Currency, Discount: steamPrice.Discount, URL: steamPrice.URL, OnSale: steamPrice.Discount > 0})
			}
		}
		response.Results[index].Prices, response.Results[index].BestDeal = sortPricesAndPickBest(response.Results[index].Prices, response.USDRate)
	}
	h.storeSearch(cacheKey, response)
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, response)
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
	popular, popularErr := h.steam.GetTopSellers(r.Context(), 12)
	deals := []models.FeaturedDeal{}
	var err error
	if popularErr == nil {
		deals, err = h.itad.GetRadarDeals(r.Context(), popular, limit)
	}
	response := models.DealsResponse{Deals: deals}
	if popularErr != nil || err != nil || len(deals) == 0 {
		deals, err = h.itad.GetDeals(r.Context(), limit)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, models.DealsResponse{Error: "No pudimos cargar las ofertas ahora."})
			return
		}
		response.Deals = deals
		response.Warnings = append(response.Warnings, "Mostramos ofertas destacadas mientras se actualiza el radar de populares.")
	}
	h.storeDeals(limit, response)
	w.Header().Set("Cache-Control", "public, max-age=60")
	writeJSON(w, http.StatusOK, response)
}
func (h *SearchHandler) cachedSearch(key string) (models.SearchResponse, bool) {
	h.mu.RLock()
	entry, ok := h.searchCache[key]
	h.mu.RUnlock()
	return entry.response, ok && time.Now().Before(entry.expiresAt)
}
func (h *SearchHandler) storeSearch(key string, response models.SearchResponse) {
	h.mu.Lock()
	h.searchCache[key] = cachedSearch{response: response, expiresAt: time.Now().Add(cacheTTL)}
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
		return price.Price, price.Price > 0
	case "USD":
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
