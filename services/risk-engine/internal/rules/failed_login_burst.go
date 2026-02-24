package rules

import "github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"

type FailedLoginBurstRule struct {
	Threshold int64
	Score     int
}

func NewFailedLoginBurstRule(threshold int64, score int) FailedLoginBurstRule {
	if threshold <= 0 {
		threshold = 3
	}
	if score <= 0 {
		score = 30
	}
	return FailedLoginBurstRule{Threshold: threshold, Score: score}
}

func (r FailedLoginBurstRule) Evaluate(event model.Event, state model.UserState) int {
	if event.EventType != model.EventTypeLogin {
		return 0
	}

	loginSuccess, ok := parseMetadataBool(event.Metadata, "login_success")
	if !ok || loginSuccess {
		return 0
	}

	if state.FailedLoginCount+1 >= r.Threshold {
		return r.Score
	}
	return 0
}
