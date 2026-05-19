package adapters

import (
	"time"
)

// finalHourBroadcastAdapter implements BroadcastPayloadAdapter for Final Hour game events.
// This adapter owns the wire protocol shape for all broadcast messages in Final Hour,
// allowing the core serverengine to remain family-agnostic while supporting Final Hour's
// unique mechanics: real-time action programming, countdown tokens, priority resolution,
// and simultaneous action coordination.
type finalHourBroadcastAdapter struct{}

// NewBroadcastAdapter creates a broadcast adapter for Final Hour.
func NewBroadcastAdapter() BroadcastPayloadAdapter {
	return &finalHourBroadcastAdapter{}
}

// ShapeGameStatePayload transforms game state into wire format for gameState messages.
// Final Hour includes: countdown token value, priority track, active objectives,
// action planning buffer state, investigator focus tokens, and stress levels.
func (a *finalHourBroadcastAdapter) ShapeGameStatePayload(state interface{}) interface{} {
	// Final Hour gameState shape: full state snapshot including real-time mechanics
	return map[string]interface{}{
		"type": "gameState",
		"data": state,
	}
}

// ShapeActionResultPayload transforms action results into wire format for gameUpdate messages.
// Final Hour actions include: placeInvestigator, resolveAction, bidPriority, spendFocus.
// The payload includes the action type, priority resolution result, and timestamp.
func (a *finalHourBroadcastAdapter) ShapeActionResultPayload(
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

// ShapeDiceResultPayload transforms priority resolution outcomes into wire format.
// Final Hour uses priority bidding instead of dice rolls. This method adapts the
// interface to support priority-based conflict resolution.
func (a *finalHourBroadcastAdapter) ShapeDiceResultPayload(priorityResult interface{}) interface{} {
	// If the input is already a properly structured payload, return it directly
	if payload, ok := priorityResult.(*PriorityResultPayload); ok {
		return payload
	}
	// If it's a map, pass through for wire serialization
	if mapResult, ok := priorityResult.(map[string]interface{}); ok {
		return mapResult
	}
	// Fallback: return as-is for other types (custom structs with json tags)
	return priorityResult
}
