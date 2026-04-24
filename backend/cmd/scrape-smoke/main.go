package main

import (
	"fmt"
	"os"

	"game-price-comparator/services"
)

func main() {
	query := "Hades"
	if len(os.Args) > 1 && os.Args[1] != "" {
		query = os.Args[1]
	}

	svc := services.NewScraperService()

	fmt.Printf("Smoke test de scraping para: %s\n", query)

	ig, igErr := svc.ScrapeInstantGaming(query)
	fmt.Printf("Instant Gaming: %+v err=%v\n", ig, igErr)

	eneba, enebaErr := svc.ScrapeEneba(query)
	fmt.Printf("Eneba: %+v err=%v\n", eneba, enebaErr)

	g2a, g2aErr := svc.ScrapeG2A(query)
	fmt.Printf("G2A: %+v err=%v\n", g2a, g2aErr)
}
