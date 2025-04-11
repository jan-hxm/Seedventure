package api

import (
	"encoding/json"
	"log"
	"net/http"

	"server/internal/models"
	"server/internal/service"

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

// HandleHistoricalData handles requests for historical price data
func (h *PriceHandler) HandleHistoricalData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	history := h.priceService.GetHistory()

	if err := json.NewEncoder(w).Encode(history); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleWebsocket handles websocket connections for live price updates
func (h *PriceHandler) HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Register client with the price service
	h.priceService.RegisterClient(conn)

	// Send current candle immediately if it exists
	currentCandle := h.priceService.GetCurrentCandle()
	if currentCandle != nil {
		data, err := json.Marshal(models.UpdateMessage{
			Type:   "update",
			Candle: *currentCandle,
		})
		if err == nil {
			conn.WriteMessage(websocket.TextMessage, data)
		}
	}

	// Handle disconnection
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				h.priceService.UnregisterClient(conn)
				conn.Close()
				break
			}
		}
	}()
}
