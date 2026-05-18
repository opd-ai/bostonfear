package rules

// ActionType represents Eldritch Horror action identifiers.
// Eldritch Horror uses a distinct action set focused on global travel,
// resource management, and cooperative mystery solving across continents.
type ActionType string

const (
	// ActionTravel moves an investigator between cities on the global map.
	// Travel requires spending action points and may consume tickets (train/ship).
	// Routes between cities have different costs and travel times.
	// This differs from Arkham's neighborhood adjacency movement.
	ActionTravel ActionType = "travel"

	// ActionLocalAction performs an action specific to the current city.
	// Local actions vary by city and include: trading, gathering supplies,
	// recruiting allies, or preparing expeditions.
	// Each city offers a unique set of local actions.
	ActionLocalAction ActionType = "localAction"

	// ActionComponentAction interacts with a game component card.
	// Components include: Assets (items, allies, conditions), Spells,
	// Gate encounters, Monster combat, Mythos effects.
	// This is the primary action for resolving encounters and using resources.
	ActionComponentAction ActionType = "componentAction"

	// ActionRestAction allows an investigator to recover Health and Sanity.
	// Resting costs 1 action and restores a portion of resources based on
	// current location and available facilities (hospitals, lodges).
	// Recovery amounts vary by city infrastructure.
	ActionRestAction ActionType = "restAction"

	// ActionTradeAction exchanges assets with another investigator.
	// Trading is only possible when investigators are in the same city.
	// Common items (clues, tickets, money) and unique assets can be traded.
	// Some assets are non-tradeable (bound items, personal conditions).
	ActionTradeAction ActionType = "tradeAction"

	// ActionPrepareExpedition initiates a multi-turn expedition encounter.
	// Expeditions are region-specific (Americas, Europe, Asia, etc.) and
	// require multiple successful tests to complete.
	// Rewards include clues, relics, and mystery progress.
	// This mechanic is unique to Eldritch Horror and does not appear in
	// Arkham Horror or Elder Sign.
	ActionPrepareExpedition ActionType = "prepareExpedition"
)

// ValidActions returns the set of all legal Eldritch Horror actions.
func ValidActions() []ActionType {
	return []ActionType{
		ActionTravel,
		ActionLocalAction,
		ActionComponentAction,
		ActionRestAction,
		ActionTradeAction,
		ActionPrepareExpedition,
	}
}

// String converts ActionType to string for logging and serialization.
func (a ActionType) String() string {
	return string(a)
}

// ActionCost returns the number of actions required to perform this action.
// Most actions cost 1 action point, but some may vary based on game state.
func (a ActionType) ActionCost() int {
	switch a {
	case ActionTravel, ActionLocalAction, ActionComponentAction,
		ActionRestAction, ActionTradeAction, ActionPrepareExpedition:
		return 1
	default:
		return 1 // Default cost for unknown actions
	}
}
