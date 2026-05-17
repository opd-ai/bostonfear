package serverengine

import (
	"errors"

	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
)

// Broadcaster sends a JSON payload to all connected clients.
// Implementations must be safe for concurrent use.
type Broadcaster interface {
	Broadcast(payload []byte) error
}

// StateValidator checks the game state for invariant violations and supports recovery.
// Implementations must be safe for concurrent use.
type StateValidator interface {
	ValidateGameState(gs *GameState) []ValidationError
	RecoverGameState(gs *GameState, errors []ValidationError) (*GameState, error)
	IsGameStateHealthy(gs *GameState) bool
	GetCorruptionHistory() []CorruptionEvent
}

// BroadcastPayloadAdapter is an alias for the canonical definition in
// serverengine/common/contracts. Both serverengine and game-family adapter packages
// (e.g. serverengine/arkhamhorror/adapters) reference this same interface to avoid
// maintaining duplicate signatures.
type BroadcastPayloadAdapter = contracts.BroadcastPayloadAdapter

// errBroadcastFull is returned by Broadcast when the channel is full.
var errBroadcastFull = errors.New("broadcast channel full: payload dropped")

// channelBroadcaster enqueues payloads on a buffered channel consumed by
// broadcastHandler, decoupling action processing from WebSocket I/O.
type channelBroadcaster struct {
	ch chan []byte
}

// Broadcast enqueues payload for delivery to all connected clients.
// Returns errBroadcastFull if the channel is full (payload is dropped).
func (b *channelBroadcaster) Broadcast(payload []byte) error {
	select {
	case b.ch <- payload:
		return nil
	default:
		return errBroadcastFull
	}
}
