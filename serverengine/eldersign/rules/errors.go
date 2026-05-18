package rules

import "errors"

// Error definitions for Elder Sign rules validation.
var (
	// ErrInvalidDieIndex indicates an attempt to lock a die at an invalid index.
	ErrInvalidDieIndex = errors.New("invalid die index")

	// ErrInsufficientStamina indicates an action requires more stamina than available.
	ErrInsufficientStamina = errors.New("insufficient stamina")

	// ErrInsufficientSanity indicates an action requires more sanity than available.
	ErrInsufficientSanity = errors.New("insufficient sanity")

	// ErrInvalidAdventure indicates an adventure card reference is invalid or missing.
	ErrInvalidAdventure = errors.New("invalid adventure reference")
)
