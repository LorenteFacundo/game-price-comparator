package models

type ITADGame struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	Assets struct {
		BoxArt    string `json:"boxart"`
		Banner300 string `json:"banner300"`
	} `json:"assets"`
}

type StorePrice struct {
	StoreName  string  `json:"store_name"`
	PriceUSD   float64 `json:"price_usd"`
	PriceARS   float64 `json:"price_ars"`
	RegularUSD float64 `json:"regular_usd"`
	RegularARS float64 `json:"regular_ars"`
	Discount   int     `json:"discount_percent"`
	URL        string  `json:"url"`
	OnSale     bool    `json:"on_sale"`
	IsRegional bool    `json:"is_regional"`
	Warning    string  `json:"warning,omitempty"`
}

type GameResult struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	ImageURL string       `json:"image_url"`
	Prices   []StorePrice `json:"prices"`
	BestDeal *StorePrice  `json:"best_deal"`
}

type SearchResponse struct {
	Query   string       `json:"query"`
	Results []GameResult `json:"results"`
	USDRate float64      `json:"usd_rate"`
	Error   string       `json:"error,omitempty"`
}
