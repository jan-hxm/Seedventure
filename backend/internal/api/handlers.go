package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"server/internal/models"
	"server/internal/service"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// PriceHandler handles HTTP and WebSocket requests related to price data
type PriceHandler struct {
	priceService *service.PriceService
	upgrader     websocket.Upgrader
}

// NewPriceHandler creates a new instance of PriceHandler
func NewPriceHandler(priceService *service.PriceService) *PriceHandler {
	return &PriceHandler{
		priceService: priceService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all connections
			},
		},
	}
}

// HandleHistoricalData handles requests for historical price data with timeframe support
func (h *PriceHandler) HandleHistoricalData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get timeframe from query params, default to 1-minute
	timeFrameStr := r.URL.Query().Get("timeframe")
	timeFrame := models.TimeFrame1Min

	if timeFrameStr != "" {
		timeFrame = models.TimeFrame(timeFrameStr)
	}

	// Get optional from/to parameters (Unix timestamp in milliseconds)
	var from, to int64
	var err error

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		from, err = strconv.ParseInt(fromStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid 'from' parameter", http.StatusBadRequest)
			return
		}
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		to, err = strconv.ParseInt(toStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid 'to' parameter", http.StatusBadRequest)
			return
		}
	}

	// Get optional limit parameter
	limit := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "Invalid 'limit' parameter", http.StatusBadRequest)
			return
		}
	}

	// Get historical data for the requested timeframe
	history := h.priceService.GetHistoryForTimeFrame(timeFrame, from, to, limit)

	response := models.TimeFrameData{
		TimeFrame: timeFrame,
		Candles:   history,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleAvailableTimeframes returns the list of supported timeframes
func (h *PriceHandler) HandleAvailableTimeframes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	timeframes := []models.TimeFrame{
		models.TimeFrame1Min,
		models.TimeFrame5Min,
		models.TimeFrame15Min,
		models.TimeFrame1Hour,
		models.TimeFrame4Hour,
		models.TimeFrame1Day,
	}

	if err := json.NewEncoder(w).Encode(timeframes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleWebsocket handles websocket connections for live price updates (basic version)
func (h *PriceHandler) HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	// This method forwards to the more specific HandleWebsocketSubscribe with default timeframe
	vars := make(map[string]string)
	vars["timeframe"] = string(models.TimeFrame1Min)
	r = mux.SetURLVars(r, vars)
	h.HandleWebsocketSubscribe(w, r)
}

// HandleWebsocketSubscribe handles websocket connections with timeframe subscriptions
func (h *PriceHandler) HandleWebsocketSubscribe(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Get timeframe from URL parameters, default to 1-minute
	vars := mux.Vars(r)
	timeFrameStr := vars["timeframe"]
	timeFrame := models.TimeFrame1Min

	if timeFrameStr != "" {
		timeFrame = models.TimeFrame(timeFrameStr)
	}

	// Register client with the price service
	h.priceService.RegisterClient(conn)

	// Send current candle immediately if it exists and matches the requested timeframe
	if timeFrame == models.TimeFrame1Min {
		currentCandle := h.priceService.GetCurrentCandle()
		if currentCandle != nil {
			data, err := json.Marshal(models.UpdateMessage{
				Type:      "update",
				Candle:    *currentCandle,
				TimeFrame: timeFrame,
			})
			if err == nil {
				conn.WriteMessage(websocket.TextMessage, data)
			}
		}
	}

	// Handle client messages (e.g., change timeframe subscription)
	go func() {
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				h.priceService.UnregisterClient(conn)
				conn.Close()
				break
			}

			// If client sends a new timeframe request, handle it
			if messageType == websocket.TextMessage {
				var request models.TimeFrameRequest
				if err := json.Unmarshal(p, &request); err == nil {
					// Client wants to change timeframe
					log.Printf("Client requested timeframe change to %s", request.TimeFrame)

					// Send the initial data for the new timeframe
					history := h.priceService.GetHistoryForTimeFrame(
						request.TimeFrame,
						request.From,
						request.To,
						request.Limit,
					)

					response := models.TimeFrameData{
						TimeFrame: request.TimeFrame,
						Candles:   history,
					}

					data, err := json.Marshal(response)
					if err == nil {
						conn.WriteMessage(websocket.TextMessage, data)
					}
				}
			}
		}
	}()
}
