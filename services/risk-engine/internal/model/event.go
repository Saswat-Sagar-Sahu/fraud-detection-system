package model

import "time"

type EventType string

const (
	EventTypeLogin         EventType = "LOGIN"
	EventTypeTransaction   EventType = "TRANSACTION"
	EventTypeDeviceChange  EventType = "DEVICE_CHANGE"
	EventTypePasswordReset EventType = "PASSWORD_RESET"
)

type Event struct {
	EventID    string                 `json:"event_id"`
	EventType  EventType              `json:"event_type"`
	UserID     string                 `json:"user_id"`
	LoginID    string                 `json:"login_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	OccurredAt time.Time              `json:"occurred_at"`
	SourceIP   string                 `json:"source_ip"`
	DeviceID   string                 `json:"device_id"`
	Country    string                 `json:"country,omitempty"`
	Latitude   *float64               `json:"latitude,omitempty"`
	Longitude  *float64               `json:"longitude,omitempty"`
	Amount     *float64               `json:"amount,omitempty"`
	Currency   string                 `json:"currency,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
}
