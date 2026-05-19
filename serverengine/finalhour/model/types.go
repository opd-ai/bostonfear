package model

// Action is a Final Hour family action identifier.
type Action string

const (
	ActionReposition Action = "reposition"
	ActionContain    Action = "contain"
	ActionCoordinate Action = "coordinate"
)

// InvestigatorState is the Final Hour-specific player state envelope.
type InvestigatorState struct {
	PlayerID         string
	Health           int
	Stress           int
	FocusTokens      int
	ActionsRemaining int
	PriorityBid      int
}

// IsZero reports whether the state has not been initialized.
func (s InvestigatorState) IsZero() bool {
	return s.PlayerID == ""
}

// ObjectiveCard represents a time-sensitive goal in Final Hour.
type ObjectiveCard struct {
	ID          string
	Name        string
	Description string
	Deadline    int // Countdown value when this objective expires
	Completed   bool
}

// FinalHourGameState extends base game state with Final Hour-specific mechanics.
type FinalHourGameState struct {
	CountdownValue     int
	ActiveObjectives   []ObjectiveCard
	PriorityTrack      map[string]int // PlayerID -> current priority bid
	ActionPlanningOpen bool           // True during planning phase, false during resolution
}

// IsZero reports whether the game state has not been initialized.
func (s FinalHourGameState) IsZero() bool {
	return s.CountdownValue == 0 && len(s.ActiveObjectives) == 0
}
