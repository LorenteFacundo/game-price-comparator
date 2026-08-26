package services

import (
	"encoding/json"
	"testing"
)

func TestBluelyticsResponseReadsOficialField(t *testing.T) {
	var response bluelyticsResponse
	err := json.Unmarshal([]byte(`{"blue":{"value_sell":1555},"oficial":{"value_sell":1533}}`), &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.Official.ValueSell != 1533 {
		t.Fatalf("expected oficial value 1533, got %.2f", response.Official.ValueSell)
	}
}
