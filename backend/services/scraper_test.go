package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCleanPrice(t *testing.T) {
	cases := map[string]string{
		"$1.234,56": "1234.56",
		"$12.345":   "12345",
		"USD 19.99": "19.99",
		"ARS 999":   "999",
		"€ 12,50":   "12.50",
	}

	for input, want := range cases {
		if got := cleanPrice(input); got != want {
			t.Fatalf("cleanPrice(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestScrapeInstantGamingURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<div class="item maingame">
				<div class="name">Hades</div>
				<div class="price">$ 12.345</div>
				<div class="discount">-20%</div>
				<a class="cover" href="/game/hades"></a>
			</div>
		`))
	}))
	defer server.Close()

	svc := NewScraperService()
	got, err := svc.scrapeInstantGamingURL("Hades", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Found || got.PriceARS != 12345 || got.Discount != 20 {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.URL != "https://www.instant-gaming.com/game/hades" {
		t.Fatalf("unexpected URL: %s", got.URL)
	}
}

func TestScrapeEnebaURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<div data-test-id="productCard">
				<div data-test-id="productName">Hades Global</div>
				<div data-test-id="price">ARS 9.999</div>
				<a href="/product/hades-key"></a>
			</div>
		`))
	}))
	defer server.Close()

	svc := NewScraperService()
	got, err := svc.scrapeEnebaURL("Hades", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Found || got.PriceARS != 9999 {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.URL != "https://www.eneba.com/product/hades-key" {
		t.Fatalf("unexpected URL: %s", got.URL)
	}
}

func TestScrapeG2AURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<div class="x-hit">
				<div class="x-hit__name">Hades Steam Key</div>
				<div class="x-hit__price-val">USD 19.99</div>
				<a href="/product/hades"></a>
			</div>
		`))
	}))
	defer server.Close()

	svc := NewScraperService()
	got, err := svc.scrapeG2AURL("Hades", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Found || got.PriceUSD != 19.99 || got.PriceARS != 0 {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.URL != "https://www.g2a.com/product/hades" {
		t.Fatalf("unexpected URL: %s", got.URL)
	}
}
