package rules

import "github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"

type ImpossibleTravelRule struct {
	Score             int
	MaxSpeedKMPH      float64
	MinimumDistanceKM float64
}

func NewImpossibleTravelRule(score int, maxSpeedKMPH, minimumDistanceKM float64) ImpossibleTravelRule {
	if score <= 0 {
		score = 50
	}
	if maxSpeedKMPH <= 0 {
		maxSpeedKMPH = 900
	}
	if minimumDistanceKM <= 0 {
		minimumDistanceKM = 500
	}
	return ImpossibleTravelRule{
		Score:             score,
		MaxSpeedKMPH:      maxSpeedKMPH,
		MinimumDistanceKM: minimumDistanceKM,
	}
}

func (r ImpossibleTravelRule) Evaluate(event model.Event, state model.UserState) int {
	if event.EventType != model.EventTypeLogin {
		return 0
	}
	if event.Latitude == nil || event.Longitude == nil {
		return 0
	}
	if state.LastLoginLocation == nil || state.LastLoginLocation.Latitude == nil || state.LastLoginLocation.Longitude == nil {
		return 0
	}
	if !event.OccurredAt.After(state.LastLoginLocation.OccurredAt) {
		return 0
	}

	distanceKM := haversineKM(*event.Latitude, *event.Longitude, *state.LastLoginLocation.Latitude, *state.LastLoginLocation.Longitude)
	if distanceKM < r.MinimumDistanceKM {
		return 0
	}

	hours := event.OccurredAt.Sub(state.LastLoginLocation.OccurredAt).Hours()
	if hours <= 0 {
		return 0
	}

	speedKMPH := distanceKM / hours
	if speedKMPH >= r.MaxSpeedKMPH {
		return r.Score
	}
	return 0
}
