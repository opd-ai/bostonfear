package serverengine

import "errors"

var (
	// ErrGameFull indicates the server cannot register additional players because
	// MaxPlayers has been reached.
	ErrGameFull = errors.New("game is full")

	// ErrInvalidPlayer indicates a player identifier or reconnect context did not
	// match a known player state.
	ErrInvalidPlayer = errors.New("invalid player")

	// ErrStateCorrupted indicates validation detected unrecoverable game-state
	// corruption.
	ErrStateCorrupted = errors.New("game state validation failed")
)
