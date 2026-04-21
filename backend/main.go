package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"game-price-comparator/handlers"
	"game-price-comparator/services"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró .env, usando variables de entorno del sistema")
	}

	apiKey := os.Getenv("ITAD_API_KEY")
	if apiKey == "" {
		log.Fatal("ITAD_API_KEY no está configurada")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	itadSvc := services.NewITADService(apiKey)
	currencySvc := services.NewCurrencyService()
	steamSvc := services.NewSteamService()
	scraperSvc := services.NewScraperService()
	searchHandler := handlers.NewSearchHandler(itadSvc, currencySvc, steamSvc, scraperSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/search", searchHandler.Handle)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	frontendURL := os.Getenv("FRONTEND_URL")
	allowedOrigins := []string{"http://localhost:5173", "http://localhost:3000"}

	if frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}

	// CORS para que el frontend React pueda conectarse
	handler := cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{"GET"},
	}).Handler(mux)

	fmt.Printf("Backend corriendo en http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
