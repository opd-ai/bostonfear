package rules

import "fmt"

// TurnBudget defines how many actions an investigator can spend each turn.
type TurnBudget struct {
	MaxActions int
}

// DefaultTurnBudget is used by Eldritch Horror scaffolding until full rules are added.
var DefaultTurnBudget = TurnBudget{MaxActions: 2}

// ValidateActionCount validates action usage against the configured budget.
func (b TurnBudget) ValidateActionCount(actionsTaken int) error {
	if actionsTaken < 0 {
		return fmt.Errorf("actions taken must be non-negative: %d", actionsTaken)
	}
	if actionsTaken > b.MaxActions {
		return fmt.Errorf("actions taken %d exceeds max %d", actionsTaken, b.MaxActions)
	}
	return nil
}
