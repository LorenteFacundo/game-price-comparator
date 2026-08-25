package models

// StorePrice preserves the currency returned by the provider. Values are never
// silently converted on the server: conversion is only used for comparison
// when both ARS and USD are known.
type StorePrice struct {
	StoreName  string  `json:"store_name"`
	Price      float64 `json:"price"`
	Regular    float64 `json:"regular"`
	Currency   string  `json:"currency"`
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
	BestDeal *StorePrice  `json:"best_deal,omitempty"`
}

type FeaturedDeal struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	ImageURL   string  `json:"image_url"`
	StoreName  string  `json:"store_name"`
	Price      float64 `json:"price"`
	Regular    float64 `json:"regular"`
	Currency   string  `json:"currency"`
	Discount   int     `json:"discount_percent"`
	URL        string  `json:"url"`
	ExpiresAt  string  `json:"expires_at,omitempty"`
	HistoryLow float64 `json:"history_low,omitempty"`
	IsNearLow  bool    `json:"is_near_low"`
	PopularityRank int      `json:"popularity_rank,omitempty"`
	MatchedStores  []string `json:"matched_stores,omitempty"`
}

type SearchResponse struct {
	Query    string       `json:"query"`
	Results  []GameResult `json:"results"`
	USDRate  float64      `json:"usd_rate"`
	Warnings []string     `json:"warnings,omitempty"`
	Error    string       `json:"error,omitempty"`
}

type DealsResponse struct {
	Deals    []FeaturedDeal `json:"deals"`
	Warnings []string       `json:"warnings,omitempty"`
	Error    string         `json:"error,omitempty"`
}
