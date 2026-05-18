package content

import "testing"

func TestDefaultInvestigators_ContainsAtLeast10(t *testing.T) {
	investigators := DefaultInvestigators()

	if len(investigators) < 10 {
		t.Errorf("expected at least 10 investigators, got %d", len(investigators))
	}
}

func TestDefaultInvestigators_AllHaveRequiredFields(t *testing.T) {
	investigators := DefaultInvestigators()

	for _, inv := range investigators {
		if inv.ID == "" {
			t.Errorf("investigator missing ID")
		}
		if inv.Name == "" {
			t.Errorf("investigator %s missing Name", inv.ID)
		}
		if !ValidateInvestigator(inv.StartingStamina, inv.StartingSanity) {
			t.Errorf("investigator %s has invalid stamina %d or sanity %d (must be 1-8)", inv.ID, inv.StartingStamina, inv.StartingSanity)
		}
		if len(inv.StartingItems) == 0 {
			t.Errorf("investigator %s has no starting items", inv.ID)
		}
		if inv.SpecialAbility == "" {
			t.Errorf("investigator %s missing SpecialAbility", inv.ID)
		}
		if inv.AbilityDesc == "" {
			t.Errorf("investigator %s missing AbilityDesc", inv.ID)
		}
	}
}

func TestDefaultInvestigators_ResourceRanges(t *testing.T) {
	investigators := DefaultInvestigators()

	for _, inv := range investigators {
		if inv.StartingStamina < 1 || inv.StartingStamina > 8 {
			t.Errorf("investigator %s has stamina %d outside Elder Sign range [1,8]", inv.ID, inv.StartingStamina)
		}
		if inv.StartingSanity < 1 || inv.StartingSanity > 8 {
			t.Errorf("investigator %s has sanity %d outside Elder Sign range [1,8]", inv.ID, inv.StartingSanity)
		}
	}
}

func TestDefaultInvestigators_BalancedDistribution(t *testing.T) {
	investigators := DefaultInvestigators()

	highStamina := 0
	highSanity := 0
	balanced := 0

	for _, inv := range investigators {
		if inv.StartingStamina >= 7 {
			highStamina++
		}
		if inv.StartingSanity >= 7 {
			highSanity++
		}
		if inv.StartingStamina == 6 && inv.StartingSanity == 6 {
			balanced++
		}
	}

	if highStamina == 0 {
		t.Error("no investigators with high stamina (7-8)")
	}
	if highSanity == 0 {
		t.Error("no investigators with high sanity (7-8)")
	}
	if balanced == 0 {
		t.Error("no balanced investigators (6 stamina, 6 sanity)")
	}
}

func TestInvestigatorByID_FindsExisting(t *testing.T) {
	testIDs := []string{
		"inv.roland.banks",
		"inv.daisy.walker",
		"inv.agnes.baker",
	}

	for _, id := range testIDs {
		inv, found := InvestigatorByID(id)
		if !found {
			t.Errorf("expected to find investigator %s", id)
		}
		if inv.ID != id {
			t.Errorf("expected ID %s, got %s", id, inv.ID)
		}
	}
}

func TestInvestigatorByID_NotFoundReturnsEmpty(t *testing.T) {
	_, found := InvestigatorByID("inv.nonexistent")
	if found {
		t.Error("expected not to find nonexistent investigator")
	}
}

func TestValidateInvestigator_AcceptsValidRanges(t *testing.T) {
	validCases := []struct {
		stamina int
		sanity  int
	}{
		{1, 8},
		{8, 1},
		{5, 5},
		{4, 8},
		{8, 4},
	}

	for _, tc := range validCases {
		if !ValidateInvestigator(tc.stamina, tc.sanity) {
			t.Errorf("expected stamina %d, sanity %d to be valid", tc.stamina, tc.sanity)
		}
	}
}

func TestValidateInvestigator_RejectsInvalidRanges(t *testing.T) {
	invalidCases := []struct {
		stamina int
		sanity  int
	}{
		{0, 5},
		{5, 0},
		{9, 5},
		{5, 9},
		{10, 10},
		{-1, 5},
	}

	for _, tc := range invalidCases {
		if ValidateInvestigator(tc.stamina, tc.sanity) {
			t.Errorf("expected stamina %d, sanity %d to be invalid", tc.stamina, tc.sanity)
		}
	}
}
