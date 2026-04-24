package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SteamService struct {
	client *http.Client
}

type SteamPrice struct {
	Currency   string
	PriceARS   float64
	RegularARS float64
	PriceUSD   float64
	RegularUSD float64
	Discount   int
	URL        string
	Found      bool
}

type steamResponse map[string]struct {
	Success bool `json:"success"`
	Data    struct {
		PriceOverview struct {
			Currency        string `json:"currency"`
			Initial         int    `json:"initial"`
			Final           int    `json:"final"`
			DiscountPercent int    `json:"discount_percent"`
			FinalFormatted  string `json:"final_formatted"`
		} `json:"price_overview"`
		Steam_Appid int    `json:"steam_appid"`
		Name        string `json:"name"`
	} `json:"data"`
}

func NewSteamService() *SteamService {
	return &SteamService{
		client: &http.Client{Timeout: 8 * time.Second},
	}
}

// GetPriceByTitle busca el appid en Steam y trae el precio para el pais indicado.
func (s *SteamService) GetPriceByTitle(title, country string) (*SteamPrice, error) {
	appID, err := s.searchAppID(title)
	if err != nil || appID == "" {
		return &SteamPrice{Found: false}, nil
	}
	return s.GetPriceByAppID(appID, country)
}

func (s *SteamService) GetPriceByAppID(appID, country string) (*SteamPrice, error) {
	if country == "" {
		country = "AR"
	}

	endpoint := fmt.Sprintf(
		"https://store.steampowered.com/api/appdetails?appids=%s&country=%s&filters=price_overview",
		appID,
		url.QueryEscape(strings.ToUpper(country)),
	)

	resp, err := s.client.Get(endpoint)
	if err != nil {
		return &SteamPrice{Found: false}, nil
	}
	defer resp.Body.Close()

	var data steamResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return &SteamPrice{Found: false}, nil
	}

	entry, ok := data[appID]
	if !ok || !entry.Success {
		return &SteamPrice{Found: false}, nil
	}

	po := entry.Data.PriceOverview
	if po.Final == 0 && po.Initial == 0 {
		return &SteamPrice{Found: false}, nil
	}

	// El endpoint country=AR devuelve precios en ARS (centavos → pesos)
	price := &SteamPrice{
		Currency: strings.ToUpper(po.Currency),
		Discount: po.DiscountPercent,
		URL:      fmt.Sprintf("https://store.steampowered.com/app/%s/", appID),
		Found:    true,
	}

	switch price.Currency {
	case "ARS":
		price.PriceARS = float64(po.Final) / 100.0
		price.RegularARS = float64(po.Initial) / 100.0
	default:
		price.PriceUSD = float64(po.Final) / 100.0
		price.RegularUSD = float64(po.Initial) / 100.0
	}

	return price, nil
}

// searchAppID busca el appid de Steam por título usando la búsqueda de Steam
func (s *SteamService) searchAppID(title string) (string, error) {
	endpoint := fmt.Sprintf(
		"https://store.steampowered.com/api/storesearch?term=%s&l=spanish&cc=AR",
		url.QueryEscape(title),
	)

	resp, err := s.client.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Items) == 0 {
		return "", fmt.Errorf("no se encontraron resultados")
	}

	// 1. Buscar coincidencia EXACTA primero
	for _, item := range result.Items {
		if strings.EqualFold(item.Name, title) {
			return fmt.Sprintf("%d", item.ID), nil
		}
	}

	// 2. Fallback: primer resultado
	return fmt.Sprintf("%d", result.Items[0].ID), nil
}
