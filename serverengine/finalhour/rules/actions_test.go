package rules

import "testing"

func TestValidActions(t *testing.T) {
	actions := ValidActions()

	expected := 4
	if len(actions) != expected {
		t.Errorf("expected %d actions, got %d", expected, len(actions))
	}

	// Verify all expected actions are present
	found := make(map[ActionType]bool)
	for _, action := range actions {
		found[action] = true
	}

	if !found[ActionPlaceInvestigator] {
		t.Error("ActionPlaceInvestigator not found in valid actions")
	}
	if !found[ActionResolveAction] {
		t.Error("ActionResolveAction not found in valid actions")
	}
	if !found[ActionBidPriority] {
		t.Error("ActionBidPriority not found in valid actions")
	}
	if !found[ActionSpendFocus] {
		t.Error("ActionSpendFocus not found in valid actions")
	}
}

func TestActionTypeString(t *testing.T) {
	action := ActionPlaceInvestigator
	if action.String() != "placeInvestigator" {
		t.Errorf("expected action string to be 'placeInvestigator', got '%s'", action.String())
	}
}
