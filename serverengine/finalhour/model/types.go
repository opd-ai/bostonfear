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
	ActionsRemaining int
}

// IsZero reports whether the state has not been initialized.
func (s InvestigatorState) IsZero() bool {
	return s.PlayerID == ""
}
