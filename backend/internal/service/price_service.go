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
	maxCandles    int    // Maximum number of candles to keep per timeframe
}

// NewPriceService creates a new instance of PriceService
func NewPriceService() *PriceService {
	// Create data directory if it doesn't exist
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("Error creating data directory: %v", err)
	}

	return &PriceService{
		timeFrameData: make(map[models.TimeFrame][]models.CandleData),
		clients:       make(map[*websocket.Conn]bool),
		dataDir:       dataDir,
		maxCandles:    100, // Store maximum of 100 candles per timeframe
	}
}

// Initialize generates historical data directly for each timeframe
func (ps *PriceService) Initialize(days int) {
	basePrice := 1.0
	volatility := 10.0
	now := time.Now()

	tf := models.TimeFrame1Min

	log.Printf("Generating data for timeframe %s...", tf)

	// We'll create 100 candles for the last 100 minutes
	numCandles := ps.maxCandles
	candles := make([]models.CandleData, 0, numCandles)

	// Initialize price variables for this timeframe
	currentPrice := basePrice
	lastClose := basePrice

	// Generate candles for the past 100 minutes
	for i := 0; i < numCandles; i++ {
		// Calculate timestamp for each candle, starting from (now - 99 minutes) to now
		// For the most recent 100 minutes, we go from (now - 99*minute) to now
		minutesAgo := int64(numCandles - 1 - i)
		candleTime := now.Add(-time.Duration(minutesAgo) * time.Minute)

		// Normalize timestamp to the beginning of the period
		timestamp := tf.NormalizeTimestamp(candleTime.Unix() * 1000)

		// Generate realistic price movement
		change := (rand.Float64() - 0.5) * volatility
		currentPrice = lastClose + change

		if currentPrice < 0 {
			currentPrice = 0 // Prevent negative prices
		}

		// Open should be close to the last close
		open := lastClose + (rand.Float64()-0.5)*(volatility*0.1)

		// Generate high and low with more realistic ranges for timeframe
		highLowRange := volatility * 0.5

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
		volumeBase := 1000.0
		volumeMultiplier := 1.0

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

	// Initialize higher timeframes based on 1-minute data
	ps.initializeHigherTimeframes()
}

// initializeHigherTimeframes creates initial data for higher timeframes from 1-minute data
func (ps *PriceService) initializeHigherTimeframes() {
	timeframes := []models.TimeFrame{
		models.TimeFrame5Min,
		models.TimeFrame15Min,
		models.TimeFrame1Hour,
		models.TimeFrame4Hour,
		models.TimeFrame1Day,
	}

	ps.timeFrameDataLock.RLock()
	minuteCandles := ps.timeFrameData[models.TimeFrame1Min]
	ps.timeFrameDataLock.RUnlock()

	// Process each timeframe
	for _, tf := range timeframes {
		// Map to group candles by normalized timestamp
		groupedCandles := make(map[int64]models.CandleData)

		// Group minute candles into higher timeframe buckets
		for _, candle := range minuteCandles {
			normalizedTimestamp := tf.NormalizeTimestamp(candle.Timestamp)

			// If this is a new timestamp, initialize the candle
			if existingCandle, exists := groupedCandles[normalizedTimestamp]; !exists {
				groupedCandles[normalizedTimestamp] = models.CandleData{
					Timestamp:  normalizedTimestamp,
					Values:     [4]float64{candle.Values[0], candle.Values[1], candle.Values[2], candle.Values[3]},
					IsComplete: true,
					Volume:     candle.Volume,
				}
			} else {
				// Update the existing candle
				updatedCandle := existingCandle

				// Keep the original open
				// Update high/low if needed
				if candle.Values[1] > updatedCandle.Values[1] {
					updatedCandle.Values[1] = candle.Values[1]
				}
				if candle.Values[2] < updatedCandle.Values[2] {
					updatedCandle.Values[2] = candle.Values[2]
				}

				// Set close to the newest candle
				updatedCandle.Values[3] = candle.Values[3]

				// Accumulate volume
				updatedCandle.Volume += candle.Volume

				groupedCandles[normalizedTimestamp] = updatedCandle
			}
		}

		// Convert map to slice and ensure we have at most maxCandles
		timeframeCandles := make([]models.CandleData, 0, len(groupedCandles))
		for _, candle := range groupedCandles {
			timeframeCandles = append(timeframeCandles, candle)
		}

		// Sort by timestamp (oldest first)
		// Note: In a real implementation, you might want to use a proper sorting function
		// For this example, we assume the data is already sorted by timestamp

		// Trim to maxCandles
		if len(timeframeCandles) > ps.maxCandles {
			timeframeCandles = timeframeCandles[len(timeframeCandles)-ps.maxCandles:]
		}

		// Store in timeFrameData
		ps.timeFrameDataLock.Lock()
		ps.timeFrameData[tf] = timeframeCandles
		ps.timeFrameDataLock.Unlock()

		// Save the timeframe data
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

	// Minimum price to avoid zero
	if open < 0.01 {
		open = 0.01
	}

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
	volatility := rand.Float64() * 10
	lastClose := ps.currentCandle.Values[3]
	change := (rand.Float64() - 0.5) * volatility
	close := lastClose + change
	close = math.Round(close*100) / 100

	// Minimum price to avoid zero
	if close < 0.01 {
		close = 0.01
	}

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

	// Add the new candle and maintain maximum size
	ps.timeFrameData[models.TimeFrame1Min] = append(ps.timeFrameData[models.TimeFrame1Min], finalCandle)
	if len(ps.timeFrameData[models.TimeFrame1Min]) > ps.maxCandles {
		ps.timeFrameData[models.TimeFrame1Min] = ps.timeFrameData[models.TimeFrame1Min][1:]
	}
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
		if err := ps.SaveTimeFrame(models.TimeFrame1Min); err != nil {
			log.Printf("Error saving 1-minute data: %v", err)
		}
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
		candleIndex := -1
		for i, c := range ps.timeFrameData[tf] {
			if c.Timestamp == normalizedTimestamp {
				candleIndex = i
				break
			}
		}

		// Check if this is a new period - we need to finalize the previous candle first
		// and potentially save data for this timeframe
		prevCandleFinalized := false
		if candleIndex == -1 {
			// Check if the most recent candle needs to be finalized
			if len(ps.timeFrameData[tf]) > 0 {
				lastCandle := &ps.timeFrameData[tf][len(ps.timeFrameData[tf])-1]
				if !lastCandle.IsComplete {
					lastCandle.IsComplete = true
					prevCandleFinalized = true

					// Broadcast the finalized candle
					ps.broadcastToClients(models.UpdateMessage{
						Type:      "update",
						Candle:    *lastCandle,
						TimeFrame: tf,
					})
				}
			}

			// This is a new candle for this timeframe
			newTimeframeCandle := models.CandleData{
				Timestamp:  normalizedTimestamp,
				Values:     [4]float64{newCandle.Values[0], newCandle.Values[1], newCandle.Values[2], newCandle.Values[3]},
				IsComplete: false,
				Volume:     newCandle.Volume,
			}

			ps.timeFrameData[tf] = append(ps.timeFrameData[tf], newTimeframeCandle)

			// Trim to maxCandles if needed
			if len(ps.timeFrameData[tf]) > ps.maxCandles {
				ps.timeFrameData[tf] = ps.timeFrameData[tf][1:]
			}

			// Broadcast the new candle to clients
			ps.broadcastToClients(models.UpdateMessage{
				Type:      "new",
				Candle:    newTimeframeCandle,
				TimeFrame: tf,
			})

			// Save the timeframe data if we finalized a candle
			if prevCandleFinalized {
				// We're inside a lock, so we need to save in a goroutine
				go func(timeFrame models.TimeFrame) {
					if err := ps.SaveTimeFrame(timeFrame); err != nil {
						log.Printf("Error saving data for %s: %v", timeFrame, err)
					}
				}(tf)
			}

			continue
		}

		// Update existing candle
		candle := &ps.timeFrameData[tf][candleIndex]

		// We only update high/low if needed
		if newCandle.Values[1] > candle.Values[1] {
			candle.Values[1] = newCandle.Values[1] // Update high
		}
		if newCandle.Values[2] < candle.Values[2] {
			candle.Values[2] = newCandle.Values[2] // Update low
		}

		// Always update close
		candle.Values[3] = newCandle.Values[3]

		// Add volume
		candle.Volume += newCandle.Volume

		// Broadcast the update
		ps.broadcastToClients(models.UpdateMessage{
			Type:      "update",
			Candle:    *candle,
			TimeFrame: tf,
		})

		// Check if this candle is now complete based on the timeframe duration
		now := time.Now()
		candleEndTime := time.Unix(normalizedTimestamp/1000, 0).Add(tf.GetDuration())

		if now.After(candleEndTime) && !candle.IsComplete {
			candle.IsComplete = true

			// Save data when we complete a candle
			go func(timeFrame models.TimeFrame) {
				if err := ps.SaveTimeFrame(timeFrame); err != nil {
					log.Printf("Error saving data for %s: %v", timeFrame, err)
				}
			}(tf)

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
func (ps *PriceService) GetHistoryForTimeFrame(timeFrame models.TimeFrame) []models.CandleData {
	ps.timeFrameDataLock.RLock()
	defer ps.timeFrameDataLock.RUnlock()

	candles, ok := ps.timeFrameData[timeFrame]
	if !ok {
		return []models.CandleData{}
	}

	// Create a copy of the candles
	filteredCandles := make([]models.CandleData, len(candles))
	copy(filteredCandles, candles)

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
	// Create a temporary lock to read the data
	ps.timeFrameDataLock.RLock()
	candles, ok := ps.timeFrameData[timeFrame]
	ps.timeFrameDataLock.RUnlock()

	if !ok {
		return fmt.Errorf("no data for timeframe %s", timeFrame)
	}

	// Create a copy of the data to avoid potential race conditions
	// and ensure we only save at most maxCandles
	var candlesCopy []models.CandleData
	if len(candles) <= ps.maxCandles {
		candlesCopy = make([]models.CandleData, len(candles))
		copy(candlesCopy, candles)
	} else {
		// Only save the most recent maxCandles
		startIdx := len(candles) - ps.maxCandles
		candlesCopy = make([]models.CandleData, ps.maxCandles)
		copy(candlesCopy, candles[startIdx:])
	}

	// Create a directory for the data file if it doesn't exist
	if err := os.MkdirAll(ps.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	filename := filepath.Join(ps.dataDir, fmt.Sprintf("price_history_%s.json", timeFrame))

	// Create a temporary file
	tempFile := filename + ".tmp"

	data, err := json.Marshal(candlesCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Write to the temporary file
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Rename the temporary file to the actual file (atomic operation)
	if err := os.Rename(tempFile, filename); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	log.Printf("Saved %d candles for timeframe %s", len(candlesCopy), timeFrame)
	return nil
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

	// Enforce maxCandles limit when loading
	if len(candles) > ps.maxCandles {
		startIdx := len(candles) - ps.maxCandles
		candles = candles[startIdx:]
	}

	ps.timeFrameDataLock.Lock()
	ps.timeFrameData[timeFrame] = candles
	ps.timeFrameDataLock.Unlock()

	log.Printf("Loaded %d candles for timeframe %s", len(candles), timeFrame)
	return nil
}
