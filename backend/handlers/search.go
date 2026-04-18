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

		// 1. Filtramos Steam de ITAD (que tiene precios viejos)
		filtered := []models.StorePrice{}
		for _, p := range main.Prices {
			if !strings.Contains(strings.ToLower(p.StoreName), "steam") {
				filtered = append(filtered, p)
			}
		}
		main.Prices = filtered

		// 2. Inyectamos Steam regional en tiempo real
		if steamPrice != nil && steamPrice.Found {
			steamStore := models.StorePrice{
				StoreName:  "Steam",
				PriceUSD:   steamPrice.PriceUSD,
				RegularUSD: steamPrice.RegularUSD,
				Discount:   steamPrice.Discount,
				URL:        steamPrice.URL,
				OnSale:     steamPrice.Discount > 0,
				IsRegional: true,
			}
			main.Prices = append([]models.StorePrice{steamStore}, main.Prices...)
		}

		// Instant Gaming — link directo sin precio (scraping bloqueado)
		main.Prices = append(main.Prices, models.StorePrice{
			StoreName: "Instant Gaming",
			URL:       fmt.Sprintf("https://www.instant-gaming.com/es/busqueda/?q=%s", url.QueryEscape(query)),
		})

		// Eneba — link directo sin precio
		main.Prices = append(main.Prices, models.StorePrice{
			StoreName: "Eneba",
			URL:       fmt.Sprintf("https://www.eneba.com/store/all?text=%s", url.QueryEscape(query)),
		})

		// G2A — link directo sin precio
		main.Prices = append(main.Prices, models.StorePrice{
			StoreName: "G2A",
			URL:       fmt.Sprintf("https://www.g2a.com/search?query=%s", url.QueryEscape(query)),
		})

		// MundoSteam — siempre con advertencia, nunca como mejor deal
		main.Prices = append(main.Prices, models.StorePrice{
			StoreName: "MundoSteam",
			URL:       fmt.Sprintf("https://mundosteam.com/buscar?q=%s", url.QueryEscape(query)),
			Warning:   "Esta tienda vende acceso a cuentas compartidas, no el juego en tu cuenta personal. No recomendamos su uso.",
		})

		// Calculamos best deal ignorando tiendas sin precio y MundoSteam
		var best *models.StorePrice
		for i := range main.Prices {
			p := &main.Prices[i]

			if p.StoreName == "MundoSteam" {
				continue
			}

			// Unificamos el precio a pesos para poder comparar manzanas con manzanas
			precioPesos := p.PriceARS
			if precioPesos == 0 && p.PriceUSD > 0 {
				precioPesos = p.PriceUSD * usdRate
			}

			// Si no tiene precio ni en dólares ni en pesos, lo salteamos
			if precioPesos == 0 {
				continue
			}

			if best == nil {
				best = p
			} else {
				bestPesos := best.PriceARS
				if bestPesos == 0 && best.PriceUSD > 0 {
					bestPesos = best.PriceUSD * usdRate
				}

				if precioPesos < bestPesos {
					best = p
				}
			}
		}
		main.BestDeal = best

		finalResults = append(finalResults, main)
		finalResults = append(finalResults, itadResults[1:]...)
	}

	json.NewEncoder(w).Encode(models.SearchResponse{
		Query:   query,
		Results: finalResults,
		USDRate: usdRate,
	})
}
