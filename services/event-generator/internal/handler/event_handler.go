package handler

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/id"
	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/kafka"
	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/model"
)

type EventHandler struct {
	publisher kafka.EventPublisher
}

type eventRequest struct {
	EventID    string                 `json:"event_id,omitempty"`
	EventType  model.EventType        `json:"event_type" binding:"required,oneof=LOGIN TRANSACTION DEVICE_CHANGE PASSWORD_RESET"`
	UserID     string                 `json:"user_id" binding:"required"`
	LoginID    string                 `json:"login_id,omitempty" binding:"required_if=EventType LOGIN,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	OccurredAt time.Time              `json:"occurred_at" binding:"required"`
	SourceIP   string                 `json:"source_ip" binding:"required,ip"`
	DeviceID   string                 `json:"device_id" binding:"required"`
	Country    string                 `json:"country,omitempty"`
	Latitude   *float64               `json:"latitude,omitempty" binding:"required_with=Longitude,omitempty,gte=-90,lte=90"`
	Longitude  *float64               `json:"longitude,omitempty" binding:"required_with=Latitude,omitempty,gte=-180,lte=180"`
	Amount     *float64               `json:"amount,omitempty" binding:"required_if=EventType TRANSACTION,omitempty,gte=0"`
	Currency   string                 `json:"currency,omitempty" binding:"required_if=EventType TRANSACTION,omitempty,len=3,uppercase"`
	Metadata   map[string]interface{} `json:"metadata" binding:"required"`
}

func NewEventHandler(publisher kafka.EventPublisher) *EventHandler {
	return &EventHandler{publisher: publisher}
}

func (h *EventHandler) PostEvent(c *gin.Context) {
	var req eventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("event bind failed remote_addr=%s err=%v", c.ClientIP(), err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(req.EventID) != "" {
		log.Printf(
			"client-supplied event_id ignored provided_event_id=%s event_type=%s user_id=%s remote_addr=%s",
			req.EventID,
			req.EventType,
			req.UserID,
			c.ClientIP(),
		)
	}

	event := model.Event{
		EventType:  req.EventType,
		UserID:     req.UserID,
		LoginID:    req.LoginID,
		SessionID:  req.SessionID,
		OccurredAt: req.OccurredAt,
		SourceIP:   req.SourceIP,
		DeviceID:   req.DeviceID,
		Country:    req.Country,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		Amount:     req.Amount,
		Currency:   req.Currency,
		Metadata:   req.Metadata,
	}

	generatedEventID, err := id.NewUUID()
	if err != nil {
		log.Printf("event id generation failed event_type=%s user_id=%s remote_addr=%s err=%v", event.EventType, event.UserID, c.ClientIP(), err)
		c.JSON(500, gin.H{"error": "failed to generate event id"})
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
			c.ClientIP(),
			err,
		)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	if err := h.publisher.PublishEvent(ctx, event); err != nil {
		log.Printf(
			"event publish failed event_id=%s event_type=%s user_id=%s login_id=%s remote_addr=%s err=%v",
			event.EventID,
			event.EventType,
			event.UserID,
			event.LoginID,
			c.ClientIP(),
			err,
		)
		c.JSON(500, gin.H{"error": "failed to publish event"})
		return
	}

	log.Printf(
		"event published event_id=%s event_type=%s user_id=%s login_id=%s",
		event.EventID,
		event.EventType,
		event.UserID,
		event.LoginID,
	)
	c.JSON(202, gin.H{"status": "published", "event_id": event.EventID})
}

func Healthz(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
