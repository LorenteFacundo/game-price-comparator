package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthHandlerReturnsOperationalMetadata(t *testing.T) {
	started := time.Now().Add(-2 * time.Minute).UTC()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	response := httptest.NewRecorder()
	healthHandler(started, "abc123").ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected health response: status=%d headers=%v", response.Code, response.Header())
	}
	var body healthResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" || body.Version != "abc123" || body.UptimeSeconds < 100 || len(body.Providers) != 3 {
		t.Fatalf("unexpected health payload: %+v", body)
	}
}

func TestObservabilityAddsRequestIDAndSecurityHeaders(t *testing.T) {
	handler := withObservability(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || response.Header().Get("X-Request-ID") == "" {
		t.Fatalf("missing observability metadata: status=%d headers=%v", response.Code, response.Header())
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing security headers: %v", response.Header())
	}
}
