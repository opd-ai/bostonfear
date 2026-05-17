// Package messaging provides shared messaging primitives for cross-engine use.
// MessageType is the canonical enumeration of wire message types sent over WebSocket.
package messaging

import (
	"encoding/json"
	"errors"
	"reflect"
)

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

// MessageCodec defines shared encode/decode boundaries for wire payloads.
type MessageCodec interface {
	Encode(value any) ([]byte, error)
	Decode(data []byte, target any) error
}

// ErrDecodeTargetRequired indicates the decode target was nil or not a pointer.
var ErrDecodeTargetRequired = errors.New("decode target must be a non-nil pointer")

// JSONCodec implements MessageCodec with encoding/json.
type JSONCodec struct{}

// Encode serializes a value into JSON bytes.
func (JSONCodec) Encode(value any) ([]byte, error) {
	return json.Marshal(value)
}

// Decode deserializes JSON bytes into target.
func (JSONCodec) Decode(data []byte, target any) error {
	if target == nil {
		return ErrDecodeTargetRequired
	}
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return ErrDecodeTargetRequired
	}
	return json.Unmarshal(data, target)
}

var defaultCodec MessageCodec = JSONCodec{}

// EncodeJSON serializes a value via the shared default codec.
func EncodeJSON(value any) ([]byte, error) {
	return defaultCodec.Encode(value)
}

// DecodeJSON deserializes JSON bytes via the shared default codec.
func DecodeJSON(data []byte, target any) error {
	return defaultCodec.Decode(data, target)
}
