package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"server/internal/api"
	"server/internal/service"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Create and initialize price service
	priceService := service.NewPriceService()

	// Try to load historical data from file, or generate if file doesn't exist
	if err := priceService.LoadFromFile("price_history.json"); err != nil {
		log.Println("Generating new historical data")
		priceService.Initialize(5)

		// Save the generated data
		if err := priceService.SaveToFile("price_history.json"); err != nil {
			log.Println("Error saving data:", err)
		}
	}

	// Set up router
	r := mux.NewRouter()

	// Create a handler with the price service
	priceHandler := api.NewPriceHandler(priceService)

	// Define routes
	r.HandleFunc("/api/prices/history", priceHandler.HandleHistoricalData).Methods("GET")
	r.HandleFunc("/api/prices/live", priceHandler.HandleWebsocket)

	// Set up CORS
	corsMiddleware := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
	)

	// Start a new candle
	priceService.StartNewCandle()

	// Update current candle every 2 seconds, create new one every 10 seconds
	go func() {
		updateTicker := time.NewTicker(2 * time.Second)
		candleTicker := time.NewTicker(10 * time.Second)
		defer updateTicker.Stop()
		defer candleTicker.Stop()

		for {
			select {
			case <-updateTicker.C:
				priceService.UpdateCurrentCandle()
			case <-candleTicker.C:
				priceService.FinalizeCurrentCandle()
				priceService.StartNewCandle()
			}
		}
	}()

	// Start server
	port := 8080
	log.Printf("Server starting on port %d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), corsMiddleware(r)); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
