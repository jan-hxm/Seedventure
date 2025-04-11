package service

import (
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"server/internal/models"

	"github.com/gorilla/websocket"
)

// PriceService manages price data
type PriceService struct {
	history        []models.CandleData
	historyLock    sync.RWMutex
	currentCandle  *models.CandleData
	clients        map[*websocket.Conn]bool
	clientsLock    sync.RWMutex
	candleDuration time.Duration
}

// NewPriceService creates a new instance of PriceService
func NewPriceService() *PriceService {
	return &PriceService{
		history:        make([]models.CandleData, 0),
		clients:        make(map[*websocket.Conn]bool),
		candleDuration: 60 * time.Second, // Default 1 min candles
	}
}

// Initialize generates initial historical data
func (ps *PriceService) Initialize(count int) {
	basePrice := 200.0
	volatility := 10.0
	now := time.Now()

	ps.historyLock.Lock()
	defer ps.historyLock.Unlock()

	for i := 0; i < count; i++ {
		// Go back 'count' days from today for historical data
		date := now.AddDate(0, 0, i-count)
		timestamp := date.Unix() * 1000 // Convert to milliseconds for JS

		// Generate random price movements
		change := (rand.Float64() - 0.5) * volatility
		basePrice += change

		open := basePrice
		high := open + rand.Float64()*5
		low := open - rand.Float64()*5

		// Ensure low is not greater than high
		if low > high {
			low = high - 1
		}

		// Determine trend direction randomly
		isUptrend := rand.Float64() > 0.5
		var close float64
		if isUptrend {
			close = open + rand.Float64()*(high-open)
		} else {
			close = open - rand.Float64()*(open-low)
		}

		// Round to 2 decimal places
		open = math.Round(open*100) / 100
		high = math.Round(high*100) / 100
		low = math.Round(low*100) / 100
		close = math.Round(close*100) / 100

		ps.history = append(ps.history, models.CandleData{
			Timestamp:  timestamp,
			Values:     [4]float64{open, high, low, close},
			IsComplete: true,
		})
	}
}

// StartNewCandle creates a new current candle based on the last price
func (ps *PriceService) StartNewCandle() {
	var lastClose float64
	var lastTimestamp int64

	ps.historyLock.RLock()
	if len(ps.history) > 0 {
		lastCandle := ps.history[len(ps.history)-1]
		lastClose = lastCandle.Values[3]
		lastTimestamp = lastCandle.Timestamp
	} else {
		lastClose = 200.0 // Default starting price
		lastTimestamp = time.Now().Add(-ps.candleDuration).Unix() * 1000
	}
	ps.historyLock.RUnlock()

	// Small random change for the open price
	change := (rand.Float64() - 0.5) * 1.0
	open := lastClose + change
	open = math.Round(open*100) / 100

	// Create new candle with only open price initially
	now := time.Now()
	timestamp := now.Unix() * 1000 // Convert to milliseconds for JS

	// Ensure the new timestamp is greater than the last one
	if timestamp <= lastTimestamp {
		timestamp = lastTimestamp + 1000 // Ensure at least 1 second difference
	}

	newCandle := models.CandleData{
		Timestamp:  timestamp,
		Values:     [4]float64{open, open, open, open}, // Initialize with open price
		IsComplete: false,
	}

	ps.currentCandle = &newCandle

	// Broadcast the new candle to all clients
	ps.broadcastToClients(models.UpdateMessage{
		Type:   "new",
		Candle: newCandle,
	})

	log.Printf("Started new candle: Open: %.2f", open)
}

// UpdateCurrentCandle updates the current candle with a new price
func (ps *PriceService) UpdateCurrentCandle() {
	if ps.currentCandle == nil {
		ps.StartNewCandle()
		return
	}

	// Get current values
	open := ps.currentCandle.Values[0]
	high := ps.currentCandle.Values[1]
	low := ps.currentCandle.Values[2]

	// Generate a new random price movement
	volatility := 0.5
	lastClose := ps.currentCandle.Values[3]
	change := (rand.Float64() - 0.5) * volatility
	close := lastClose + change
	close = math.Round(close*100) / 100

	// Update high and low if needed
	if close > high {
		high = close
	}
	if close < low {
		low = close
	}

	// Update the current candle
	ps.currentCandle.Values = [4]float64{open, high, low, close}

	// Broadcast the update to all clients
	ps.broadcastToClients(models.UpdateMessage{
		Type:   "update",
		Candle: *ps.currentCandle,
	})

	log.Printf("Updated current candle: Open: %.2f, High: %.2f, Low: %.2f, Close: %.2f",
		open, high, low, close)
}

// FinalizeCurrentCandle completes the current candle and adds it to history
func (ps *PriceService) FinalizeCurrentCandle() {
	if ps.currentCandle == nil {
		return
	}

	// Mark the candle as complete
	ps.currentCandle.IsComplete = true
	finalCandle := *ps.currentCandle

	// Add to history
	ps.historyLock.Lock()
	ps.history = append(ps.history, finalCandle)
	ps.historyLock.Unlock()

	// Broadcast the final update with isComplete flag
	ps.broadcastToClients(models.UpdateMessage{
		Type:   "update",
		Candle: finalCandle,
	})

	log.Printf("Finalized candle: Open: %.2f, Close: %.2f",
		finalCandle.Values[0], finalCandle.Values[3])

	// Save to file
	if err := ps.SaveToFile("price_history.json"); err != nil {
		log.Println("Error saving data:", err)
	}

	// Reset current candle
	ps.currentCandle = nil
}

// GetHistory returns all historical candles
func (ps *PriceService) GetHistory() []models.CandleData {
	ps.historyLock.RLock()
	defer ps.historyLock.RUnlock()

	// Create a copy to avoid race conditions
	result := make([]models.CandleData, len(ps.history))
	copy(result, ps.history)

	// If we have a current candle, add it to the result
	if ps.currentCandle != nil {
		result = append(result, *ps.currentCandle)
	}

	return result
}

// GetCurrentCandle returns the current candle if it exists
func (ps *PriceService) GetCurrentCandle() *models.CandleData {
	if ps.currentCandle == nil {
		return nil
	}

	// Return a copy to avoid race conditions
	candle := *ps.currentCandle
	return &candle
}

// RegisterClient adds a new WebSocket client
func (ps *PriceService) RegisterClient(conn *websocket.Conn) {
	ps.clientsLock.Lock()
	defer ps.clientsLock.Unlock()
	ps.clients[conn] = true
}

// UnregisterClient removes a WebSocket client
func (ps *PriceService) UnregisterClient(conn *websocket.Conn) {
	ps.clientsLock.Lock()
	defer ps.clientsLock.Unlock()
	delete(ps.clients, conn)
}

// broadcastToClients sends a message to all connected clients
func (ps *PriceService) broadcastToClients(message models.UpdateMessage) {
	ps.clientsLock.RLock()
	defer ps.clientsLock.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling data:", err)
		return
	}

	for client := range ps.clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Println("Error sending message:", err)
			client.Close()
			ps.clientsLock.Lock()
			delete(ps.clients, client)
			ps.clientsLock.Unlock()
		}
	}
}

// SaveToFile saves historical data to a file
func (ps *PriceService) SaveToFile(filename string) error {
	ps.historyLock.RLock()
	defer ps.historyLock.RUnlock()

	data, err := json.Marshal(ps.history)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadFromFile loads historical data from a file
func (ps *PriceService) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	ps.historyLock.Lock()
	defer ps.historyLock.Unlock()

	return json.Unmarshal(data, &ps.history)
}
