// Package validation provides shared validation primitives for cross-engine use.
// ActionChecker validates whether a requested action is legal in the current game state.
package validation

import "fmt"

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

// TurnChecker validates common turn-gated action preconditions.
// It intentionally excludes player-existence and defeat checks, which remain
// engine-owned concerns with richer domain context.
type TurnChecker struct {
	GamePhase        string
	CurrentPlayer    string
	ActionsRemaining int
	IsAllowedAction  func(actionType string) bool
}

// IsLegal validates action legality for the given player under current turn state.
func (c TurnChecker) IsLegal(actionType string, playerID string) error {
	if c.GamePhase != "playing" {
		return &Error{Field: "gamePhase", Message: "game is not in playing state"}
	}
	if c.CurrentPlayer != playerID {
		return &Error{Field: "currentPlayer", Message: fmt.Sprintf("not player %s's turn (current: %s)", playerID, c.CurrentPlayer)}
	}
	if c.ActionsRemaining <= 0 {
		return &Error{Field: "actionsRemaining", Message: fmt.Sprintf("player %s has no actions remaining", playerID)}
	}
	if c.IsAllowedAction != nil && !c.IsAllowedAction(actionType) {
		return &Error{Field: "action", Message: fmt.Sprintf("invalid action type: %s", actionType)}
	}
	return nil
}
