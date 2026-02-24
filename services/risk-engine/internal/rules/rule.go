package rules

import "github.com/saswatsagarsahu/fraud-detection-system/services/risk-engine/internal/model"

// Rule returns the score contribution for a single risk rule.
type Rule interface {
	Evaluate(event model.Event, state model.UserState) int
}

type NamedRule struct {
	Name string
	Rule Rule
}

type Evaluation struct {
	TotalScore int
	RuleScores map[string]int
}
