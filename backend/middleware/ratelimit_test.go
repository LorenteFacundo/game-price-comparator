package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestLimiter(cfg Config) (*RateLimiter, *time.Time) {
	limiter := NewRateLimiter(cfg)
	current := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return current }
	limiter.global.lastSeen = current
	limiter.lastCleanup = current
	return limiter, &current
}

func TestRateLimiterBlocksAfterBurstAndRefills(t *testing.T) {
	limiter, current := newTestLimiter(Config{PerIPPerMinute: 60, PerIPBurst: 3, GlobalPerMinute: 6000, GlobalBurst: 100})
	for i := 0; i < 3; i++ {
		if ok, _ := limiter.Allow("1.1.1.1"); !ok {
			t.Fatalf("request %d should pass", i+1)
		}
	}
	ok, wait := limiter.Allow("1.1.1.1")
	if ok || wait < 1 {
		t.Fatalf("expected block with retry-after, got ok=%v wait=%d", ok, wait)
	}
	if ok, _ := limiter.Allow("2.2.2.2"); !ok {
		t.Fatal("other IPs must not be affected")
	}
	*current = current.Add(2 * time.Second)
	if ok, _ := limiter.Allow("1.1.1.1"); !ok {
		t.Fatal("bucket should refill over time")
	}
}

func TestRateLimiterGlobalCapAppliesAcrossIPs(t *testing.T) {
	limiter, _ := newTestLimiter(Config{PerIPPerMinute: 600, PerIPBurst: 10, GlobalPerMinute: 60, GlobalBurst: 2})
	limiter.Allow("1.1.1.1")
	limiter.Allow("2.2.2.2")
	if ok, _ := limiter.Allow("3.3.3.3"); ok {
		t.Fatal("global cap should block a new IP once exhausted")
	}
}

func TestMiddlewareReturns429WithRetryAfter(t *testing.T) {
	limiter, _ := newTestLimiter(Config{PerIPPerMinute: 60, PerIPBurst: 1, GlobalPerMinute: 600, GlobalBurst: 10})
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	request := httptest.NewRequest(http.MethodGet, "/api/search?q=hades", nil)
	request.Header.Set("X-Forwarded-For", "9.9.9.9, 10.0.0.1")

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, request)
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, request)

	if first.Code != http.StatusOK || second.Code != http.StatusTooManyRequests || second.Header().Get("Retry-After") == "" {
		t.Fatalf("unexpected statuses: first=%d second=%d headers=%v", first.Code, second.Code, second.Header())
	}
}

func TestClientIPPrefersForwardedHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "10.0.0.5:1234"
	if got := ClientIP(request); got != "10.0.0.5" {
		t.Fatalf("expected remote addr host, got %q", got)
	}
	request.Header.Set("X-Forwarded-For", " 203.0.113.7 , 10.0.0.1")
	if got := ClientIP(request); got != "203.0.113.7" {
		t.Fatalf("expected forwarded IP, got %q", got)
	}
}
