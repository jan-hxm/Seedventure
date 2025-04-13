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

	// Try to load historical data from files
	if err := priceService.LoadAllTimeFrames(); err != nil {
		log.Println("Generating new historical data:", err)

		// Generate 1 day of historical data
		priceService.Initialize(1)

		// Save the generated data
		priceService.SaveAllTimeFrames()
	}

	// Set up router
	r := mux.NewRouter()

	// Create a handler with the price service
	priceHandler := api.NewPriceHandler(priceService)

	// Define routes with timeframe support
	r.HandleFunc("/api/prices/history", priceHandler.HandleHistoricalData).Methods("GET")
	r.HandleFunc("/api/prices/timeframes", priceHandler.HandleAvailableTimeframes).Methods("GET")
	r.HandleFunc("/api/prices/live", priceHandler.HandleWebsocket)
	r.HandleFunc("/api/prices/live/{timeframe}", priceHandler.HandleWebsocketSubscribe)

	// Set up CORS
	corsMiddleware := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
	)

	// Start a new candle
	priceService.StartNewCandle()

	// Update current candle every second, create new one every minute
	go func() {
		updateTicker := time.NewTicker(time.Second)
		candleTicker := time.NewTicker(time.Minute)
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
