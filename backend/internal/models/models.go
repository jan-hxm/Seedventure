package models

import (
	"time"
)

// TimeFrame represents a specific time interval for candles
type TimeFrame string

// Available timeframes
const (
	TimeFrame1Min  TimeFrame = "1m"
	TimeFrame5Min  TimeFrame = "5m"
	TimeFrame15Min TimeFrame = "15m"
	TimeFrame1Hour TimeFrame = "1h"
	TimeFrame4Hour TimeFrame = "4h"
	TimeFrame1Day  TimeFrame = "1d"
)

// CandleData represents OHLC data for a specific time
type CandleData struct {
	Timestamp  int64      `json:"x"`
	Values     [4]float64 `json:"y"`                    // [open, high, low, close]
	IsComplete bool       `json:"isComplete,omitempty"` // Flag to indicate if the candle is complete
	Volume     float64    `json:"volume,omitempty"`     // Optional volume data
}

// UpdateMessage represents a message sent to the client
type UpdateMessage struct {
	Type      string     `json:"type"` // "new" or "update"
	Candle    CandleData `json:"candle"`
	TimeFrame TimeFrame  `json:"timeFrame,omitempty"` // The timeframe of the candle
}

// TimeFrameRequest represents a request for historical data with a specific timeframe
type TimeFrameRequest struct {
	TimeFrame TimeFrame `json:"timeFrame"`
	From      int64     `json:"from,omitempty"`  // Optional start timestamp (Unix milliseconds)
	To        int64     `json:"to,omitempty"`    // Optional end timestamp (Unix milliseconds)
	Limit     int       `json:"limit,omitempty"` // Optional limit on number of candles
}

// TimeFrameData represents all historical data for a specific timeframe
type TimeFrameData struct {
	TimeFrame TimeFrame    `json:"timeFrame"`
	Candles   []CandleData `json:"candles"`
}

// GetDuration returns the duration of a timeframe
func (tf TimeFrame) GetDuration() time.Duration {
	switch tf {
	case TimeFrame1Min:
		return time.Minute
	case TimeFrame5Min:
		return 5 * time.Minute
	case TimeFrame15Min:
		return 15 * time.Minute
	case TimeFrame1Hour:
		return time.Hour
	case TimeFrame4Hour:
		return 4 * time.Hour
	case TimeFrame1Day:
		return 24 * time.Hour
	default:
		return time.Minute // Default to 1 minute
	}
}

// NormalizeTimestamp normalizes a timestamp to the beginning of the period for this timeframe
func (tf TimeFrame) NormalizeTimestamp(timestamp int64) int64 {
	// Convert from milliseconds to seconds for Go time functions
	t := time.Unix(timestamp/1000, 0)

	switch tf {
	case TimeFrame1Min:
		// Normalize to the beginning of the minute
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	case TimeFrame5Min:
		// Normalize to the beginning of the 5-minute period
		minute := t.Minute() - (t.Minute() % 5)
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location())
	case TimeFrame15Min:
		// Normalize to the beginning of the 15-minute period
		minute := t.Minute() - (t.Minute() % 15)
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), minute, 0, 0, t.Location())
	case TimeFrame1Hour:
		// Normalize to the beginning of the hour
		t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case TimeFrame4Hour:
		// Normalize to the beginning of the 4-hour period
		hour := t.Hour() - (t.Hour() % 4)
		t = time.Date(t.Year(), t.Month(), t.Day(), hour, 0, 0, 0, t.Location())
	case TimeFrame1Day:
		// Normalize to the beginning of the day
		t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}

	// Convert back to milliseconds
	return t.Unix() * 1000
}
