// Package messaging provides shared messaging primitives for cross-engine use.
// MessageType is the canonical enumeration of wire message types sent over WebSocket.
package messaging

// MessageType identifies the kind of broadcast message sent over the WebSocket wire.
type MessageType string

const (
	MessageGameState    MessageType = "gameState"        // Full game state snapshot
	MessagePlayerAction MessageType = "playerAction"     // A player's action request
	MessageGameUpdate   MessageType = "gameUpdate"       // Lightweight delta event
	MessageDiceResult   MessageType = "diceResult"       // Dice roll outcome
	MessageConnStatus   MessageType = "connectionStatus" // Connection lifecycle event
)

// IsValid reports whether m is one of the well-known message types.
func (m MessageType) IsValid() bool {
	switch m {
	case MessageGameState, MessagePlayerAction, MessageGameUpdate,
		MessageDiceResult, MessageConnStatus:
		return true
	}
	return false
}
