// File: serverengine/arkhamhorror/actions/perform.go
// S1: Action dispatch and legality gates - arkhamhorror module ownership
//
// This package defines the action dispatcher logic without importing serverengine.
// The actual GameServer method calls are delegated through callback functions.
package actions

import (
	"fmt"
	"strings"
)

// ActionType represents the canonical action type name
type ActionType string

// CallbackSet provides the actual action performer methods
type CallbackSet struct {
	Move        func(target string) error
	Gather      func(playerID string, focusSpend int) (interface{}, int, string, error) // (*DiceResultMessage, int, actionResult, error)
	Investigate func(playerID string, focusSpend int) (interface{}, int, string, error)
	CastWard    func(playerID string, focusSpend int) (interface{}, int, string, error)
	Focus       func() error
	Research    func(playerID string, focusSpend int) (interface{}, int, string, error)
	Trade       func(fromID, toID string) error
	Encounter   func(playerID string) error
	Component   func(playerID string) (string, error)
	Attack      func(playerID string) (interface{}, int, string, error)
	Evade       func(playerID string) (interface{}, int, string, error)
	CloseGate   func(playerID string) (string, error)
}

// DispatchAction is the module-owned dispatcher that routes actions to callbacks.
// This function implements S1 migration: action dispatch logic is owned by arkhamhorror.
// The actual perform implementations are provided via callbacks from serverengine.GameServer.
//
// actionName is the lowercase action type (e.g., "move", "gather", "investigate").
// callbacks provide the actual perform method implementations.
// playerID, target, and focusSpend are the action parameters.
//
// Returns: (diceResult interface{}, doomIncrease int, actionResult string, err error)
func DispatchAction(
	actionName string,
	target string,
	focusSpend int,
	callbacks CallbackSet,
	playerID string,
) (interface{}, int, string, error) {
	actionResult := "success"
	var diceResult interface{}
	var doomIncrease int
	var actionErr error

	// Normalize action name to lowercase for comparison
	normalized := strings.ToLower(actionName)

	// S1 Migration: Action dispatch logic owned by arkhamhorror module
	// Routes to callback implementations provided by serverengine.GameServer
	switch normalized {
	case "move":
		if callbacks.Move == nil {
			return nil, 0, "", fmt.Errorf("move callback not set")
		}
		actionErr = callbacks.Move(target)

	case "gather":
		if callbacks.Gather == nil {
			return nil, 0, "", fmt.Errorf("gather callback not set")
		}
		diceResult, doomIncrease, actionResult, actionErr = callbacks.Gather(playerID, focusSpend)

	case "investigate":
		if callbacks.Investigate == nil {
			return nil, 0, "", fmt.Errorf("investigate callback not set")
		}
		diceResult, doomIncrease, actionResult, actionErr = callbacks.Investigate(playerID, focusSpend)

	case "ward":
		if callbacks.CastWard == nil {
			return nil, 0, "", fmt.Errorf("castward callback not set")
		}
		diceResult, doomIncrease, actionResult, actionErr = callbacks.CastWard(playerID, focusSpend)

	case "focus":
		if callbacks.Focus == nil {
			return nil, 0, "", fmt.Errorf("focus callback not set")
		}
		actionErr = callbacks.Focus()

	case "research":
		if callbacks.Research == nil {
			return nil, 0, "", fmt.Errorf("research callback not set")
		}
		diceResult, doomIncrease, actionResult, actionErr = callbacks.Research(playerID, focusSpend)

	case "trade":
		if callbacks.Trade == nil {
			return nil, 0, "", fmt.Errorf("trade callback not set")
		}
		actionErr = callbacks.Trade(playerID, target)

	case "encounter":
		if callbacks.Encounter == nil {
			return nil, 0, "", fmt.Errorf("encounter callback not set")
		}
		actionErr = callbacks.Encounter(playerID)

	case "component":
		if callbacks.Component == nil {
			return nil, 0, "", fmt.Errorf("component callback not set")
		}
		actionResult, actionErr = callbacks.Component(playerID)

	case "attack":
		if callbacks.Attack == nil {
			return nil, 0, "", fmt.Errorf("attack callback not set")
		}
		diceResult, doomIncrease, actionResult, actionErr = callbacks.Attack(playerID)

	case "evade":
		if callbacks.Evade == nil {
			return nil, 0, "", fmt.Errorf("evade callback not set")
		}
		diceResult, doomIncrease, actionResult, actionErr = callbacks.Evade(playerID)

	case "closegate":
		if callbacks.CloseGate == nil {
			return nil, 0, "", fmt.Errorf("closegate callback not set")
		}
		actionResult, actionErr = callbacks.CloseGate(playerID)

	default:
		return nil, 0, "", fmt.Errorf("unknown action type: %s", actionName)
	}

	return diceResult, doomIncrease, actionResult, actionErr
}
