package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

type StockServer struct {
	mu           sync.Mutex
	currentPrice float64
	history      []float64
}

func NewStockServer() *StockServer {
	return &StockServer{
		currentPrice: 100.0, // Initial stock price
		history:      []float64{100.0},
	}
}

func (s *StockServer) updatePrice() {
	for {
		time.Sleep(1 * time.Second)
		s.mu.Lock()
		change := (rand.Float64() - 0.5) * 2 // Random change between -1 and 1
		s.currentPrice += change
		s.history = append(s.history, s.currentPrice)
		s.mu.Unlock()
	}
}

func (s *StockServer) getCurrentPrice(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	json.NewEncoder(w).Encode(map[string]float64{"currentPrice": s.currentPrice})
}

func (s *StockServer) getHistoricalData(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	json.NewEncoder(w).Encode(s.history)
}

func main() {
	server := NewStockServer()
	go server.updatePrice()

	http.HandleFunc("/current", enableCORS(server.getCurrentPrice))
	http.HandleFunc("/history", enableCORS(server.getHistoricalData))

	fmt.Println("Server started at http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
