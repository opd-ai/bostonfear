package rules

import (
	"testing"
)

func TestValidActions(t *testing.T) {
	actions := ValidActions()
	if len(actions) != 5 {
		t.Errorf("ValidActions() returned %d actions, expected 5", len(actions))
	}

	expected := []ActionType{
		ActionPlaceInvestigator,
		ActionRollDice,
		ActionLockDie,
		ActionDiscardItem,
		ActionClaimAdventure,
	}

	for i, exp := range expected {
		if actions[i] != exp {
			t.Errorf("ValidActions()[%d] = %v, want %v", i, actions[i], exp)
		}
	}
}

func TestActionTypeString(t *testing.T) {
	tests := []struct {
		action ActionType
		want   string
	}{
		{ActionPlaceInvestigator, "placeInvestigator"},
		{ActionRollDice, "rollDice"},
		{ActionLockDie, "lockDie"},
		{ActionDiscardItem, "discardItem"},
		{ActionClaimAdventure, "claimAdventure"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.action.String(); got != tt.want {
				t.Errorf("ActionType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActionTypeConstants(t *testing.T) {
	// Verify action type values match expected string identifiers
	tests := []struct {
		name   string
		action ActionType
		want   string
	}{
		{"PlaceInvestigator", ActionPlaceInvestigator, "placeInvestigator"},
		{"RollDice", ActionRollDice, "rollDice"},
		{"LockDie", ActionLockDie, "lockDie"},
		{"DiscardItem", ActionDiscardItem, "discardItem"},
		{"ClaimAdventure", ActionClaimAdventure, "claimAdventure"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.action) != tt.want {
				t.Errorf("%s constant = %v, want %v", tt.name, string(tt.action), tt.want)
			}
		})
	}
}
