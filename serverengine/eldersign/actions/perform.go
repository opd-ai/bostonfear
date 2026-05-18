// Package actions provides Elder Sign-specific action dispatching and validation.
// This package owns the routing logic for Elder Sign actions without importing serverengine,
// enabling testability through callback injection.
package actions

import (
	"fmt"
	"strings"
)

// CallbackSet provides the actual action performer methods for Elder Sign.
// These callbacks are injected by the serverengine GameServer to execute
// game state mutations while keeping the dispatch logic module-owned.
type CallbackSet struct {
	PlaceInvestigator func(adventureID string) error
	RollDice          func(playerID string) (interface{}, int, string, error)
	LockDie           func(dieIndex int) error
	DiscardItem       func(itemID string) error
	ClaimAdventure    func(playerID string) (interface{}, int, string, error)
}

// DispatchAction routes Elder Sign actions to appropriate callbacks.
// This function implements action dispatch logic owned by the eldersign module.
// The actual perform implementations are provided via callbacks from serverengine.GameServer.
//
// actionName is the lowercase action type (e.g., "placeinvestigator", "rolldice").
// callbacks provide the actual perform method implementations.
// playerID, target, and actionParam are the action parameters.
//
// Returns: (result interface{}, doomIncrease int, actionResult string, err error)
func DispatchAction(
	actionName string,
	target string,
	actionParam int,
	callbacks CallbackSet,
	playerID string,
) (interface{}, int, string, error) {
	actionResult := "success"
	var result interface{}
	var doomIncrease int
	var actionErr error

	// Normalize action name to lowercase for comparison
	normalized := strings.ToLower(actionName)

	// Route to callback implementations provided by serverengine.GameServer
	switch normalized {
	case "placeinvestigator":
		if callbacks.PlaceInvestigator == nil {
			return nil, 0, "", fmt.Errorf("placeinvestigator callback not set")
		}
		actionErr = callbacks.PlaceInvestigator(target)

	case "rolldice":
		if callbacks.RollDice == nil {
			return nil, 0, "", fmt.Errorf("rolldice callback not set")
		}
		result, doomIncrease, actionResult, actionErr = callbacks.RollDice(playerID)

	case "lockdie":
		if callbacks.LockDie == nil {
			return nil, 0, "", fmt.Errorf("lockdie callback not set")
		}
		actionErr = callbacks.LockDie(actionParam)

	case "discarditem":
		if callbacks.DiscardItem == nil {
			return nil, 0, "", fmt.Errorf("discarditem callback not set")
		}
		actionErr = callbacks.DiscardItem(target)

	case "claimadventure":
		if callbacks.ClaimAdventure == nil {
			return nil, 0, "", fmt.Errorf("claimadventure callback not set")
		}
		result, doomIncrease, actionResult, actionErr = callbacks.ClaimAdventure(playerID)

	default:
		return nil, 0, "", fmt.Errorf("unknown Elder Sign action type: %s", actionName)
	}

	return result, doomIncrease, actionResult, actionErr
}
