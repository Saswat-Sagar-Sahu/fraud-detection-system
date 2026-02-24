package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/saswatsagarsahu/fraud-detection-system/services/event-generator/internal/model"
)

type EventPublisher interface {
	PublishEvent(ctx context.Context, event model.Event) error
	Close() error
}

type Publisher struct {
	writer *kafkago.Writer
	topic  string
}

func NewPublisher(brokers []string, topic string) *Publisher {
	return &Publisher{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafkago.LeastBytes{},
			RequiredAcks: kafkago.RequireOne,
		},
		topic: topic,
	}
}

func (p *Publisher) PublishEvent(ctx context.Context, event model.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event failed event_id=%s event_type=%s user_id=%s: %w", event.EventID, event.EventType, event.UserID, err)
	}

	if err := p.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(event.UserID),
		Value: payload,
	}); err != nil {
		return fmt.Errorf(
			"kafka write failed topic=%s event_id=%s event_type=%s user_id=%s: %w",
			p.topic,
			event.EventID,
			event.EventType,
			event.UserID,
			err,
		)
	}

	return nil
}

func (p *Publisher) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("kafka writer close failed topic=%s: %w", p.topic, err)
	}
	return nil
}
