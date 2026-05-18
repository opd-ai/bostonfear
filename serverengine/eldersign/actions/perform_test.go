package actions

import (
	"errors"
	"testing"

	"github.com/opd-ai/bostonfear/serverengine/eldersign/rules"
)

func TestDispatchAction_PlaceInvestigator(t *testing.T) {
	called := false
	cb := CallbackSet{
		PlaceInvestigator: func(adventureID string) error {
			called = true
			if adventureID != "entryHall" {
				t.Errorf("expected adventureID 'entryHall', got %s", adventureID)
			}
			return nil
		},
	}

	_, doom, result, err := DispatchAction(string(rules.ActionPlaceInvestigator), "entryHall", 0, cb, "p1")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !called {
		t.Error("PlaceInvestigator callback not called")
	}
	if doom != 0 {
		t.Errorf("expected doom=0, got %d", doom)
	}
	if result != "success" {
		t.Errorf("expected result='success', got %s", result)
	}
}

func TestDispatchAction_RollDice(t *testing.T) {
	called := false
	cb := CallbackSet{
		RollDice: func(playerID string) (interface{}, int, string, error) {
			called = true
			if playerID != "p1" {
				t.Errorf("expected playerID 'p1', got %s", playerID)
			}
			return map[string]interface{}{"results": []string{"red", "green"}}, 1, "success", nil
		},
	}

	res, doom, result, err := DispatchAction(string(rules.ActionRollDice), "", 0, cb, "p1")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !called {
		t.Error("RollDice callback not called")
	}
	if doom != 1 {
		t.Errorf("expected doom=1, got %d", doom)
	}
	if result != "success" {
		t.Errorf("expected result='success', got %s", result)
	}
	if res == nil {
		t.Error("expected non-nil result")
	}
}

func TestDispatchAction_LockDie(t *testing.T) {
	called := false
	cb := CallbackSet{
		LockDie: func(dieIndex int) error {
			called = true
			if dieIndex != 2 {
				t.Errorf("expected dieIndex 2, got %d", dieIndex)
			}
			return nil
		},
	}

	_, doom, result, err := DispatchAction(string(rules.ActionLockDie), "", 2, cb, "p1")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !called {
		t.Error("LockDie callback not called")
	}
	if doom != 0 {
		t.Errorf("expected doom=0, got %d", doom)
	}
	if result != "success" {
		t.Errorf("expected result='success', got %s", result)
	}
}

func TestDispatchAction_DiscardItem(t *testing.T) {
	called := false
	cb := CallbackSet{
		DiscardItem: func(itemID string) error {
			called = true
			if itemID != "flashlight" {
				t.Errorf("expected itemID 'flashlight', got %s", itemID)
			}
			return nil
		},
	}

	_, doom, result, err := DispatchAction(string(rules.ActionDiscardItem), "flashlight", 0, cb, "p1")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !called {
		t.Error("DiscardItem callback not called")
	}
	if doom != 0 {
		t.Errorf("expected doom=0, got %d", doom)
	}
	if result != "success" {
		t.Errorf("expected result='success', got %s", result)
	}
}

func TestDispatchAction_ClaimAdventure(t *testing.T) {
	called := false
	cb := CallbackSet{
		ClaimAdventure: func(playerID string) (interface{}, int, string, error) {
			called = true
			if playerID != "p1" {
				t.Errorf("expected playerID 'p1', got %s", playerID)
			}
			return map[string]interface{}{"elderSigns": 1}, 0, "success", nil
		},
	}

	res, doom, result, err := DispatchAction(string(rules.ActionClaimAdventure), "", 0, cb, "p1")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !called {
		t.Error("ClaimAdventure callback not called")
	}
	if doom != 0 {
		t.Errorf("expected doom=0, got %d", doom)
	}
	if result != "success" {
		t.Errorf("expected result='success', got %s", result)
	}
	if res == nil {
		t.Error("expected non-nil result")
	}
}

func TestDispatchAction_UnknownAction(t *testing.T) {
	_, _, _, err := DispatchAction("teleport", "", 0, CallbackSet{}, "p1")
	if err == nil {
		t.Error("expected error for unknown action")
	}
	if err.Error() != "unknown Elder Sign action type: teleport" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestDispatchAction_NilCallback(t *testing.T) {
	_, _, _, err := DispatchAction(string(rules.ActionPlaceInvestigator), "entryHall", 0, CallbackSet{}, "p1")
	if err == nil {
		t.Error("expected error for nil callback")
	}
	if err.Error() != "placeinvestigator callback not set" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestDispatchAction_CallbackError(t *testing.T) {
	cb := CallbackSet{
		PlaceInvestigator: func(adventureID string) error {
			return errors.New("adventure location full")
		},
	}
	_, _, _, err := DispatchAction(string(rules.ActionPlaceInvestigator), "entryHall", 0, cb, "p1")
	if err == nil {
		t.Error("expected callback error to propagate")
	}
	if err.Error() != "adventure location full" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestDispatchAction_CaseInsensitive(t *testing.T) {
	called := false
	cb := CallbackSet{
		PlaceInvestigator: func(adventureID string) error {
			called = true
			return nil
		},
	}

	// Test uppercase variant
	_, _, _, err := DispatchAction("PlaceInvestigator", "entryHall", 0, cb, "p1")
	if err != nil {
		t.Errorf("expected no error for uppercase action, got %v", err)
	}
	if !called {
		t.Error("callback not called for uppercase action")
	}
}
