package actions

import (
	"fmt"
	"testing"
)

func TestDispatchAction_Travel(t *testing.T) {
	travelCalled := false
	callbacks := CallbackSet{
		Travel: func(fromCity, toCity string, usesTicket bool) error {
			travelCalled = true
			if toCity != "london" {
				return fmt.Errorf("expected london, got %s", toCity)
			}
			if fromCity != "arkham" {
				return fmt.Errorf("expected from arkham, got %s", fromCity)
			}
			if !usesTicket {
				return fmt.Errorf("expected usesTicket to be true")
			}
			return nil
		},
	}

	params := map[string]interface{}{
		"fromCity":   "arkham",
		"usesTicket": true,
	}

	_, _, _, err := DispatchAction("travel", "london", params, callbacks, "player1")
	if err != nil {
		t.Errorf("DispatchAction failed: %v", err)
	}
	if !travelCalled {
		t.Error("Travel callback was not called")
	}
}

func TestDispatchAction_LocalAction(t *testing.T) {
	localActionCalled := false
	callbacks := CallbackSet{
		LocalAction: func(playerID, cityID, actionID string) (interface{}, int, string, error) {
			localActionCalled = true
			if cityID != "tokyo" {
				return nil, 0, "", fmt.Errorf("expected tokyo, got %s", cityID)
			}
			if actionID != "gather" {
				return nil, 0, "", fmt.Errorf("expected gather, got %s", actionID)
			}
			return map[string]interface{}{"result": "gathered"}, 0, "success", nil
		},
	}

	params := map[string]interface{}{
		"cityID": "tokyo",
	}

	diceResult, doom, result, err := DispatchAction("localaction", "gather", params, callbacks, "player1")
	if err != nil {
		t.Errorf("DispatchAction failed: %v", err)
	}
	if !localActionCalled {
		t.Error("LocalAction callback was not called")
	}
	if result != "success" {
		t.Errorf("Expected success, got %s", result)
	}
	if doom != 0 {
		t.Errorf("Expected doom 0, got %d", doom)
	}
	if diceResult == nil {
		t.Error("Expected non-nil dice result")
	}
}

func TestDispatchAction_Rest(t *testing.T) {
	restCalled := false
	callbacks := CallbackSet{
		Rest: func(playerID string) (int, int, error) {
			restCalled = true
			if playerID != "player1" {
				return 0, 0, fmt.Errorf("expected player1, got %s", playerID)
			}
			return 2, 1, nil // healthGained, sanityGained
		},
	}

	diceResult, doom, result, err := DispatchAction("rest", "", nil, callbacks, "player1")
	if err != nil {
		t.Errorf("DispatchAction failed: %v", err)
	}
	if !restCalled {
		t.Error("Rest callback was not called")
	}
	if result != "success" {
		t.Errorf("Expected success, got %s", result)
	}
	if doom != 0 {
		t.Errorf("Expected doom 0, got %d", doom)
	}

	// Check rest result structure
	resultMap, ok := diceResult.(map[string]interface{})
	if !ok {
		t.Fatal("Expected diceResult to be a map")
	}
	if resultMap["healthGained"] != 2 {
		t.Errorf("Expected healthGained 2, got %v", resultMap["healthGained"])
	}
	if resultMap["sanityGained"] != 1 {
		t.Errorf("Expected sanityGained 1, got %v", resultMap["sanityGained"])
	}
}

func TestDispatchAction_Trade(t *testing.T) {
	tradeCalled := false
	callbacks := CallbackSet{
		Trade: func(fromPlayerID, toPlayerID, itemID string) error {
			tradeCalled = true
			if fromPlayerID != "player1" {
				return fmt.Errorf("expected player1, got %s", fromPlayerID)
			}
			if toPlayerID != "player2" {
				return fmt.Errorf("expected player2, got %s", toPlayerID)
			}
			if itemID != "ticket" {
				return fmt.Errorf("expected ticket, got %s", itemID)
			}
			return nil
		},
	}

	params := map[string]interface{}{
		"itemID": "ticket",
	}

	_, _, _, err := DispatchAction("trade", "player2", params, callbacks, "player1")
	if err != nil {
		t.Errorf("DispatchAction failed: %v", err)
	}
	if !tradeCalled {
		t.Error("Trade callback was not called")
	}
}

func TestDispatchAction_ComponentAction(t *testing.T) {
	componentCalled := false
	callbacks := CallbackSet{
		ComponentAction: func(playerID, componentID, actionID string) (interface{}, int, string, error) {
			componentCalled = true
			if componentID != "gate123" {
				return nil, 0, "", fmt.Errorf("expected gate123, got %s", componentID)
			}
			if actionID != "close" {
				return nil, 0, "", fmt.Errorf("expected close, got %s", actionID)
			}
			return map[string]interface{}{"closed": true}, 0, "success", nil
		},
	}

	params := map[string]interface{}{
		"actionID": "close",
	}

	diceResult, doom, result, err := DispatchAction("componentaction", "gate123", params, callbacks, "player1")
	if err != nil {
		t.Errorf("DispatchAction failed: %v", err)
	}
	if !componentCalled {
		t.Error("ComponentAction callback was not called")
	}
	if result != "success" {
		t.Errorf("Expected success, got %s", result)
	}
	if doom != 0 {
		t.Errorf("Expected doom 0, got %d", doom)
	}
	if diceResult == nil {
		t.Error("Expected non-nil dice result")
	}
}

func TestDispatchAction_PrepareExpedition(t *testing.T) {
	expeditionCalled := false
	callbacks := CallbackSet{
		PrepareExpedition: func(playerID, regionID string) (interface{}, int, string, error) {
			expeditionCalled = true
			if regionID != "amazon" {
				return nil, 0, "", fmt.Errorf("expected amazon, got %s", regionID)
			}
			return map[string]interface{}{"prepared": true}, 0, "success", nil
		},
	}

	diceResult, doom, result, err := DispatchAction("expedition", "amazon", nil, callbacks, "player1")
	if err != nil {
		t.Errorf("DispatchAction failed: %v", err)
	}
	if !expeditionCalled {
		t.Error("PrepareExpedition callback was not called")
	}
	if result != "success" {
		t.Errorf("Expected success, got %s", result)
	}
	if doom != 0 {
		t.Errorf("Expected doom 0, got %d", doom)
	}
	if diceResult == nil {
		t.Error("Expected non-nil dice result")
	}
}

func TestDispatchAction_UnknownAction(t *testing.T) {
	callbacks := CallbackSet{}
	_, _, _, err := DispatchAction("unknownaction", "", nil, callbacks, "player1")
	if err == nil {
		t.Error("Expected error for unknown action")
	}
	if err.Error() != "unknown Eldritch Horror action type: unknownaction" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestDispatchAction_MissingCallback(t *testing.T) {
	callbacks := CallbackSet{}
	_, _, _, err := DispatchAction("travel", "london", nil, callbacks, "player1")
	if err == nil {
		t.Error("Expected error for missing travel callback")
	}
	if err.Error() != "travel callback not set" {
		t.Errorf("Unexpected error message: %v", err)
	}
}
