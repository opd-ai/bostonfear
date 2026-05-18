package adapters

// ConnectionStatusPayload creates a minimal wire payload for Elder Sign clients.
func ConnectionStatusPayload(playerID, status string) map[string]interface{} {
	return map[string]interface{}{
		"type":     "connectionStatus",
		"playerId": playerID,
		"status":   status,
	}
}

// GameUpdatePayload creates a minimal event payload for Elder Sign updates.
func GameUpdatePayload(playerID, event, result string) map[string]interface{} {
	return map[string]interface{}{
		"type":     "gameUpdate",
		"playerId": playerID,
		"event":    event,
		"result":   result,
	}
}
