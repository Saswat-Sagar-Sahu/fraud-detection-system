package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"
)

type RiskAlertPublisher struct {
	writer *kafkago.Writer
	topic  string
}

func NewRiskAlertPublisher(brokers []string, topic string) *RiskAlertPublisher {
	return &RiskAlertPublisher{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafkago.LeastBytes{},
			RequiredAcks: kafkago.RequireOne,
		},
		topic: topic,
	}
}

func (p *RiskAlertPublisher) PublishRiskAlert(ctx context.Context, alert model.RiskAlert) error {
	payload, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("marshal risk alert alert_id=%s: %w", alert.AlertID, err)
	}

	if err := p.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(alert.UserID),
		Value: payload,
	}); err != nil {
		return fmt.Errorf("publish risk alert topic=%s alert_id=%s event_id=%s: %w", p.topic, alert.AlertID, alert.EventID, err)
	}
	return nil
}

func (p *RiskAlertPublisher) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("close risk alert publisher topic=%s: %w", p.topic, err)
	}
	return nil
}
