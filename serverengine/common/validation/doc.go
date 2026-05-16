// Package validation provides shared validation primitives for cross-engine use.
// ActionChecker validates whether a requested action is legal in the current game state.
package validation

// ActionChecker is the interface for checking whether an action is permitted.
type ActionChecker interface {
	// IsLegal returns nil if the action identified by actionType is permitted,
	// or a descriptive error explaining why it is not.
	IsLegal(actionType string, playerID string) error
}

// Error is a typed validation error returned when an action or state is invalid.
type Error struct {
	Field   string // The field or context that failed validation
	Message string // Human-readable description
}

func (e *Error) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Message
	}
	return e.Message
}
