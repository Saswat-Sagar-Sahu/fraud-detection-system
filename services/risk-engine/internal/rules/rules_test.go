package rules

import (
	"testing"
	"time"

	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"
)

func TestFailedLoginBurstRule(t *testing.T) {
	rule := NewFailedLoginBurstRule(3, 30)
	event := model.Event{
		EventType: model.EventTypeLogin,
		Metadata:  map[string]interface{}{"login_success": false},
	}
	state := model.UserState{FailedLoginCount: 2}

	if got := rule.Evaluate(event, state); got != 30 {
		t.Fatalf("expected score 30, got %d", got)
	}
}

func TestNewDeviceRule(t *testing.T) {
	rule := NewNewDeviceRule(20)
	event := model.Event{EventType: model.EventTypeLogin, DeviceID: "device-new"}
	state := model.UserState{LastLoginLocation: &model.LoginLocation{DeviceID: "device-old"}}

	if got := rule.Evaluate(event, state); got != 20 {
		t.Fatalf("expected score 20, got %d", got)
	}
}

func TestImpossibleTravelRule(t *testing.T) {
	rule := NewImpossibleTravelRule(50, 900, 500)
	prevLat := 37.7749
	prevLon := -122.4194
	curLat := 40.7128
	curLon := -74.0060

	event := model.Event{
		EventType:  model.EventTypeLogin,
		OccurredAt: time.Date(2026, 2, 25, 14, 0, 0, 0, time.UTC),
		Latitude:   &curLat,
		Longitude:  &curLon,
	}
	state := model.UserState{
		LastLoginLocation: &model.LoginLocation{
			OccurredAt: time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC),
			Latitude:   &prevLat,
			Longitude:  &prevLon,
		},
	}

	if got := rule.Evaluate(event, state); got != 50 {
		t.Fatalf("expected score 50, got %d", got)
	}
}

func TestTransactionSpikeRule(t *testing.T) {
	rule := NewTransactionSpikeRule(40, 3, 3.0)
	a1 := 100.0
	a2 := 120.0
	a3 := 80.0
	curr := 360.0

	event := model.Event{EventType: model.EventTypeTransaction, Amount: &curr}
	state := model.UserState{RecentTransactions: []model.TransactionRecord{
		{Amount: &a1},
		{Amount: &a2},
		{Amount: &a3},
	}}

	if got := rule.Evaluate(event, state); got != 40 {
		t.Fatalf("expected score 40, got %d", got)
	}
}

func TestEngineEvaluate(t *testing.T) {
	engine := DefaultEngine()
	event := model.Event{EventType: model.EventTypeLogin, DeviceID: "new-device", Metadata: map[string]interface{}{"login_success": false}}
	state := model.UserState{FailedLoginCount: 2, LastLoginLocation: &model.LoginLocation{DeviceID: "old-device"}}

	eval := engine.Evaluate(event, state)
	if eval.TotalScore != 50 {
		t.Fatalf("expected total score 50, got %d", eval.TotalScore)
	}
}
