package rules

import "github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"

type Engine struct {
	rules []NamedRule
}

func NewEngine(rules []NamedRule) *Engine {
	cloned := make([]NamedRule, 0, len(rules))
	for _, r := range rules {
		if r.Rule == nil || r.Name == "" {
			continue
		}
		cloned = append(cloned, r)
	}
	return &Engine{rules: cloned}
}

func DefaultEngine() *Engine {
	return NewEngine([]NamedRule{
		{Name: "failed_login_burst", Rule: NewFailedLoginBurstRule(3, 30)},
		{Name: "new_device", Rule: NewNewDeviceRule(20)},
		{Name: "impossible_travel", Rule: NewImpossibleTravelRule(50, 900, 500)},
		{Name: "transaction_spike", Rule: NewTransactionSpikeRule(40, 3, 3.0)},
	})
}

func (e *Engine) Evaluate(event model.Event, state model.UserState) Evaluation {
	result := Evaluation{RuleScores: make(map[string]int)}
	if e == nil {
		return result
	}

	for _, namedRule := range e.rules {
		score := namedRule.Rule.Evaluate(event, state)
		if score <= 0 {
			continue
		}
		result.TotalScore += score
		result.RuleScores[namedRule.Name] = score
	}
	return result
}
