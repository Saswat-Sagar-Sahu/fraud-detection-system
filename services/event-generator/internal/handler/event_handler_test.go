package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/model"
)

type testPublisher struct {
	calls int
	event model.Event
	err   error
}

func (p *testPublisher) PublishEvent(_ context.Context, event model.Event) error {
	p.calls++
	p.event = event
	return p.err
}

func (p *testPublisher) Close() error {
	return nil
}

func TestPostEventReturnsBadRequestOnBindingValidationError(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	publisher := &testPublisher{}
	eventHandler := NewEventHandler(publisher)

	router := gin.New()
	router.POST("/event", eventHandler.PostEvent)

	body := map[string]any{
		"event_type":  "LOGIN",
		"login_id":    "login-1",
		"occurred_at": "2026-02-26T10:00:00Z",
		"source_ip":   "192.168.0.10",
		"device_id":   "device-1",
		"metadata":    map[string]any{},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/event", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	if publisher.calls != 0 {
		t.Fatalf("publisher should not be called for invalid payload")
	}
}

func TestPostEventPublishesValidPayload(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	publisher := &testPublisher{}
	eventHandler := NewEventHandler(publisher)

	router := gin.New()
	router.POST("/event", eventHandler.PostEvent)

	body := map[string]any{
		"event_type":  "LOGIN",
		"user_id":     "user-1",
		"login_id":    "login-1",
		"occurred_at": "2026-02-26T10:00:00Z",
		"source_ip":   "192.168.0.10",
		"device_id":   "device-1",
		"metadata":    map[string]any{},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/event", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusAccepted, recorder.Code, recorder.Body.String())
	}
	if publisher.calls != 1 {
		t.Fatalf("expected publisher to be called once, got %d", publisher.calls)
	}
	if publisher.event.EventID == "" {
		t.Fatalf("expected generated event_id")
	}
}
