package rules

import "testing"

func TestValidActions(t *testing.T) {
	actions := ValidActions()
	if len(actions) != 6 {
		t.Errorf("expected 6 valid actions, got %d", len(actions))
	}

	expected := map[ActionType]bool{
		ActionTravel:            true,
		ActionLocalAction:       true,
		ActionComponentAction:   true,
		ActionRestAction:        true,
		ActionTradeAction:       true,
		ActionPrepareExpedition: true,
	}

	for _, action := range actions {
		if !expected[action] {
			t.Errorf("unexpected action in ValidActions: %s", action)
		}
	}
}

func TestActionTypeString(t *testing.T) {
	tests := []struct {
		action   ActionType
		expected string
	}{
		{ActionTravel, "travel"},
		{ActionLocalAction, "localAction"},
		{ActionComponentAction, "componentAction"},
		{ActionRestAction, "restAction"},
		{ActionTradeAction, "tradeAction"},
		{ActionPrepareExpedition, "prepareExpedition"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.action.String(); got != tt.expected {
				t.Errorf("ActionType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestActionCost(t *testing.T) {
	tests := []struct {
		action       ActionType
		expectedCost int
	}{
		{ActionTravel, 1},
		{ActionLocalAction, 1},
		{ActionComponentAction, 1},
		{ActionRestAction, 1},
		{ActionTradeAction, 1},
		{ActionPrepareExpedition, 1},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := tt.action.ActionCost(); got != tt.expectedCost {
				t.Errorf("ActionType.ActionCost() = %v, want %v", got, tt.expectedCost)
			}
		})
	}
}

func TestActionTypeConstants(t *testing.T) {
	// Verify constants match expected string values
	if ActionTravel != "travel" {
		t.Errorf("ActionTravel constant mismatch")
	}
	if ActionLocalAction != "localAction" {
		t.Errorf("ActionLocalAction constant mismatch")
	}
	if ActionComponentAction != "componentAction" {
		t.Errorf("ActionComponentAction constant mismatch")
	}
	if ActionRestAction != "restAction" {
		t.Errorf("ActionRestAction constant mismatch")
	}
	if ActionTradeAction != "tradeAction" {
		t.Errorf("ActionTradeAction constant mismatch")
	}
	if ActionPrepareExpedition != "prepareExpedition" {
		t.Errorf("ActionPrepareExpedition constant mismatch")
	}
}
