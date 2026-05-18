// File: serverengine/eldritchhorror/actions/perform.go
// Action dispatch and routing for Eldritch Horror game family.
//
// This package implements the action dispatcher pattern specific to Eldritch Horror,
// which differs from Arkham Horror in its action set (travel, local, component, rest, trade, expedition).
package actions

import (
	"fmt"
	"strings"
)

// ActionType represents the canonical action type name for Eldritch Horror.
type ActionType string

// CallbackSet provides the actual action performer methods for Eldritch Horror.
// These callbacks are injected by the game engine and execute the actual game logic.
type CallbackSet struct {
	Travel            func(fromCity, toCity string, usesTicket bool) error
	LocalAction       func(playerID, cityID, actionID string) (interface{}, int, string, error)
	ComponentAction   func(playerID, componentID, actionID string) (interface{}, int, string, error)
	Rest              func(playerID string) (healthGained, sanityGained int, error error)
	Trade             func(fromPlayerID, toPlayerID, itemID string) error
	PrepareExpedition func(playerID, regionID string) (interface{}, int, string, error)
}

// DispatchAction is the Eldritch Horror action dispatcher.
// Routes actions to appropriate callbacks based on action type.
//
// actionName is the lowercase action type (e.g., "travel", "localAction", "rest").
// callbacks provide the actual perform method implementations.
// playerID identifies the acting investigator.
// target is the action parameter (city name, component ID, etc.).
// extraParams contains additional action-specific parameters as a map.
//
// Returns: (diceResult interface{}, doomIncrease int, actionResult string, err error)
func DispatchAction(
	actionName string,
	target string,
	extraParams map[string]interface{},
	callbacks CallbackSet,
	playerID string,
) (interface{}, int, string, error) {
	actionResult := "success"
	var diceResult interface{}
	var doomIncrease int
	var actionErr error

	// Normalize action name to lowercase for comparison
	normalized := strings.ToLower(actionName)

	switch normalized {
	case "travel":
		if callbacks.Travel == nil {
			return nil, 0, "", fmt.Errorf("travel callback not set")
		}
		// Extract travel parameters: fromCity, toCity, usesTicket
		fromCity := getStringParam(extraParams, "fromCity", "")
		toCity := target
		usesTicket := getBoolParam(extraParams, "usesTicket", false)
		actionErr = callbacks.Travel(fromCity, toCity, usesTicket)

	case "localaction":
		if callbacks.LocalAction == nil {
			return nil, 0, "", fmt.Errorf("localAction callback not set")
		}
		// Extract local action parameters: cityID, actionID
		cityID := getStringParam(extraParams, "cityID", "")
		actionID := target
		diceResult, doomIncrease, actionResult, actionErr = callbacks.LocalAction(playerID, cityID, actionID)

	case "componentaction":
		if callbacks.ComponentAction == nil {
			return nil, 0, "", fmt.Errorf("componentAction callback not set")
		}
		// Extract component parameters: componentID, actionID
		componentID := target
		actionID := getStringParam(extraParams, "actionID", "")
		diceResult, doomIncrease, actionResult, actionErr = callbacks.ComponentAction(playerID, componentID, actionID)

	case "restaction", "rest":
		if callbacks.Rest == nil {
			return nil, 0, "", fmt.Errorf("rest callback not set")
		}
		healthGained, sanityGained, restErr := callbacks.Rest(playerID)
		if restErr != nil {
			actionErr = restErr
		} else {
			// Package rest results into a simple result structure
			diceResult = map[string]interface{}{
				"healthGained": healthGained,
				"sanityGained": sanityGained,
			}
		}

	case "tradeaction", "trade":
		if callbacks.Trade == nil {
			return nil, 0, "", fmt.Errorf("trade callback not set")
		}
		// Extract trade parameters: toPlayerID, itemID
		toPlayerID := target
		itemID := getStringParam(extraParams, "itemID", "")
		actionErr = callbacks.Trade(playerID, toPlayerID, itemID)

	case "prepareexpedition", "expedition":
		if callbacks.PrepareExpedition == nil {
			return nil, 0, "", fmt.Errorf("prepareExpedition callback not set")
		}
		// Extract expedition parameters: regionID
		regionID := target
		diceResult, doomIncrease, actionResult, actionErr = callbacks.PrepareExpedition(playerID, regionID)

	default:
		return nil, 0, "", fmt.Errorf("unknown Eldritch Horror action type: %s", actionName)
	}

	return diceResult, doomIncrease, actionResult, actionErr
}

// Helper functions to extract parameters from extraParams map

func getStringParam(params map[string]interface{}, key, defaultValue string) string {
	if params == nil {
		return defaultValue
	}
	if val, ok := params[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if params == nil {
		return defaultValue
	}
	if val, ok := params[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultValue
}
