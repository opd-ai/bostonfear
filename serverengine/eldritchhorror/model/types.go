package model

// Action is an Eldritch Horror family action identifier.
type Action string

const (
	ActionTravel   Action = "travel"
	ActionPrepare  Action = "prepare"
	ActionResearch Action = "research"
)

// InvestigatorState is the Eldritch Horror-specific player state envelope.
type InvestigatorState struct {
	PlayerID         string
	Health           int
	Sanity           int
	Focus            int
	ActionsRemaining int
}

// IsZero reports whether the state has not been initialized.
func (s InvestigatorState) IsZero() bool {
	return s.PlayerID == ""
}
