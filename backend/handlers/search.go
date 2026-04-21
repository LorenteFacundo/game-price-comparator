package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		json.NewEncoder(w).Encode(models.SearchResponse{Error: "falta el parámetro q"})
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

		// 1. Filtramos Steam de ITAD (tiene precios desactualizados)
		filtered := []models.StorePrice{}
		for _, p := range main.Prices {
			if !strings.Contains(strings.ToLower(p.StoreName), "steam") {
				filtered = append(filtered, p)
			}
		}
		main.Prices = filtered

		// 2. Inyectamos Steam regional en tiempo real
		// steam.go usa country=AR → los precios ya vienen en ARS
		if steamPrice != nil && steamPrice.Found {
			steamStore := models.StorePrice{
				StoreName:  "Steam",
				PriceARS:   steamPrice.PriceARS,
				RegularARS: steamPrice.RegularARS,
				// PriceUSD queda en 0; la comparación usará PriceARS directamente
				Discount:   steamPrice.Discount,
				URL:        steamPrice.URL,
				OnSale:     steamPrice.Discount > 0,
				IsRegional: true,
			}
			main.Prices = append([]models.StorePrice{steamStore}, main.Prices...)
		}

		// 3. Tiendas sin precio en tiempo real — links directos
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

		// 4. Calculamos best deal DESPUÉS de todos los appends,
		//    usando índice en vez de puntero para evitar punteros invalidados por realloc.
		//    Ignoramos tiendas sin precio y MundoSteam.
		bestIdx := -1
		bestPesos := 0.0

		for i := range main.Prices {
			p := main.Prices[i]

			if p.StoreName == "MundoSteam" {
				continue
			}

			// Precio en pesos: usamos PriceARS directo, o convertimos PriceUSD si es necesario
			precioPesos := p.PriceARS
			if precioPesos == 0 && p.PriceUSD > 0 && usdRate > 0 {
				precioPesos = p.PriceUSD * usdRate
			}

			// Sin precio → saltar (links directos como IG/Eneba/G2A)
			if precioPesos == 0 {
				continue
			}

			if bestIdx == -1 || precioPesos < bestPesos {
				bestIdx = i
				bestPesos = precioPesos
			}
		}

		if bestIdx >= 0 {
			best := main.Prices[bestIdx]
			main.BestDeal = &best
		}

		finalResults = append(finalResults, main)
		finalResults = append(finalResults, itadResults[1:]...)
	}

	json.NewEncoder(w).Encode(models.SearchResponse{
		Query:   query,
		Results: finalResults,
		USDRate: usdRate,
	})
}
