package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/id"
	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/kafka"
	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/model"
)

type EventHandler struct {
	publisher kafka.EventPublisher
}

func NewEventHandler(publisher kafka.EventPublisher) *EventHandler {
	return &EventHandler{publisher: publisher}
}

func (h *EventHandler) PostEvent(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var event model.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		log.Printf("event decode failed remote_addr=%s err=%v", r.RemoteAddr, err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
		return
	}
	if strings.TrimSpace(event.EventID) != "" {
		log.Printf(
			"client-supplied event_id ignored provided_event_id=%s event_type=%s user_id=%s remote_addr=%s",
			event.EventID,
			event.EventType,
			event.UserID,
			r.RemoteAddr,
		)
	}
	generatedEventID, err := id.NewUUID()
	if err != nil {
		log.Printf("event id generation failed event_type=%s user_id=%s remote_addr=%s err=%v", event.EventType, event.UserID, r.RemoteAddr, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate event id"})
		return
	}
	event.EventID = generatedEventID

	if err := event.Validate(); err != nil {
		log.Printf(
			"event validation failed event_id=%s event_type=%s user_id=%s login_id=%s remote_addr=%s err=%v",
			event.EventID,
			event.EventType,
			event.UserID,
			event.LoginID,
			r.RemoteAddr,
			err,
		)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.publisher.PublishEvent(ctx, event); err != nil {
		log.Printf(
			"event publish failed event_id=%s event_type=%s user_id=%s login_id=%s remote_addr=%s err=%v",
			event.EventID,
			event.EventType,
			event.UserID,
			event.LoginID,
			r.RemoteAddr,
			err,
		)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to publish event"})
		return
	}

	log.Printf(
		"event published event_id=%s event_type=%s user_id=%s login_id=%s",
		event.EventID,
		event.EventType,
		event.UserID,
		event.LoginID,
	)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "published", "event_id": event.EventID})
}

func Healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
