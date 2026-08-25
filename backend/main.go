package main

import (
	"log"
	"net/http"
	"os"
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
	mux := http.NewServeMux()
	mux.HandleFunc("/api/search", searchHandler.Handle)
	mux.HandleFunc("/api/deals", searchHandler.HandleDeals)
	mux.HandleFunc("/api/discover", searchHandler.HandleDiscover)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	allowedOrigins := []string{"http://localhost:5173", "http://localhost:3000"}
	if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}
	handler := cors.New(cors.Options{AllowedOrigins: allowedOrigins, AllowedMethods: []string{http.MethodGet}, AllowedHeaders: []string{"Content-Type"}, MaxAge: 600}).Handler(mux)
	server := &http.Server{Addr: ":" + port, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("API escuchando en http://localhost:%s", port)
	log.Fatal(server.ListenAndServe())
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
