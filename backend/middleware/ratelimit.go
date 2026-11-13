// Package middleware contiene protecciones HTTP compartidas por la API.
package middleware

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// bucket implementa un token bucket simple: se recarga a `rate` tokens por
// segundo hasta `burst` y cada request consume uno.
type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// RateLimiter limita requests por IP y, además, aplica un tope global. El tope
// global protege la cuota de los proveedores externos aunque alguien rote IPs
// o falsifique X-Forwarded-For.
type RateLimiter struct {
	perIPRate   float64
	perIPBurst  float64
	globalRate  float64
	globalBurst float64
	idleTTL     time.Duration
	now         func() time.Time

	mu          sync.Mutex
	clients     map[string]*bucket
	global      bucket
	lastCleanup time.Time
}

// Config define los límites. Rates en requests por minuto.
type Config struct {
	PerIPPerMinute  float64
	PerIPBurst      int
	GlobalPerMinute float64
	GlobalBurst     int
}

func NewRateLimiter(cfg Config) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		perIPRate:   cfg.PerIPPerMinute / 60,
		perIPBurst:  float64(cfg.PerIPBurst),
		globalRate:  cfg.GlobalPerMinute / 60,
		globalBurst: float64(cfg.GlobalBurst),
		idleTTL:     10 * time.Minute,
		now:         time.Now,
		clients:     make(map[string]*bucket),
		global:      bucket{tokens: float64(cfg.GlobalBurst), lastSeen: now},
		lastCleanup: now,
	}
}

// Allow informa si la request puede pasar y, si no, cuántos segundos esperar.
func (l *RateLimiter) Allow(key string) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.cleanup(now)

	client, ok := l.clients[key]
	if !ok {
		client = &bucket{tokens: l.perIPBurst, lastSeen: now}
		l.clients[key] = client
	}
	refill(client, now, l.perIPRate, l.perIPBurst)
	refill(&l.global, now, l.globalRate, l.globalBurst)

	if client.tokens < 1 {
		return false, retryAfter(client.tokens, l.perIPRate)
	}
	if l.global.tokens < 1 {
		return false, retryAfter(l.global.tokens, l.globalRate)
	}
	client.tokens--
	l.global.tokens--
	return true, 0
}

// Middleware responde 429 con Retry-After cuando se supera el límite.
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, wait := l.Allow(ClientIP(r))
		if !allowed {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Retry-After", strconv.Itoa(wait))
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Demasiadas consultas seguidas. Esperá unos segundos y probá de nuevo."})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func refill(b *bucket, now time.Time, rate, burst float64) {
	elapsed := now.Sub(b.lastSeen).Seconds()
	if elapsed > 0 {
		b.tokens = math.Min(burst, b.tokens+elapsed*rate)
	}
	b.lastSeen = now
}

func retryAfter(tokens, rate float64) int {
	if rate <= 0 {
		return 60
	}
	return max(1, int(math.Ceil((1-tokens)/rate)))
}

func (l *RateLimiter) cleanup(now time.Time) {
	if now.Sub(l.lastCleanup) < time.Minute {
		return
	}
	l.lastCleanup = now
	for key, client := range l.clients {
		if now.Sub(client.lastSeen) > l.idleTTL {
			delete(l.clients, key)
		}
	}
}

// ClientIP obtiene la IP del cliente. Detrás de un proxy (Render, Netlify,
// Cloudflare) la IP real llega en X-Forwarded-For; es falsificable, por eso el
// limitador también tiene un tope global.
func ClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		first, _, _ := strings.Cut(forwarded, ",")
		if ip := strings.TrimSpace(first); ip != "" {
			return ip
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
