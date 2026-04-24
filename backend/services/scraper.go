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
	PriceUSD   float64
	PriceARS   float64
	RegularUSD float64
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
		"https://www.instant-gaming.com/en/search/?query=%s",
		strings.ReplaceAll(title, " ", "+"),
	)

	return s.scrapeInstantGamingURL(title, searchURL)
}

func (s *ScraperService) scrapeInstantGamingURL(title, searchURL string) (*ScrapedPrice, error) {
	result := ScrapedPrice{StoreName: "Instant Gaming"}
	c := newScraperCollector()

	c.OnHTML(".item.maingame", func(e *colly.HTMLElement) {
		if result.Found {
			return
		}

		itemTitle := strings.ToLower(e.ChildText(".name"))
		searchTitle := strings.ToLower(title)
		if !strings.Contains(itemTitle, searchTitle) {
			return
		}

		priceStr := cleanPrice(e.ChildText(".price"))
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

		result.URL = absoluteURL("https://www.instant-gaming.com", e.ChildAttr("a.cover", "href"))
		if result.URL == "" {
			result.URL = "https://www.instant-gaming.com/es/"
		}
		result.Found = true
	})

	if err := c.Visit(searchURL); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *ScraperService) ScrapeEneba(title string) (*ScrapedPrice, error) {
	searchURL := fmt.Sprintf(
		"https://www.eneba.com/store/all?text=%s&currency=ARS",
		strings.ReplaceAll(title, " ", "+"),
	)

	return s.scrapeEnebaURL(title, searchURL)
}

func (s *ScraperService) scrapeEnebaURL(title, searchURL string) (*ScrapedPrice, error) {
	result := ScrapedPrice{StoreName: "Eneba"}
	c := newScraperCollector()

	c.OnHTML("[data-test-id='productCard']", func(e *colly.HTMLElement) {
		if result.Found {
			return
		}

		itemTitle := strings.ToLower(e.ChildText("[data-test-id='productName']"))
		if !strings.Contains(itemTitle, strings.ToLower(title)) {
			return
		}

		priceStr := cleanPrice(e.ChildText("[data-test-id='price']"))
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price == 0 {
			return
		}

		result.PriceARS = price
		result.RegularARS = price
		result.URL = absoluteURL("https://www.eneba.com", e.ChildAttr("a", "href"))
		result.Found = true
	})

	if err := c.Visit(searchURL); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *ScraperService) ScrapeG2A(title string) (*ScrapedPrice, error) {
	searchURL := fmt.Sprintf(
		"https://www.g2a.com/search?query=%s",
		strings.ReplaceAll(title, " ", "+"),
	)

	return s.scrapeG2AURL(title, searchURL)
}

func (s *ScraperService) scrapeG2AURL(title, searchURL string) (*ScrapedPrice, error) {
	result := ScrapedPrice{StoreName: "G2A"}
	c := newScraperCollector()

	c.OnHTML(".x-hit", func(e *colly.HTMLElement) {
		if result.Found {
			return
		}

		itemTitle := strings.ToLower(e.ChildText(".x-hit__name"))
		if !strings.Contains(itemTitle, strings.ToLower(title)) {
			return
		}

		priceStr := cleanPrice(e.ChildText(".x-hit__price-val"))
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price == 0 {
			return
		}

		// G2A suele mostrar el precio base en USD.
		result.PriceUSD = price
		result.RegularUSD = price
		result.URL = absoluteURL("https://www.g2a.com", e.ChildAttr("a", "href"))
		result.Found = true
	})

	if err := c.Visit(searchURL); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *ScraperService) MundoSteamEntry(title string) *ScrapedPrice {
	return &ScrapedPrice{
		StoreName: "MundoSteam",
		Found:     false,
		Warning:   "Esta tienda vende acceso a cuentas compartidas, no el juego en tu cuenta personal. No recomendamos su uso.",
		URL:       fmt.Sprintf("https://mundosteam.com/buscar?q=%s", strings.ReplaceAll(title, " ", "+")),
	}
}

func newScraperCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
		colly.MaxDepth(1),
	)
	c.SetRequestTimeout(10 * time.Second)
	return c
}

func absoluteURL(baseURL, value string) string {
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return baseURL + value
}

// cleanPrice limpia strings como "$1.234,56" o "USD 19.99" y deja un numero parseable.
func cleanPrice(s string) string {
	s = strings.TrimSpace(s)
	s = strings.NewReplacer("$", "", "€", "", "USD", "", "ARS", "", " ", "").Replace(s)

	if strings.Contains(s, ",") && strings.Contains(s, ".") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		if len(parts) > 1 && len(parts[len(parts)-1]) == 3 {
			s = strings.Join(parts, "")
		}
	} else if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ",", ".")
	}

	return strings.TrimSpace(s)
}
