package rules

import "testing"

func TestNewAncientOneState(t *testing.T) {
	ao := AncientOne{
		ID:        "test",
		Name:      "Test Ancient One",
		DoomTrack: 12,
	}

	state := NewAncientOneState(ao)
	if state == nil {
		t.Fatal("expected ancient one state")
	}
	if state.CurrentDoom != 0 {
		t.Errorf("expected initial doom 0, got %d", state.CurrentDoom)
	}
	if state.Current.Name != "Test Ancient One" {
		t.Errorf("expected name 'Test Ancient One', got %s", state.Current.Name)
	}
}

func TestAddDoom(t *testing.T) {
	ao := AncientOne{
		ID:        "test",
		DoomTrack: 12,
	}
	state := NewAncientOneState(ao)

	// Add doom below threshold
	shouldAwaken := state.AddDoom(5)
	if shouldAwaken {
		t.Error("should not awaken at 5 doom")
	}
	if state.CurrentDoom != 5 {
		t.Errorf("expected doom 5, got %d", state.CurrentDoom)
	}

	// Add doom to reach threshold
	shouldAwaken = state.AddDoom(7)
	if !shouldAwaken {
		t.Error("should awaken at 12 doom")
	}
	if state.CurrentDoom != 12 {
		t.Errorf("expected doom 12, got %d", state.CurrentDoom)
	}
}

func TestAwaken(t *testing.T) {
	ao := AncientOne{
		ID:         "test",
		DoomTrack:  12,
		IsAwakened: false,
	}
	state := NewAncientOneState(ao)

	if state.IsAwakened() {
		t.Error("should not be awakened initially")
	}

	err := state.Awaken()
	if err != nil {
		t.Errorf("unexpected error awakening: %v", err)
	}

	if !state.IsAwakened() {
		t.Error("should be awakened after Awaken()")
	}

	// Test double awakening fails
	err = state.Awaken()
	if err == nil {
		t.Error("expected error on double awakening")
	}
}

func TestGetDoomProgress(t *testing.T) {
	ao := AncientOne{
		ID:        "test",
		DoomTrack: 12,
	}
	state := NewAncientOneState(ao)
	state.AddDoom(5)

	current, max := state.GetDoomProgress()
	if current != 5 {
		t.Errorf("expected current doom 5, got %d", current)
	}
	if max != 12 {
		t.Errorf("expected max doom 12, got %d", max)
	}
}

func TestCheckAwakeningCondition(t *testing.T) {
	ao := AncientOne{
		ID:        "test",
		DoomTrack: 12,
	}
	state := NewAncientOneState(ao)
	state.AddDoom(12)

	tests := []struct {
		name              string
		condition         AwakeningCondition
		gateCount         int
		investigatorCount int
		expected          bool
	}{
		{
			name: "doom reaches max",
			condition: AwakeningCondition{
				DoomReachesMax: true,
			},
			gateCount:         0,
			investigatorCount: 4,
			expected:          true,
		},
		{
			name: "too many gates",
			condition: AwakeningCondition{
				TooManyGates: 5,
			},
			gateCount:         6,
			investigatorCount: 4,
			expected:          true,
		},
		{
			name: "too few investigators",
			condition: AwakeningCondition{
				TooFewInvestigators: 2,
			},
			gateCount:         0,
			investigatorCount: 1,
			expected:          true,
		},
		{
			name: "no condition met",
			condition: AwakeningCondition{
				TooManyGates:        10,
				TooFewInvestigators: 1,
			},
			gateCount:         5,
			investigatorCount: 4,
			expected:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := state.CheckAwakeningCondition(tt.condition, tt.gateCount, tt.investigatorCount)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetAbilities(t *testing.T) {
	ao := AncientOne{
		ID: "test",
		Abilities: []AncientOneAbility{
			{Name: "Ability 1", Trigger: TriggerPassive},
			{Name: "Ability 2", Trigger: TriggerEachMythos},
		},
	}
	state := NewAncientOneState(ao)

	abilities := state.GetAbilities()
	if len(abilities) != 2 {
		t.Errorf("expected 2 abilities, got %d", len(abilities))
	}
}

func TestGetAbilitiesForTrigger(t *testing.T) {
	ao := AncientOne{
		ID: "test",
		Abilities: []AncientOneAbility{
			{Name: "Passive 1", Trigger: TriggerPassive},
			{Name: "Mythos 1", Trigger: TriggerEachMythos},
			{Name: "Passive 2", Trigger: TriggerPassive},
		},
	}
	state := NewAncientOneState(ao)

	passives := state.GetAbilitiesForTrigger(TriggerPassive)
	if len(passives) != 2 {
		t.Errorf("expected 2 passive abilities, got %d", len(passives))
	}

	mythos := state.GetAbilitiesForTrigger(TriggerEachMythos)
	if len(mythos) != 1 {
		t.Errorf("expected 1 mythos ability, got %d", len(mythos))
	}

	combat := state.GetAbilitiesForTrigger(TriggerInCombat)
	if len(combat) != 0 {
		t.Errorf("expected 0 combat abilities, got %d", len(combat))
	}
}

func TestAncientOneString(t *testing.T) {
	ao := AncientOne{
		ID:         "test",
		Name:       "Test Ancient One",
		DoomTrack:  12,
		IsAwakened: false,
	}

	str := ao.String()
	expected := "Ancient One: Test Ancient One [Dormant] (Doom: 12)"
	if str != expected {
		t.Errorf("expected '%s', got '%s'", expected, str)
	}

	ao.IsAwakened = true
	str = ao.String()
	expected = "Ancient One: Test Ancient One [Awakened] (Doom: 12)"
	if str != expected {
		t.Errorf("expected '%s', got '%s'", expected, str)
	}
}

func TestPredefinedAncientOnes(t *testing.T) {
	ancientOnes := PredefinedAncientOnes()
	if len(ancientOnes) < 3 {
		t.Errorf("expected at least 3 predefined Ancient Ones, got %d", len(ancientOnes))
	}

	// Verify each has required fields
	for _, ao := range ancientOnes {
		if ao.ID == "" {
			t.Error("Ancient One missing ID")
		}
		if ao.Name == "" {
			t.Error("Ancient One missing Name")
		}
		if ao.DoomTrack <= 0 {
			t.Errorf("Ancient One %s has invalid doom track: %d", ao.Name, ao.DoomTrack)
		}
		if ao.CombatRating <= 0 {
			t.Errorf("Ancient One %s has invalid combat rating: %d", ao.Name, ao.CombatRating)
		}
	}
}

func TestAbilityTriggers(t *testing.T) {
	// Verify trigger constants exist and are distinct
	triggers := []AbilityTrigger{
		TriggerOnAwaken,
		TriggerEachMythos,
		TriggerWhenGateOpens,
		TriggerInCombat,
		TriggerPassive,
	}

	seen := make(map[AbilityTrigger]bool)
	for _, trigger := range triggers {
		if seen[trigger] {
			t.Errorf("duplicate trigger: %s", trigger)
		}
		seen[trigger] = true
	}

	if len(seen) != 5 {
		t.Errorf("expected 5 distinct triggers, got %d", len(seen))
	}
}
