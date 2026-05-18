package rules

// ActionType represents Elder Sign action identifiers.
// Elder Sign uses a different action set than Arkham Horror:
// investigators place themselves at adventure locations, roll dice,
// lock favorable results, and claim completed adventures.
type ActionType string

const (
	// PlaceInvestigator moves an investigator to an adventure card location.
	// Unlike Arkham's Move action, there are no adjacency restrictions -
	// all museum rooms are accessible from any position.
	ActionPlaceInvestigator ActionType = "placeInvestigator"

	// RollDice initiates dice rolling for the current adventure task.
	// Uses 6-sided Elder Sign dice with colored results (red/green/yellow)
	// plus special icons (Terror, Peril, Lore), distinct from Arkham's
	// 3-sided Success/Blank/Tentacle dice.
	ActionRollDice ActionType = "rollDice"

	// LockDie secures a favorable die result to the dice pool.
	// Locked dice cannot be rerolled and persist for subsequent task attempts.
	// This mechanic does not exist in Arkham Horror.
	ActionLockDie ActionType = "lockDie"

	// DiscardItem sacrifices an item card to gain its benefit.
	// Items may provide dice results, resource restoration, or special effects.
	ActionDiscardItem ActionType = "discardItem"

	// ClaimAdventure completes an adventure card when all tasks are resolved.
	// Rewards include Elder Sign tokens, trophies, items, or special abilities.
	// This is the primary win condition mechanic for Elder Sign.
	ActionClaimAdventure ActionType = "claimAdventure"
)

// ValidActions returns the set of all legal Elder Sign actions.
func ValidActions() []ActionType {
	return []ActionType{
		ActionPlaceInvestigator,
		ActionRollDice,
		ActionLockDie,
		ActionDiscardItem,
		ActionClaimAdventure,
	}
}

// String converts ActionType to string for logging and serialization.
func (a ActionType) String() string {
	return string(a)
}
