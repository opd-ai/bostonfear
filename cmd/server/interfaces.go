package main

import "errors"

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
