package rules

import "github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"

type NewDeviceRule struct {
	Score int
}

func NewNewDeviceRule(score int) NewDeviceRule {
	if score <= 0 {
		score = 20
	}
	return NewDeviceRule{Score: score}
}

func (r NewDeviceRule) Evaluate(event model.Event, state model.UserState) int {
	if event.EventType != model.EventTypeLogin {
		return 0
	}
	if event.DeviceID == "" || state.LastLoginLocation == nil || state.LastLoginLocation.DeviceID == "" {
		return 0
	}
	if event.DeviceID != state.LastLoginLocation.DeviceID {
		return r.Score
	}
	return 0
}
