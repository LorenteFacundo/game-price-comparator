package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetReviewSummariesBatchesAppIDsAndParsesStoreItems(t *testing.T) {
	var requested []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/IStoreBrowseService/GetItems/v1/" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		var input struct {
			IDs []struct {
				AppID int `json:"appid"`
			} `json:"ids"`
			DataRequest struct {
				IncludeReviews bool `json:"include_reviews"`
			} `json:"data_request"`
		}
		if err := json.Unmarshal([]byte(r.URL.Query().Get("input_json")), &input); err != nil || !input.DataRequest.IncludeReviews {
			t.Fatalf("invalid input_json %q: %v", r.URL.Query().Get("input_json"), err)
		}
		for _, id := range input.IDs {
			requested = append(requested, id.AppID)
		}
		_, _ = w.Write([]byte(`{"response":{"store_items":[
			{"appid":489830,"success":1,"reviews":{"summary_filtered":{"review_count":223819,"percent_positive":94}}},
			{"appid":8930,"success":1},
			{"appid":0,"success":15}
		]}}`))
	}))
	defer server.Close()

	steam := &SteamService{client: server.Client(), apiBase: server.URL}
	summaries, err := steam.GetReviewSummaries(context.Background(), []string{"489830", "8930", "489830", "not-a-number", "999999999"})
	if err != nil {
		t.Fatal(err)
	}
	if len(requested) != 3 {
		t.Fatalf("expected 3 unique numeric appIDs in one batch, got %v", requested)
	}
	if got := summaries["489830"]; got.Percent != 94 || got.Count != 223819 {
		t.Fatalf("unexpected Skyrim summary: %+v", got)
	}
	if _, ok := summaries["8930"]; ok {
		t.Fatal("games without reviews must be omitted")
	}
}

func TestGetReviewSummariesReportsNonJSONResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<!DOCTYPE html><title>Welcome to Steam</title>"))
	}))
	defer server.Close()
	steam := &SteamService{client: server.Client(), apiBase: server.URL}
	if _, err := steam.GetReviewSummaries(context.Background(), []string{"1"}); err == nil {
		t.Fatal("expected an error when Steam answers with HTML")
	}
}
