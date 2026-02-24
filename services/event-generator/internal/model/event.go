package model

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type EventType string

const (
	EventTypeLogin         EventType = "LOGIN"
	EventTypeTransaction   EventType = "TRANSACTION"
	EventTypeDeviceChange  EventType = "DEVICE_CHANGE"
	EventTypePasswordReset EventType = "PASSWORD_RESET"
)

var validEventTypes = map[EventType]struct{}{
	EventTypeLogin:         {},
	EventTypeTransaction:   {},
	EventTypeDeviceChange:  {},
	EventTypePasswordReset: {},
}

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

func (e Event) Validate() error {
	if strings.TrimSpace(e.EventID) == "" {
		return errors.New("event_id is required")
	}
	if _, ok := validEventTypes[e.EventType]; !ok {
		return fmt.Errorf("event_type must be one of LOGIN, TRANSACTION, DEVICE_CHANGE, PASSWORD_RESET")
	}
	if strings.TrimSpace(e.UserID) == "" {
		return errors.New("user_id is required")
	}
	if e.EventType == EventTypeLogin && strings.TrimSpace(e.LoginID) == "" {
		return errors.New("login_id is required for LOGIN event")
	}
	if e.OccurredAt.IsZero() {
		return errors.New("occurred_at is required and must be RFC3339 date-time")
	}
	if strings.TrimSpace(e.SourceIP) == "" || net.ParseIP(e.SourceIP) == nil {
		return errors.New("source_ip must be a valid IP address")
	}
	if strings.TrimSpace(e.DeviceID) == "" {
		return errors.New("device_id is required")
	}
	if e.Metadata == nil {
		return errors.New("metadata is required")
	}
	if (e.Latitude == nil) != (e.Longitude == nil) {
		return errors.New("latitude and longitude must be provided together")
	}
	if e.Latitude != nil {
		if *e.Latitude < -90 || *e.Latitude > 90 {
			return errors.New("latitude must be between -90 and 90")
		}
		if *e.Longitude < -180 || *e.Longitude > 180 {
			return errors.New("longitude must be between -180 and 180")
		}
	}
	if e.EventType == EventTypeTransaction {
		if e.Amount == nil || *e.Amount < 0 {
			return errors.New("amount is required and must be >= 0 for TRANSACTION")
		}
		if len(strings.TrimSpace(e.Currency)) != 3 {
			return errors.New("currency is required as 3-letter code for TRANSACTION")
		}
	}
	return nil
}
