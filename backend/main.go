package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"game-price-comparator/handlers"
	"game-price-comparator/services"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró .env; usando variables de entorno del sistema")
	}
	apiKey := os.Getenv("ITAD_API_KEY")
	if apiKey == "" {
		log.Fatal("ITAD_API_KEY no está configurada")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	searchHandler := handlers.NewSearchHandler(services.NewITADService(apiKey), services.NewCurrencyService(), services.NewSteamService(), taxRateFromEnv())
	startedAt := time.Now().UTC()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/search", searchHandler.Handle)
	mux.HandleFunc("/api/deals", searchHandler.HandleDeals)
	mux.HandleFunc("/api/discover", searchHandler.HandleDiscover)
	mux.HandleFunc("/api/health", healthHandler(startedAt, versionFromEnv()))
	allowedOrigins := []string{"http://localhost:5173", "http://localhost:3000"}
	if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}
	handler := withObservability(cors.New(cors.Options{AllowedOrigins: allowedOrigins, AllowedMethods: []string{http.MethodGet, http.MethodHead}, AllowedHeaders: []string{"Content-Type", "X-Request-ID"}, ExposedHeaders: []string{"X-Request-ID"}, MaxAge: 600}).Handler(mux))
	server := &http.Server{Addr: ":" + port, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("API escuchando en http://localhost:%s", port)
	log.Fatal(server.ListenAndServe())
}

func versionFromEnv() string {
	for _, key := range []string{"APP_VERSION", "RENDER_GIT_COMMIT", "RAILWAY_GIT_COMMIT_SHA"} {
		if value := os.Getenv(key); value != "" {
			if len(value) > 12 {
				return value[:12]
			}
			return value
		}
	}
	return "development"
}

// Estas piezas viven en main.go intencionalmente: el deploy histórico de
// Render compila `go build -o server main.go`, que no incluye otros archivos
// del package main.
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

func taxRateFromEnv() float64 {
	const defaultTaxRate = 0.21
	value := os.Getenv("DIGITAL_SERVICES_TAX_RATE")
	if value == "" {
		return defaultTaxRate
	}
	rate, err := strconv.ParseFloat(value, 64)
	if err != nil || rate < 0 || rate > 1 {
		log.Printf("DIGITAL_SERVICES_TAX_RATE inválida; usando %.0f%%", defaultTaxRate*100)
		return defaultTaxRate
	}
	return rate
}
