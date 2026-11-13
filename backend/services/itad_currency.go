package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

// IsThereAnyDeal devuelve los precios de Argentina convertidos a ARS con su
// propia cotización y sin impuestos, aunque Steam, Epic y casi todas las
// tiendas cobran en dólares. Para calcular bien el total con tarjeta hay que
// recuperar el precio original en USD.
//
// La cotización de ITAD se calibra comparando el precio normal en ARS que
// informa para algunos juegos de referencia con el precio en USD que publica
// la propia tienda de Steam para Argentina.

// nativeARSStores son las tiendas que cobran en pesos en Argentina; sus
// precios se mantienen tal como los informa ITAD.
var nativeARSStores = map[string]struct{}{
	"microsoft store": {},
}

// calibrationApps son juegos de Steam con precio estable, usados para estimar
// la cotización que aplica ITAD.
var calibrationApps = []string{"1145360", "8930", "489830", "292030", "413150"}

const itadRateTTL = time.Hour

func isNativeARSStore(store string) bool {
	_, ok := nativeARSStores[strings.ToLower(strings.TrimSpace(store))]
	return ok
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

// restoreDollarDeal devuelve el precio de la tienda en USD cuando ITAD lo
// convirtió a ARS. Los precios de tiendas que cobran en pesos no se tocan.
func restoreDollarDeal(deal itadDeal, rate float64) itadDeal {
	if rate <= 0 || !strings.EqualFold(deal.Price.Currency, "ARS") || isNativeARSStore(deal.Shop.Name) {
		return deal
	}
	deal.Price = itadMoney{Amount: round2(deal.Price.Amount / rate), Currency: "USD"}
	if strings.EqualFold(deal.Regular.Currency, "ARS") {
		deal.Regular = itadMoney{Amount: round2(deal.Regular.Amount / rate), Currency: "USD"}
	}
	if deal.HistoryLow != nil && strings.EqualFold(deal.HistoryLow.Currency, "ARS") {
		low := itadMoney{Amount: round2(deal.HistoryLow.Amount / rate), Currency: "USD"}
		deal.HistoryLow = &low
	}
	return deal
}

func (s *ITADService) restoreDollarPrices(ctx context.Context, results []itadPriceResult) []itadPriceResult {
	rate := s.arsPerUSD(ctx)
	for i := range results {
		for j := range results[i].Deals {
			results[i].Deals[j] = restoreDollarDeal(results[i].Deals[j], rate)
		}
	}
	return results
}

// arsPerUSD devuelve la cotización que usa ITAD, calibrada como máximo una vez
// por hora. Si no se puede calibrar devuelve la última conocida, o 0 (en ese
// caso los precios quedan como los informa ITAD).
func (s *ITADService) arsPerUSD(ctx context.Context) float64 {
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	if s.rate > 0 && time.Since(s.rateAt) < itadRateTTL {
		return s.rate
	}
	rate, err := s.calibrateRate(ctx)
	if err != nil {
		log.Printf("itad_rate_calibration_error error=%v", err)
		return s.rate
	}
	s.rate, s.rateAt = rate, time.Now()
	return rate
}

func (s *ITADService) calibrateRate(ctx context.Context) (float64, error) {
	itadIDs, err := s.lookupSteamApps(ctx, calibrationApps)
	if err != nil {
		return 0, err
	}
	ids := make([]string, 0, len(itadIDs))
	appByITAD := make(map[string]string, len(itadIDs))
	for appID, itadID := range itadIDs {
		ids = append(ids, itadID)
		appByITAD[itadID] = appID
	}
	if len(ids) == 0 {
		return 0, fmt.Errorf("calibración ITAD: no se encontraron juegos de referencia")
	}
	prices, err := s.getPrices(ctx, ids)
	if err != nil {
		return 0, err
	}
	steamUSD, err := s.steamRegularUSD(ctx, calibrationApps)
	if err != nil {
		return 0, err
	}
	ratios := make([]float64, 0, len(prices))
	for _, result := range prices {
		usd := steamUSD[appByITAD[result.ID]]
		if usd <= 0 {
			continue
		}
		for _, deal := range result.Deals {
			if strings.EqualFold(deal.Shop.Name, "Steam") && strings.EqualFold(deal.Regular.Currency, "ARS") && deal.Regular.Amount > 0 {
				ratios = append(ratios, deal.Regular.Amount/usd)
			}
		}
	}
	if len(ratios) < 2 {
		return 0, fmt.Errorf("calibración ITAD: sólo %d referencias válidas", len(ratios))
	}
	sort.Float64s(ratios)
	rate := ratios[len(ratios)/2]
	if rate < 10 || rate > 100_000 {
		return 0, fmt.Errorf("calibración ITAD: cotización fuera de rango %.2f", rate)
	}
	return rate, nil
}

// lookupSteamApps traduce appIDs de Steam a IDs de ITAD (appID -> itadID).
func (s *ITADService) lookupSteamApps(ctx context.Context, appIDs []string) (map[string]string, error) {
	keys := make([]string, 0, len(appIDs))
	for _, appID := range appIDs {
		keys = append(keys, "app/"+appID)
	}
	body, err := json.Marshal(keys)
	if err != nil {
		return nil, err
	}
	req, err := s.newRequest(ctx, http.MethodPost, fmt.Sprintf("%s/lookup/id/shop/%d/v1", s.baseURL, steamShopID), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	var lookup map[string]*string
	if err := s.doJSON(req, &lookup); err != nil {
		return nil, fmt.Errorf("lookup Steam en ITAD: %w", err)
	}
	result := make(map[string]string, len(lookup))
	for key, itadID := range lookup {
		if itadID != nil && *itadID != "" {
			result[strings.TrimPrefix(key, "app/")] = *itadID
		}
	}
	return result, nil
}

// steamRegularUSD devuelve el precio normal (sin descuento) en USD que publica
// Steam para Argentina, en una sola consulta.
func (s *ITADService) steamRegularUSD(ctx context.Context, appIDs []string) (map[string]float64, error) {
	endpoint := fmt.Sprintf("%s/api/appdetails?appids=%s&cc=AR&filters=price_overview", s.steamStoreURL, strings.Join(appIDs, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("precios Steam para calibrar: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("precios Steam para calibrar respondió con status %d", resp.StatusCode)
	}
	var payload map[string]struct {
		Success bool `json:"success"`
		Data    struct {
			PriceOverview struct {
				Currency string `json:"currency"`
				Initial  int    `json:"initial"`
			} `json:"price_overview"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("parseando precios Steam para calibrar: %w", err)
	}
	prices := make(map[string]float64, len(payload))
	for appID, entry := range payload {
		overview := entry.Data.PriceOverview
		if entry.Success && strings.EqualFold(overview.Currency, "USD") && overview.Initial > 0 {
			prices[appID] = float64(overview.Initial) / 100
		}
	}
	return prices, nil
}
