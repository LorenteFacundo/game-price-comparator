package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

type healthResponse struct {
	Status        string   `json:"status"`
	Service       string   `json:"service"`
	Version       string   `json:"version"`
	StartedAt     string   `json:"started_at"`
	CheckedAt     string   `json:"checked_at"`
	UptimeSeconds int64    `json:"uptime_seconds"`
	Providers     []string `json:"providers"`
}

func healthHandler(startedAt time.Time, version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		now := time.Now().UTC()
		_ = json.NewEncoder(w).Encode(healthResponse{
			Status: "ok", Service: "pricepulse-api", Version: version,
			StartedAt: startedAt.Format(time.RFC3339), CheckedAt: now.Format(time.RFC3339),
			UptimeSeconds: int64(now.Sub(startedAt).Seconds()),
			Providers:     []string{"IsThereAnyDeal", "Steam", "Bluelytics"},
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	if recorder.status != 0 {
		return
	}
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *statusRecorder) Write(body []byte) (int, error) {
	if recorder.status == 0 {
		recorder.WriteHeader(http.StatusOK)
	}
	return recorder.ResponseWriter.Write(body)
}

func withObservability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		recorder := &statusRecorder{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic request_id=%s method=%s path=%s error=%v\n%s", requestID, r.Method, r.URL.Path, recovered, debug.Stack())
				if recorder.status == 0 {
					http.Error(recorder, "error interno", http.StatusInternalServerError)
				}
			}
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			log.Printf("request request_id=%s method=%s path=%s status=%d duration_ms=%d", requestID, r.Method, r.URL.Path, status, time.Since(started).Milliseconds())
		}()
		next.ServeHTTP(recorder, r)
	})
}

func newRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err == nil {
		return hex.EncodeToString(bytes)
	}
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
