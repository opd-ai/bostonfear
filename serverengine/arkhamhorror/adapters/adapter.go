package adapters

import (
	"time"
)

// arkhamBroadcastAdapter implements BroadcastPayloadAdapter for Arkham Horror game events.
// This adapter owns the wire protocol shape for all broadcast messages in Arkham Horror,
// allowing the core serverengine to remain family-agnostic.
type arkhamBroadcastAdapter struct{}

// NewBroadcastAdapter creates a broadcast adapter for Arkham Horror.
func NewBroadcastAdapter() BroadcastPayloadAdapter {
	return &arkhamBroadcastAdapter{}
}

// ShapeGameStatePayload transforms game state into wire format for gameState messages.
// The payload includes current player, doom counter, player resources, and locations.
func (a *arkhamBroadcastAdapter) ShapeGameStatePayload(state interface{}) interface{} {
	// Arkham Horror gameState shape: full state snapshot
	// This is passed directly by the serverengine caller.
	return map[string]interface{}{
		"type": "gameState",
		"data": state,
	}
}

// ShapeActionResultPayload transforms action results into wire format for gameUpdate messages.
// The payload includes the action type, result, resource delta, and timestamp.
func (a *arkhamBroadcastAdapter) ShapeActionResultPayload(
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
// The payload includes successes, tentacles, doom increment, and overall success/failure.
func (a *arkhamBroadcastAdapter) ShapeDiceResultPayload(diceResult interface{}) interface{} {
	// diceResult is expected to be a *serverengine.DiceResultMessage
	// We return it directly since the struct tags are already correct for wire format.
	return diceResult
}
