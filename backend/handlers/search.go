package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	"game-price-comparator/models"
	"game-price-comparator/services"
)

type SearchHandler struct {
	itad     *services.ITADService
	currency *services.CurrencyService
	steam    *services.SteamService
	scraper  *services.ScraperService
}

func NewSearchHandler(
	itad *services.ITADService,
	currency *services.CurrencyService,
	steam *services.SteamService,
	scraper *services.ScraperService,
) *SearchHandler {
	return &SearchHandler{itad: itad, currency: currency, steam: steam, scraper: scraper}
}

func (h *SearchHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		json.NewEncoder(w).Encode(models.SearchResponse{Error: "falta el parametro q"})
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	var itadResults []models.GameResult
	var usdRate float64
	var steamPrice *services.SteamPrice

	wg.Add(3)

	go func() {
		defer wg.Done()
		results, _ := h.itad.Search(query)
		mu.Lock()
		itadResults = results
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		rate, _ := h.currency.GetBlueRate()
		mu.Lock()
		usdRate = rate
		mu.Unlock()
	}()

	go func() {
		defer wg.Done()
		p, _ := h.steam.GetPriceByTitle(query)
		mu.Lock()
		steamPrice = p
		mu.Unlock()
	}()

	wg.Wait()

	var finalResults []models.GameResult

	if len(itadResults) > 0 {
		main := itadResults[0]

		filtered := []models.StorePrice{}
		for _, p := range main.Prices {
			if !strings.Contains(strings.ToLower(p.StoreName), "steam") {
				filtered = append(filtered, p)
			}
		}
		main.Prices = filtered

		if steamPrice != nil && steamPrice.Found {
			steamStore := models.StorePrice{
				StoreName:  "Steam",
				PriceUSD:   steamPrice.PriceUSD,
				PriceARS:   steamPrice.PriceARS,
				RegularUSD: steamPrice.RegularUSD,
				RegularARS: steamPrice.RegularARS,
				Discount:   steamPrice.Discount,
				URL:        steamPrice.URL,
				OnSale:     steamPrice.Discount > 0,
				IsRegional: true,
			}
			main.Prices = append([]models.StorePrice{steamStore}, main.Prices...)
		}

		main.Prices = append(main.Prices, models.StorePrice{
			StoreName: "Instant Gaming",
			URL:       fmt.Sprintf("https://www.instant-gaming.com/es/busqueda/?q=%s", url.QueryEscape(query)),
		})

		main.Prices = append(main.Prices, models.StorePrice{
			StoreName: "Eneba",
			URL:       fmt.Sprintf("https://www.eneba.com/store/all?text=%s", url.QueryEscape(query)),
		})

		main.Prices = append(main.Prices, models.StorePrice{
			StoreName: "G2A",
			URL:       fmt.Sprintf("https://www.g2a.com/search?query=%s", url.QueryEscape(query)),
		})

		main.Prices = append(main.Prices, models.StorePrice{
			StoreName: "MundoSteam",
			URL:       fmt.Sprintf("https://mundosteam.com/buscar?q=%s", url.QueryEscape(query)),
			Warning:   "Esta tienda vende acceso a cuentas compartidas, no el juego en tu cuenta personal. No recomendamos su uso.",
		})

		main.Prices, main.BestDeal = sortPricesAndPickBest(main.Prices, usdRate)

		finalResults = append(finalResults, main)
		finalResults = append(finalResults, itadResults[1:]...)
	}

	json.NewEncoder(w).Encode(models.SearchResponse{
		Query:   query,
		Results: finalResults,
		USDRate: usdRate,
	})
}

func sortPricesAndPickBest(prices []models.StorePrice, usdRate float64) ([]models.StorePrice, *models.StorePrice) {
	sorted := append([]models.StorePrice(nil), prices...)

	sort.SliceStable(sorted, func(i, j int) bool {
		leftPrice, leftHasPrice := normalizedARS(sorted[i], usdRate)
		rightPrice, rightHasPrice := normalizedARS(sorted[j], usdRate)

		if leftHasPrice != rightHasPrice {
			return leftHasPrice
		}

		if leftHasPrice && rightHasPrice && leftPrice != rightPrice {
			return leftPrice < rightPrice
		}

		if sorted[i].StoreName == "MundoSteam" || sorted[j].StoreName == "MundoSteam" {
			return sorted[j].StoreName == "MundoSteam"
		}

		return sorted[i].StoreName < sorted[j].StoreName
	})

	for i := range sorted {
		if _, ok := normalizedARS(sorted[i], usdRate); ok && sorted[i].StoreName != "MundoSteam" {
			best := sorted[i]
			return sorted, &best
		}
	}

	return sorted, nil
}

func normalizedARS(price models.StorePrice, usdRate float64) (float64, bool) {
	if price.StoreName == "MundoSteam" {
		return 0, false
	}

	if price.PriceARS > 0 {
		return price.PriceARS, true
	}

	if price.PriceUSD > 0 && usdRate > 0 {
		return price.PriceUSD * usdRate, true
	}

	return 0, false
}
