package adapters

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
