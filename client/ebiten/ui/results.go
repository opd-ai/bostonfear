package ui

import "fmt"

// ActionOutcome represents the result of a player action.
type ActionOutcome struct {
	PlayerID       string
	ActionType     string
	ActionTarget   string
	Successful     bool
	Description    string
	ResourceDelta  ResourceDelta // Reusing from feedback.go
	LocationChange LocationChange
	DiceRoll       *DiceRollResult
}

// DiceRollResult shows what was rolled and the outcome.
type DiceRollResult struct {
	Dice        []string // e.g., ["success", "blank", "tentacle"]
	Required    int      // Required successes for this action.
	Achieved    int      // Actual successes rolled.
	Passed      bool     // Whether the roll met requirements.
	FailureText string   // Reason for failure.
}

// ResultsPanel displays action outcomes with visual feedback.
type ResultsPanel struct {
	currentOutcome *ActionOutcome
	history        []*ActionOutcome
	maxHistory     int
	displayTime    int64 // Milliseconds to keep outcome visible.
	displayStart   int64 // Tick when outcome was displayed.
	isVisible      bool
}

// NewResultsPanel creates a panel for action outcome display.
func NewResultsPanel() *ResultsPanel {
	return &ResultsPanel{
		currentOutcome: nil,
		history:        make([]*ActionOutcome, 0),
		maxHistory:     20,
		displayTime:    3000, // 3 seconds.
		isVisible:      false,
	}
}

// DisplayOutcome shows an action result on the panel.
func (rp *ResultsPanel) DisplayOutcome(outcome *ActionOutcome) {
	if rp == nil || outcome == nil {
		return
	}
	rp.currentOutcome = outcome
	rp.history = append(rp.history, outcome)
	if len(rp.history) > rp.maxHistory {
		rp.history = rp.history[1:]
	}
	rp.isVisible = true
	rp.displayStart = 0 // Set by caller's time source.
}

// IsVisible reports whether an outcome is currently displayed.
func (rp *ResultsPanel) IsVisible() bool {
	return rp != nil && rp.isVisible
}

// CurrentOutcome returns the active outcome, or nil if none.
func (rp *ResultsPanel) CurrentOutcome() *ActionOutcome {
	if rp == nil {
		return nil
	}
	return rp.currentOutcome
}

// History returns all past outcomes.
func (rp *ResultsPanel) History() []*ActionOutcome {
	if rp == nil {
		return nil
	}
	copy := make([]*ActionOutcome, len(rp.history))
	for i, o := range rp.history {
		copy[i] = o
	}
	return copy
}

// Clear removes the current outcome from display.
func (rp *ResultsPanel) Clear() {
	if rp != nil {
		rp.currentOutcome = nil
		rp.isVisible = false
	}
}

// OutcomeText formats a human-readable outcome message.
func (rp *ResultsPanel) OutcomeText() string {
	if rp == nil || rp.currentOutcome == nil {
		return ""
	}
	o := rp.currentOutcome

	// Build base message.
	msg := o.Description
	if msg == "" {
		msg = fmt.Sprintf("%s: %s", o.ActionType, o.ActionTarget)
	}

	// Add result indicator.
	if o.Successful {
		msg += " ✓"
	} else {
		msg += " ✗"
	}

	return msg
}

// ResourceDeltaText formats resource changes.
func (rp *ResultsPanel) ResourceDeltaText() string {
	if rp == nil || rp.currentOutcome == nil {
		return ""
	}
	d := rp.currentOutcome.ResourceDelta
	if d.HealthDelta == 0 && d.SanityDelta == 0 && d.ClueDelta == 0 && d.DoomDelta == 0 {
		return "No resource changes"
	}

	var msg string
	if d.HealthDelta != 0 {
		sign := "+"
		if d.HealthDelta < 0 {
			sign = ""
		}
		msg += fmt.Sprintf("Health: %s%d ", sign, d.HealthDelta)
	}
	if d.SanityDelta != 0 {
		sign := "+"
		if d.SanityDelta < 0 {
			sign = ""
		}
		msg += fmt.Sprintf("Sanity: %s%d ", sign, d.SanityDelta)
	}
	if d.ClueDelta != 0 {
		sign := "+"
		if d.ClueDelta < 0 {
			sign = ""
		}
		msg += fmt.Sprintf("Clues: %s%d ", sign, d.ClueDelta)
	}

	return msg
}

// DoomChangeText formats doom counter changes.
func (rp *ResultsPanel) DoomChangeText() string {
	if rp == nil || rp.currentOutcome == nil {
		return ""
	}
	d := rp.currentOutcome.ResourceDelta
	if d.DoomDelta == 0 {
		return "No doom change"
	}
	// DoomDelta is just the change amount, not from/to. Format accordingly.
	sign := "+"
	if d.DoomDelta < 0 {
		sign = ""
	}
	return fmt.Sprintf("Doom: %s%d", sign, d.DoomDelta)
}

// DiceText formats dice roll results.
func (rp *ResultsPanel) DiceText() string {
	if rp == nil || rp.currentOutcome == nil || rp.currentOutcome.DiceRoll == nil {
		return "No dice roll"
	}
	r := rp.currentOutcome.DiceRoll

	result := fmt.Sprintf("Rolled: %v ", r.Dice)
	result += fmt.Sprintf("(%d/%d successes) ", r.Achieved, r.Required)

	if r.Passed {
		result += "✓ Success"
	} else {
		result += "✗ Failed"
		if r.FailureText != "" {
			result += ": " + r.FailureText
		}
	}

	return result
}

// SetDisplayDuration updates how long outcomes stay visible (in milliseconds).
func (rp *ResultsPanel) SetDisplayDuration(ms int64) {
	if rp != nil {
		rp.displayTime = ms
	}
}
