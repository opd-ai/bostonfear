package actions

import (
	"errors"
	"testing"
)

func TestDispatchAction_Move(t *testing.T) {
	called := false
	cb := CallbackSet{
		Move: func(target string) error {
			called = true
			if target != "University" {
				t.Errorf("expected target University, got %s", target)
			}
			return nil
		},
	}
	_, doom, result, err := DispatchAction("move", "University", 0, cb, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("Move callback not called")
	}
	if doom != 0 {
		t.Errorf("expected doom 0, got %d", doom)
	}
	if result != "success" {
		t.Errorf("expected result success, got %s", result)
	}
}

func TestDispatchAction_MoveUppercase(t *testing.T) {
	called := false
	cb := CallbackSet{
		Move: func(_ string) error { called = true; return nil },
	}
	_, _, _, err := DispatchAction("Move", "Downtown", 0, cb, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("Move callback not called for uppercase action name")
	}
}

func TestDispatchAction_Gather(t *testing.T) {
	cb := CallbackSet{
		Gather: func(playerID string, focusSpend int) (interface{}, int, string, error) {
			return "dice-result", 1, "success", nil
		},
	}
	dice, doom, result, err := DispatchAction("gather", "", 0, cb, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dice != "dice-result" {
		t.Errorf("expected dice-result, got %v", dice)
	}
	if doom != 1 {
		t.Errorf("expected doom 1, got %d", doom)
	}
	if result != "success" {
		t.Errorf("expected success, got %s", result)
	}
}

func TestDispatchAction_Investigate(t *testing.T) {
	cb := CallbackSet{
		Investigate: func(playerID string, focusSpend int) (interface{}, int, string, error) {
			return nil, 0, "fail", nil
		},
	}
	_, doom, result, err := DispatchAction("investigate", "", 0, cb, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doom != 0 {
		t.Errorf("expected doom 0, got %d", doom)
	}
	if result != "fail" {
		t.Errorf("expected fail, got %s", result)
	}
}

func TestDispatchAction_Ward(t *testing.T) {
	cb := CallbackSet{
		CastWard: func(playerID string, focusSpend int) (interface{}, int, string, error) {
			return nil, 0, "success", nil
		},
	}
	_, _, result, err := DispatchAction("ward", "", 0, cb, "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected success, got %s", result)
	}
}

func TestDispatchAction_UnknownAction(t *testing.T) {
	_, _, _, err := DispatchAction("teleport", "", 0, CallbackSet{}, "p1")
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
}

func TestDispatchAction_NilCallback(t *testing.T) {
	_, _, _, err := DispatchAction("move", "Downtown", 0, CallbackSet{}, "p1")
	if err == nil {
		t.Fatal("expected error when Move callback is nil")
	}
}

func TestDispatchAction_CallbackError(t *testing.T) {
	cb := CallbackSet{
		Move: func(_ string) error { return errors.New("blocked") },
	}
	_, _, _, err := DispatchAction("move", "Northside", 0, cb, "p1")
	if err == nil || err.Error() != "blocked" {
		t.Fatalf("expected blocked error, got %v", err)
	}
}
