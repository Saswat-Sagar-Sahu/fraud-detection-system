package model

import "time"

type RiskAlert struct {
	AlertID       string         `json:"alert_id"`
	EventID       string         `json:"event_id"`
	EventType     EventType      `json:"event_type"`
	UserID        string         `json:"user_id"`
	RiskScore     int            `json:"risk_score"`
	Threshold     int            `json:"threshold"`
	RuleBreakdown map[string]int `json:"rule_breakdown"`
	OccurredAt    time.Time      `json:"occurred_at"`
	DetectedAt    time.Time      `json:"detected_at"`
}
