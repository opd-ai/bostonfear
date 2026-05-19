package rules

// ActionType represents Final Hour action identifiers.
// Final Hour uses real-time simultaneous action programming where all players
// submit actions within a time window, then resolve them via priority bidding.
type ActionType string

const (
	// PlaceInvestigator repositions an investigator within the crisis location.
	// Unlike Arkham's multi-neighborhood movement or Elder Sign's museum rooms,
	// Final Hour uses a single location with internal room-based positioning.
	ActionPlaceInvestigator ActionType = "placeInvestigator"

	// ResolveAction executes a planned action during the resolution phase.
	// Actions are submitted simultaneously by all players during the planning phase,
	// then resolved in priority order. This mechanic does not exist in turn-based
	// Arkham Horror or Elder Sign.
	ActionResolveAction ActionType = "resolveAction"

	// BidPriority assigns priority value to break ties when multiple investigators
	// target the same space or objective. Higher priority acts first but consumes
	// more focus tokens. This is a Final Hour-specific conflict resolution mechanic.
	ActionBidPriority ActionType = "bidPriority"

	// SpendFocus converts focus tokens to gain action advantages.
	// Focus is the primary resource for priority bidding and action enhancement.
	// Similar to Arkham's Clues but used for action coordination rather than investigation.
	ActionSpendFocus ActionType = "spendFocus"
)

// ValidActions returns the set of all legal Final Hour actions.
func ValidActions() []ActionType {
	return []ActionType{
		ActionPlaceInvestigator,
		ActionResolveAction,
		ActionBidPriority,
		ActionSpendFocus,
	}
}

// String converts ActionType to string for logging and serialization.
func (a ActionType) String() string {
	return string(a)
}
