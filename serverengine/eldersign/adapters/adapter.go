package adapters

import (
	"time"
)

// elderSignBroadcastAdapter implements BroadcastPayloadAdapter for Elder Sign game events.
// This adapter owns the wire protocol shape for all broadcast messages in Elder Sign,
// allowing the core serverengine to remain family-agnostic while supporting Elder Sign's
// unique mechanics: 6-sided dice, adventure cards, dice locking, and museum exploration.
type elderSignBroadcastAdapter struct{}

// NewBroadcastAdapter creates a broadcast adapter for Elder Sign.
func NewBroadcastAdapter() BroadcastPayloadAdapter {
	return &elderSignBroadcastAdapter{}
}

// ShapeGameStatePayload transforms game state into wire format for gameState messages.
// Elder Sign includes: current player, doom counter, museum room locations,
// adventure card deck, dice tower state, investigator stamina/sanity, and Elder Sign tokens.
func (a *elderSignBroadcastAdapter) ShapeGameStatePayload(state interface{}) interface{} {
	// Elder Sign gameState shape: full state snapshot including museum-specific mechanics
	return map[string]interface{}{
		"type": "gameState",
		"data": state,
	}
}

// ShapeActionResultPayload transforms action results into wire format for gameUpdate messages.
// Elder Sign actions include: placeInvestigator, rollDice, lockDie, discardItem, claimAdventure.
// The payload includes the action type, result, resource delta, and timestamp.
func (a *elderSignBroadcastAdapter) ShapeActionResultPayload(
	action string,
	result string,
	resources interface{},
) interface{} {
	// Preserve existing wire compatibility by forwarding the full gameUpdate payload
	// when provided by serverengine.
	if resources != nil {
		return resources
	}

	// Fallback shape for defensive compatibility.
	return map[string]interface{}{
		"type":      "gameUpdate",
		"event":     action,
		"result":    result,
		"timestamp": time.Now(),
	}
}

// ShapeDiceResultPayload transforms dice outcomes into wire format for diceResult messages.
// Elder Sign uses 6-sided dice with colored results (red/green/yellow) plus special icons
// (Terror, Peril, Lore), distinct from Arkham Horror's 3-sided dice.
// The payload includes locked dice, active dice, terror count, and task completion status.
func (a *elderSignBroadcastAdapter) ShapeDiceResultPayload(diceResult interface{}) interface{} {
	// diceResult is expected to be an Elder Sign-specific dice result structure
	// We return it directly since the struct tags are already correct for wire format.
	return diceResult
}
