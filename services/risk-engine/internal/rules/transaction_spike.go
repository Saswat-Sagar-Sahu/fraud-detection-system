package rules

import "github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"

type TransactionSpikeRule struct {
	Score      int
	MinSamples int
	Multiplier float64
}

func NewTransactionSpikeRule(score, minSamples int, multiplier float64) TransactionSpikeRule {
	if score <= 0 {
		score = 40
	}
	if minSamples <= 0 {
		minSamples = 3
	}
	if multiplier <= 1 {
		multiplier = 3.0
	}
	return TransactionSpikeRule{Score: score, MinSamples: minSamples, Multiplier: multiplier}
}

func (r TransactionSpikeRule) Evaluate(event model.Event, state model.UserState) int {
	if event.EventType != model.EventTypeTransaction || event.Amount == nil || *event.Amount <= 0 {
		return 0
	}

	sum := 0.0
	count := 0
	for _, tx := range state.RecentTransactions {
		if tx.Amount == nil || *tx.Amount <= 0 {
			continue
		}
		sum += *tx.Amount
		count++
	}
	if count < r.MinSamples {
		return 0
	}

	average := sum / float64(count)
	if average <= 0 {
		return 0
	}

	if *event.Amount >= average*r.Multiplier {
		return r.Score
	}
	return 0
}
