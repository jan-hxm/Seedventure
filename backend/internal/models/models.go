package models

// CandleData represents OHLC data for a specific time
type CandleData struct {
	Timestamp  int64      `json:"x"`
	Values     [4]float64 `json:"y"`                    // [open, high, low, close]
	IsComplete bool       `json:"isComplete,omitempty"` // Flag to indicate if the candle is complete
}

// UpdateMessage represents a message sent to the client
type UpdateMessage struct {
	Type   string     `json:"type"` // "new" or "update"
	Candle CandleData `json:"candle"`
}
