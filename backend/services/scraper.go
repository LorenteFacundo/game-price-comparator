package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

type ScrapedPrice struct {
	StoreName  string
	PriceARS   float64
	RegularARS float64
	Discount   int
	URL        string
	Found      bool
	Warning    string
}

type ScraperService struct{}

func NewScraperService() *ScraperService {
	return &ScraperService{}
}

func (s *ScraperService) ScrapeInstantGaming(title string) (*ScrapedPrice, error) {
	searchURL := fmt.Sprintf(
		"https://www.instant-gaming.com/es/busqueda/?q=%s",
		strings.ReplaceAll(title, " ", "+"),
	)

	var result ScrapedPrice
	result.StoreName = "Instant Gaming"

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
		colly.MaxDepth(1),
	)
	c.SetRequestTimeout(10 * time.Second)

	c.OnHTML(".item.maingame", func(e *colly.HTMLElement) {
		if result.Found {
			return
		}

		itemTitle := strings.ToLower(e.ChildText(".name"))
		searchTitle := strings.ToLower(title)
		if !strings.Contains(itemTitle, searchTitle) {
			return
		}

		priceStr := e.ChildText(".price")
		priceStr = cleanPrice(priceStr)
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price == 0 {
			return
		}

		result.PriceARS = price
		result.RegularARS = price

		discountStr := e.ChildText(".discount")
		discountStr = strings.TrimPrefix(discountStr, "-")
		discountStr = strings.TrimSuffix(discountStr, "%")
		if d, err := strconv.Atoi(strings.TrimSpace(discountStr)); err == nil {
			result.Discount = d
		}

		result.URL = e.ChildAttr("a.cover", "href")
		if result.URL == "" {
			result.URL = "https://www.instant-gaming.com/es/"
		}
		result.Found = true
	})

	c.Visit(searchURL)

	return &result, nil
}

func (s *ScraperService) ScrapeEneba(title string) (*ScrapedPrice, error) {
	searchURL := fmt.Sprintf(
		"https://www.eneba.com/store/all?text=%s&currency=ARS",
		strings.ReplaceAll(title, " ", "+"),
	)

	var result ScrapedPrice
	result.StoreName = "Eneba"

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
		colly.MaxDepth(1),
	)
	c.SetRequestTimeout(10 * time.Second)

	c.OnHTML("[data-test-id='productCard']", func(e *colly.HTMLElement) {
		if result.Found {
			return
		}

		itemTitle := strings.ToLower(e.ChildText("[data-test-id='productName']"))
		if !strings.Contains(itemTitle, strings.ToLower(title)) {
			return
		}

		priceStr := e.ChildText("[data-test-id='price']")
		priceStr = cleanPrice(priceStr)
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price == 0 {
			return
		}

		result.PriceARS = price
		result.RegularARS = price
		result.URL = "https://www.eneba.com" + e.ChildAttr("a", "href")
		result.Found = true
	})

	c.Visit(searchURL)

	return &result, nil
}

func (s *ScraperService) ScrapeG2A(title string) (*ScrapedPrice, error) {
	searchURL := fmt.Sprintf(
		"https://www.g2a.com/search?query=%s",
		strings.ReplaceAll(title, " ", "+"),
	)

	var result ScrapedPrice
	result.StoreName = "G2A"

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
		colly.MaxDepth(1),
	)
	c.SetRequestTimeout(10 * time.Second)

	c.OnHTML(".x-hit", func(e *colly.HTMLElement) {
		if result.Found {
			return
		}

		itemTitle := strings.ToLower(e.ChildText(".x-hit__name"))
		if !strings.Contains(itemTitle, strings.ToLower(title)) {
			return
		}

		priceStr := e.ChildText(".x-hit__price-val")
		priceStr = cleanPrice(priceStr)
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price == 0 {
			return
		}

		// G2A devuelve USD, lo marcamos como tal
		result.PriceARS = price
		result.RegularARS = price
		result.URL = "https://www.g2a.com" + e.ChildAttr("a", "href")
		result.Found = true
	})

	c.Visit(searchURL)

	return &result, nil
}

func (s *ScraperService) MundoSteamEntry(title string) *ScrapedPrice {
	return &ScrapedPrice{
		StoreName: "MundoSteam",
		Found:     false, // nunca mostramos precio de MundoSteam como deal real
		Warning:   "Esta tienda vende acceso a cuentas compartidas, no el juego en tu cuenta personal. No recomendamos su uso.",
		URL:       fmt.Sprintf("https://mundosteam.com/buscar?q=%s", strings.ReplaceAll(title, " ", "+")),
	}
}

// cleanPrice limpia strings de precio como "$1.234,56" → "1234.56"
func cleanPrice(s string) string {
	s = strings.TrimSpace(s)
	// Eliminar símbolos de moneda y espacios
	s = strings.NewReplacer("$", "", "€", "", "USD", "", "ARS", "", " ", "").Replace(s)
	// Formato europeo: punto como separador de miles, coma como decimal
	if strings.Contains(s, ",") && strings.Contains(s, ".") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ",", ".")
	}
	return strings.TrimSpace(s)
}
