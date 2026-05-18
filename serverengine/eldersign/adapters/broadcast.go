// Package adapters defines broadcast payload shaping for Elder Sign.
package adapters

import "github.com/opd-ai/bostonfear/serverengine/common/contracts"

// BroadcastPayloadAdapter is the canonical interface from serverengine/common/contracts.
// Re-exported here for use within the adapters package without requiring callers of
// NewBroadcastAdapter to import contracts directly.
type BroadcastPayloadAdapter = contracts.BroadcastPayloadAdapter

// DiceResultPayload is the Elder Sign-owned shape for dice outcomes.
// Elder Sign uses 6-sided dice with colored results (red/green/yellow)
// plus special icons (Terror, Peril, Lore).
type DiceResultPayload struct {
	Type          string        `json:"type"`
	PlayerID      string        `json:"playerId"`
	Action        string        `json:"action"`
	LockedResults []interface{} `json:"lockedResults"`
	ActiveResults []interface{} `json:"activeResults"`
	TerrorCount   int           `json:"terrorCount"`
	Success       bool          `json:"success"`
	DoomIncrease  int           `json:"doomIncrease"`
}

// AdventureResultPayload is the Elder Sign-owned shape for adventure completion.
type AdventureResultPayload struct {
	Type              string      `json:"type"`
	PlayerID          string      `json:"playerId"`
	AdventureID       string      `json:"adventureId"`
	Completed         bool        `json:"completed"`
	ElderSignsAwarded int         `json:"elderSignsAwarded"`
	Reward            interface{} `json:"reward"`
	Timestamp         interface{} `json:"timestamp"`
}

// NewDiceResultPayload creates a properly formatted Elder Sign dice result payload
// from the dice mechanics state. This is the canonical wire format for Elder Sign
// dice outcomes, including locked/active dice, terror counts, and success status.
func NewDiceResultPayload(playerID, action string, lockedResults, activeResults []interface{}, terrorCount int, success bool, doomIncrease int) *DiceResultPayload {
	return &DiceResultPayload{
		Type:          "diceResult",
		PlayerID:      playerID,
		Action:        action,
		LockedResults: lockedResults,
		ActiveResults: activeResults,
		TerrorCount:   terrorCount,
		Success:       success,
		DoomIncrease:  doomIncrease,
	}
}
