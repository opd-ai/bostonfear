package adapters

import "github.com/opd-ai/bostonfear/serverengine/common/contracts"

// BroadcastPayloadAdapter is the canonical interface from serverengine/common/contracts.
// Re-exported here for use within the adapters package without requiring callers of
// NewBroadcastAdapter to import contracts directly.
type BroadcastPayloadAdapter = contracts.BroadcastPayloadAdapter

// ConnectionStatusPayload creates a minimal wire payload for Final Hour clients.
func ConnectionStatusPayload(playerID, status string) map[string]interface{} {
	return map[string]interface{}{
		"type":     "connectionStatus",
		"playerId": playerID,
		"status":   status,
	}
}

// GameUpdatePayload creates a minimal event payload for Final Hour updates.
func GameUpdatePayload(playerID, event, result string) map[string]interface{} {
	return map[string]interface{}{
		"type":     "gameUpdate",
		"playerId": playerID,
		"event":    event,
		"result":   result,
	}
}

// PriorityResultPayload is the Final Hour-owned shape for priority resolution outcomes.
type PriorityResultPayload struct {
	Type         string      `json:"type"`
	PlayerID     string      `json:"playerId"`
	Action       string      `json:"action"`
	PriorityBid  int         `json:"priorityBid"`
	Success      bool        `json:"success"`
	CountdownDec int         `json:"countdownDec"`
	Timestamp    interface{} `json:"timestamp"`
}

// ObjectiveResultPayload is the Final Hour-owned shape for objective completion.
type ObjectiveResultPayload struct {
	Type        string      `json:"type"`
	ObjectiveID string      `json:"objectiveId"`
	Completed   bool        `json:"completed"`
	Reward      interface{} `json:"reward"`
	Timestamp   interface{} `json:"timestamp"`
}
