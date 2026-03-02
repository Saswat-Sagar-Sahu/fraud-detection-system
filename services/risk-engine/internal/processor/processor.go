package processor

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"
	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/rules"
	"github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/state"
)

type EventProcessor interface {
	Process(ctx context.Context, event model.Event, partition int, offset int64) error
}

type AlertPublisher interface {
	PublishRiskAlert(ctx context.Context, alert model.RiskAlert) error
}

type RiskProcessor struct {
	stateReader        state.Reader
	tracker            state.Tracker
	engine             *rules.Engine
	alertPublisher     AlertPublisher
	riskAlertThreshold int
	enableRiskAlerts   bool
}

func NewRiskProcessor(
	stateReader state.Reader,
	tracker state.Tracker,
	engine *rules.Engine,
	alertPublisher AlertPublisher,
	riskAlertThreshold int,
	enableRiskAlerts bool,
) *RiskProcessor {
	if engine == nil {
		engine = rules.DefaultEngine()
	}
	if riskAlertThreshold <= 0 {
		riskAlertThreshold = 50
	}
	return &RiskProcessor{
		stateReader:        stateReader,
		tracker:            tracker,
		engine:             engine,
		alertPublisher:     alertPublisher,
		riskAlertThreshold: riskAlertThreshold,
		enableRiskAlerts:   enableRiskAlerts,
	}
}

func (p *RiskProcessor) Process(ctx context.Context, event model.Event, partition int, offset int64) error {
	if event.EventID == "" || event.UserID == "" || event.EventType == "" {
		return fmt.Errorf("missing core fields: event_id/user_id/event_type")
	}

	var userState model.UserState
	if p.stateReader != nil {
		s, err := p.stateReader.GetUserState(ctx, event.UserID)
		if err != nil {
			return fmt.Errorf("load user state event_id=%s user_id=%s: %w", event.EventID, event.UserID, err)
		}
		userState = s
	}

	evaluation := p.engine.Evaluate(event, userState)

	if p.enableRiskAlerts && p.alertPublisher != nil && evaluation.TotalScore >= p.riskAlertThreshold {
		alert := model.RiskAlert{
			AlertID:       fmt.Sprintf("alert-%s", event.EventID),
			EventID:       event.EventID,
			EventType:     event.EventType,
			UserID:        event.UserID,
			RiskScore:     evaluation.TotalScore,
			Threshold:     p.riskAlertThreshold,
			RuleBreakdown: evaluation.RuleScores,
			OccurredAt:    event.OccurredAt.UTC(),
			DetectedAt:    time.Now().UTC(),
		}
		if err := p.alertPublisher.PublishRiskAlert(ctx, alert); err != nil {
			return fmt.Errorf("publish risk alert event_id=%s user_id=%s: %w", event.EventID, event.UserID, err)
		}
	} 

	if p.tracker != nil {
		if err := p.tracker.TrackEvent(ctx, event); err != nil {
			return fmt.Errorf("track redis state event_id=%s user_id=%s: %w", event.EventID, event.UserID, err)
		}
	}

	log.Printf(
		"processed event event_id=%s event_type=%s user_id=%s partition=%d offset=%d risk_score=%d risk_alert_raised=%t rule_breakdown=%s",
		event.EventID,
		event.EventType,
		event.UserID,
		partition,
		offset,
		evaluation.TotalScore,
		p.enableRiskAlerts && p.alertPublisher != nil && evaluation.TotalScore >= p.riskAlertThreshold,
		formatRuleBreakdown(evaluation.RuleScores),
	)
	return nil
}

func formatRuleBreakdown(ruleScores map[string]int) string {
	if len(ruleScores) == 0 {
		return "none"
	}

	keys := make([]string, 0, len(ruleScores))
	for k := range ruleScores {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, ruleScores[k]))
	}
	return strings.Join(parts, ",")
}
