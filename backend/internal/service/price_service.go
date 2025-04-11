package service

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"server/internal/models"

	"github.com/gorilla/websocket"
)

// PriceService manages price data for multiple timeframes
type PriceService struct {
	// Map of timeframe to candle data
	timeFrameData     map[models.TimeFrame][]models.CandleData
	timeFrameDataLock sync.RWMutex

	currentCandle *models.CandleData
	clients       map[*websocket.Conn]bool
	clientsLock   sync.RWMutex
	dataDir       string // Directory to store data files
}

// NewPriceService creates a new instance of PriceService
func NewPriceService(dataDir string) *PriceService {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("Error creating data directory: %v", err)
	}

	return &PriceService{
		timeFrameData: make(map[models.TimeFrame][]models.CandleData),
		clients:       make(map[*websocket.Conn]bool),
		dataDir:       dataDir,
	}
}

// Initialize generates historical data directly for each timeframe
func (ps *PriceService) Initialize(days int) {
	basePrice := 200.0
	volatility := 10.0
	now := time.Now()

	// Start from midnight 'days' days ago
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -days)

	// Define timeframes to generate data for
	timeframes := []models.TimeFrame{
		models.TimeFrame1Min,
		models.TimeFrame5Min,
		models.TimeFrame15Min,
		models.TimeFrame1Hour,
		models.TimeFrame4Hour,
		models.TimeFrame1Day,
	}

	// Generate data for each timeframe independently
	for _, tf := range timeframes {
		log.Printf("Generating data for timeframe %s...", tf)

		// Get the duration for this timeframe
		duration := tf.GetDuration()

		// Estimate number of candles to generate
		totalDuration := now.Sub(startDate)
		estimatedCount := int(totalDuration / duration)

		candles := make([]models.CandleData, 0, estimatedCount)

		// Initialize price variables for this timeframe
		currentPrice := basePrice
		lastClose := basePrice

		// Generate candles for this timeframe
		for t := startDate; t.Before(now); t = t.Add(duration) {
			// Normalize timestamp to the beginning of the period
			timestamp := tf.NormalizeTimestamp(t.Unix() * 1000)

			// Skip if we already have a candle with this timestamp
			// (possible with normalized timestamps)
			isDuplicate := false
			for _, c := range candles {
				if c.Timestamp == timestamp {
					isDuplicate = true
					break
				}
			}
			if isDuplicate {
				continue
			}

			// Adjust volatility based on timeframe - higher timeframes have more volatility
			adjustedVolatility := volatility
			switch tf {
			case models.TimeFrame5Min:
				adjustedVolatility *= 1.5
			case models.TimeFrame15Min:
				adjustedVolatility *= 2
			case models.TimeFrame1Hour:
				adjustedVolatility *= 3
			case models.TimeFrame4Hour:
				adjustedVolatility *= 4
			case models.TimeFrame1Day:
				adjustedVolatility *= 6
			}

			// Generate realistic price movement
			change := (rand.Float64() - 0.5) * adjustedVolatility
			currentPrice = lastClose + change

			if currentPrice < 0 {
				currentPrice = 0 // Prevent negative prices
			}

			// Open should be close to the last close
			open := lastClose + (rand.Float64()-0.5)*(adjustedVolatility*0.1)

			// Generate high and low with more realistic ranges for timeframe
			var highLowRange float64
			switch tf {
			case models.TimeFrame1Min:
				highLowRange = adjustedVolatility * 0.5
			case models.TimeFrame5Min:
				highLowRange = adjustedVolatility * 0.8
			case models.TimeFrame15Min:
				highLowRange = adjustedVolatility * 1.0
			case models.TimeFrame1Hour:
				highLowRange = adjustedVolatility * 1.5
			case models.TimeFrame4Hour:
				highLowRange = adjustedVolatility * 2.0
			case models.TimeFrame1Day:
				highLowRange = adjustedVolatility * 3.0
			}

			high := math.Max(open, currentPrice) + rand.Float64()*highLowRange
			low := math.Min(open, currentPrice) - rand.Float64()*highLowRange

			// Ensure low is not greater than high
			if low > high {
				low = high - (rand.Float64() * highLowRange * 0.1)
			}

			open = math.Round(open*100) / 100
			high = math.Round(high*100) / 100
			low = math.Round(low*100) / 100
			close := math.Round(currentPrice*100) / 100

			lastClose = close

			// Generate volume appropriate for the timeframe
			// Higher timeframes have higher volume
			volumeBase := 1000.0
			var volumeMultiplier float64
			switch tf {
			case models.TimeFrame1Min:
				volumeMultiplier = 1
			case models.TimeFrame5Min:
				volumeMultiplier = 5
			case models.TimeFrame15Min:
				volumeMultiplier = 15
			case models.TimeFrame1Hour:
				volumeMultiplier = 60
			case models.TimeFrame4Hour:
				volumeMultiplier = 240
			case models.TimeFrame1Day:
				volumeMultiplier = 1440
			}

			volume := math.Round((rand.Float64()*volumeBase*volumeMultiplier)*100) / 100

			// Create candle
			candle := models.CandleData{
				Timestamp:  timestamp,
				Values:     [4]float64{open, high, low, close},
				IsComplete: true,
				Volume:     volume,
			}

			candles = append(candles, candle)
		}

		log.Printf("Generated %d candles for timeframe %s", len(candles), tf)

		// Store candles for this timeframe
		ps.timeFrameDataLock.Lock()
		ps.timeFrameData[tf] = candles
		ps.timeFrameDataLock.Unlock()

		// Save timeframe data immediately
		if err := ps.SaveTimeFrame(tf); err != nil {
			log.Printf("Error saving data for %s: %v", tf, err)
		}
	}
}

// StartNewCandle creates a new current candle based on the last price
func (ps *PriceService) StartNewCandle() {
	ps.timeFrameDataLock.RLock()
	minuteCandles, ok := ps.timeFrameData[models.TimeFrame1Min]
	var lastClose float64
	var lastTimestamp int64

	if ok && len(minuteCandles) > 0 {
		lastCandle := minuteCandles[len(minuteCandles)-1]
		lastClose = lastCandle.Values[3]
		lastTimestamp = lastCandle.Timestamp
	} else {
		lastClose = 200.0 // Default starting price
		lastTimestamp = time.Now().Add(-time.Minute).Unix() * 1000
	}
	ps.timeFrameDataLock.RUnlock()

	// Small random change for the open price
	change := (rand.Float64() - 0.5) * 1.0
	open := lastClose + change
	open = math.Round(open*100) / 100

	// Create new candle with only open price initially
	now := time.Now()
	timestamp := models.TimeFrame1Min.NormalizeTimestamp(now.Unix() * 1000)

	// Ensure the new timestamp is greater than the last one
	if timestamp <= lastTimestamp {
		timestamp = lastTimestamp + 60000 // One minute later
	}

	// Generate random volume
	volume := math.Round(rand.Float64()*100) / 100

	newCandle := models.CandleData{
		Timestamp:  timestamp,
		Values:     [4]float64{open, open, open, open}, // Initialize with open price
		IsComplete: false,
		Volume:     volume,
	}

	ps.currentCandle = &newCandle

	// Broadcast the new candle to all clients
	ps.broadcastToClients(models.UpdateMessage{
		Type:      "new",
		Candle:    newCandle,
		TimeFrame: models.TimeFrame1Min,
	})

	log.Printf("Started new 1-minute candle: Open: %.2f", open)
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
	volatility := rand.Float64() * 2.5 // Random volatility between 0 and 2.5
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

	// Increase volume slightly
	ps.currentCandle.Volume += math.Round(rand.Float64()*5) / 100

	// Broadcast the update to all clients
	ps.broadcastToClients(models.UpdateMessage{
		Type:      "update",
		Candle:    *ps.currentCandle,
		TimeFrame: models.TimeFrame1Min,
	})
}

// FinalizeCurrentCandle completes the current candle and adds it to history
func (ps *PriceService) FinalizeCurrentCandle() {
	if ps.currentCandle == nil {
		return
	}

	// Mark the candle as complete
	ps.currentCandle.IsComplete = true
	finalCandle := *ps.currentCandle

	// Add to history for 1-minute timeframe
	ps.timeFrameDataLock.Lock()

	// Ensure the 1-minute slice exists
	if _, ok := ps.timeFrameData[models.TimeFrame1Min]; !ok {
		ps.timeFrameData[models.TimeFrame1Min] = make([]models.CandleData, 0)
	}

	ps.timeFrameData[models.TimeFrame1Min] = append(ps.timeFrameData[models.TimeFrame1Min], finalCandle)
	ps.timeFrameDataLock.Unlock()

	// Broadcast the final update with isComplete flag
	ps.broadcastToClients(models.UpdateMessage{
		Type:      "update",
		Candle:    finalCandle,
		TimeFrame: models.TimeFrame1Min,
	})

	log.Printf("Finalized 1-minute candle: Open: %.2f, Close: %.2f",
		finalCandle.Values[0], finalCandle.Values[3])

	// Update higher timeframes if needed
	ps.updateHigherTimeframes(finalCandle)

	// Save 1-minute data periodically (every 15 minutes)
	if time.Now().Minute()%15 == 0 {
		ps.SaveTimeFrame(models.TimeFrame1Min)
	}

	// Reset current candle
	ps.currentCandle = nil
}

// updateHigherTimeframes updates aggregated timeframes when a new 1-minute candle is finalized
func (ps *PriceService) updateHigherTimeframes(newCandle models.CandleData) {
	timeframes := []models.TimeFrame{
		models.TimeFrame5Min,
		models.TimeFrame15Min,
		models.TimeFrame1Hour,
		models.TimeFrame4Hour,
		models.TimeFrame1Day,
	}

	ps.timeFrameDataLock.Lock()
	defer ps.timeFrameDataLock.Unlock()

	for _, tf := range timeframes {
		// Get normalized timestamp for this timeframe
		normalizedTimestamp := tf.NormalizeTimestamp(newCandle.Timestamp)

		// Check if we have candles for this timeframe
		if _, ok := ps.timeFrameData[tf]; !ok {
			ps.timeFrameData[tf] = make([]models.CandleData, 0)
		}

		// Find or create a candle for this timestamp
		var candle *models.CandleData

		for i, c := range ps.timeFrameData[tf] {
			if c.Timestamp == normalizedTimestamp {
				candle = &ps.timeFrameData[tf][i]
				break
			}
		}

		// If no candle exists for this timestamp, create one
		if candle == nil {
			// This is a new candle for this timeframe
			newTimeframeCandle := models.CandleData{
				Timestamp:  normalizedTimestamp,
				Values:     [4]float64{newCandle.Values[0], newCandle.Values[1], newCandle.Values[2], newCandle.Values[3]},
				IsComplete: false,
				Volume:     newCandle.Volume,
			}

			ps.timeFrameData[tf] = append(ps.timeFrameData[tf], newTimeframeCandle)

			// Broadcast the new candle to clients
			ps.broadcastToClients(models.UpdateMessage{
				Type:      "new",
				Candle:    newTimeframeCandle,
				TimeFrame: tf,
			})

			continue
		}

		// Update existing candle
		if newCandle.Values[1] > candle.Values[1] {
			candle.Values[1] = newCandle.Values[1] // Update high
		}
		if newCandle.Values[2] < candle.Values[2] {
			candle.Values[2] = newCandle.Values[2] // Update low
		}

		// Update close
		candle.Values[3] = newCandle.Values[3]

		// Add volume
		candle.Volume += newCandle.Volume

		// Broadcast the update
		ps.broadcastToClients(models.UpdateMessage{
			Type:      "update",
			Candle:    *candle,
			TimeFrame: tf,
		})

		// Check if this candle is now complete based on the timeframe
		now := time.Now()
		candleEndTime := time.Unix(normalizedTimestamp/1000, 0).Add(tf.GetDuration())

		if now.After(candleEndTime) && !candle.IsComplete {
			candle.IsComplete = true

			// Save data periodically for higher timeframes
			ps.SaveTimeFrame(tf)

			// Broadcast the finalized candle
			ps.broadcastToClients(models.UpdateMessage{
				Type:      "update",
				Candle:    *candle,
				TimeFrame: tf,
			})
		}
	}
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

// GetHistoryForTimeFrame returns historical candles for a specific timeframe
func (ps *PriceService) GetHistoryForTimeFrame(
	timeFrame models.TimeFrame,
	from, to int64,
	limit int,
) []models.CandleData {
	ps.timeFrameDataLock.RLock()
	defer ps.timeFrameDataLock.RUnlock()

	candles, ok := ps.timeFrameData[timeFrame]
	if !ok {
		return []models.CandleData{}
	}

	// Filter by time range if specified
	var filteredCandles []models.CandleData

	if from > 0 || to > 0 {
		for _, candle := range candles {
			if (from <= 0 || candle.Timestamp >= from) &&
				(to <= 0 || candle.Timestamp <= to) {
				filteredCandles = append(filteredCandles, candle)
			}
		}
	} else {
		filteredCandles = make([]models.CandleData, len(candles))
		copy(filteredCandles, candles)
	}

	// Apply limit if specified
	if limit > 0 && limit < len(filteredCandles) {
		// Return the most recent candles if limit is applied
		start := len(filteredCandles) - limit
		if start < 0 {
			start = 0
		}
		filteredCandles = filteredCandles[start:]
	}

	// If we have a current candle and this is the 1-minute timeframe, add it
	if timeFrame == models.TimeFrame1Min && ps.currentCandle != nil {
		filteredCandles = append(filteredCandles, *ps.currentCandle)
	}

	return filteredCandles
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

// SaveTimeFrame saves data for a specific timeframe to a file
func (ps *PriceService) SaveTimeFrame(timeFrame models.TimeFrame) error {
	ps.timeFrameDataLock.RLock()
	candles, ok := ps.timeFrameData[timeFrame]
	ps.timeFrameDataLock.RUnlock()

	if !ok {
		return fmt.Errorf("no data for timeframe %s", timeFrame)
	}

	// Create a copy of the data to avoid potential race conditions
	candlesCopy := make([]models.CandleData, len(candles))
	copy(candlesCopy, candles)

	filename := filepath.Join(ps.dataDir, fmt.Sprintf("price_history_%s.json", timeFrame))

	data, err := json.Marshal(candlesCopy)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

// SaveAllTimeFrames saves data for all timeframes
func (ps *PriceService) SaveAllTimeFrames() {
	timeframes := []models.TimeFrame{
		models.TimeFrame1Min,
		models.TimeFrame5Min,
		models.TimeFrame15Min,
		models.TimeFrame1Hour,
		models.TimeFrame4Hour,
		models.TimeFrame1Day,
	}

	for _, tf := range timeframes {
		if err := ps.SaveTimeFrame(tf); err != nil {
			log.Printf("Error saving data for %s: %v", tf, err)
		}
	}
}

// LoadAllTimeFrames loads data for all timeframes
func (ps *PriceService) LoadAllTimeFrames() error {
	timeframes := []models.TimeFrame{
		models.TimeFrame1Min,
		models.TimeFrame5Min,
		models.TimeFrame15Min,
		models.TimeFrame1Hour,
		models.TimeFrame4Hour,
		models.TimeFrame1Day,
	}

	var loadErr error
	dataLoaded := false

	for _, tf := range timeframes {
		err := ps.LoadTimeFrame(tf)
		if err == nil {
			dataLoaded = true
		} else if !os.IsNotExist(err) {
			// Only store errors that aren't "file not found"
			loadErr = err
		}
	}

	if !dataLoaded {
		return fmt.Errorf("no data files found")
	}

	return loadErr
}

// LoadTimeFrame loads data for a specific timeframe from a file
func (ps *PriceService) LoadTimeFrame(timeFrame models.TimeFrame) error {
	filename := filepath.Join(ps.dataDir, fmt.Sprintf("price_history_%s.json", timeFrame))

	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var candles []models.CandleData
	if err := json.Unmarshal(data, &candles); err != nil {
		return err
	}

	ps.timeFrameDataLock.Lock()
	ps.timeFrameData[timeFrame] = candles
	ps.timeFrameDataLock.Unlock()

	return nil
}
