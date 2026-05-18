package model

// Action is an Elder Sign family action identifier.
type Action string

const (
	ActionAcquireItem Action = "acquireItem"
	ActionResolveTask Action = "resolveTask"
	ActionUseSpell    Action = "useSpell"
)

// InvestigatorState is the Elder Sign-specific player state envelope.
type InvestigatorState struct {
	PlayerID         string
	Stamina          int
	Sanity           int
	ElderSigns       int
	ActionsRemaining int
}

// IsZero reports whether the state has not been initialized.
func (s InvestigatorState) IsZero() bool {
	return s.PlayerID == ""
}
