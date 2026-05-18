package content

import "testing"

func TestDefaultMythosCards_ContainsAtLeast20(t *testing.T) {
	mythosCards := DefaultMythosCards()

	if len(mythosCards) < 20 {
		t.Errorf("expected at least 20 mythos cards, got %d", len(mythosCards))
	}
}

func TestDefaultMythosCards_AllHaveRequiredFields(t *testing.T) {
	mythosCards := DefaultMythosCards()

	for _, mythos := range mythosCards {
		if mythos.ID == "" {
			t.Errorf("mythos card missing ID")
		}
		if mythos.Name == "" {
			t.Errorf("mythos card %s missing Name", mythos.ID)
		}
		if mythos.Type == "" {
			t.Errorf("mythos card %s missing Type", mythos.ID)
		}
		if mythos.Effect == "" {
			t.Errorf("mythos card %s missing Effect", mythos.ID)
		}
		if len(mythos.ScenarioIDs) == 0 {
			t.Errorf("mythos card %s not assigned to any scenarios", mythos.ID)
		}
	}
}

func TestDefaultMythosCards_TypeDistribution(t *testing.T) {
	mythosCards := DefaultMythosCards()
	typeCount := map[string]int{}

	for _, mythos := range mythosCards {
		typeCount[mythos.Type]++
	}

	if typeCount["encounter"] == 0 {
		t.Error("no encounter-type mythos cards")
	}
	if typeCount["event"] == 0 {
		t.Error("no event-type mythos cards")
	}
	if typeCount["omen"] == 0 {
		t.Error("no omen-type mythos cards")
	}
}

func TestMythosForScenario_FiltersCorrectly(t *testing.T) {
	scenarios := []string{
		"eldersign.azathoth.madness",
		"eldersign.yig.serpent",
		"eldersign.cthulhu.depths",
		"eldersign.hastur.king",
	}

	for _, scenarioID := range scenarios {
		mythosCards := MythosForScenario(scenarioID)

		if len(mythosCards) == 0 {
			t.Errorf("scenario %s has no mythos cards", scenarioID)
		}

		// Verify all returned cards actually include this scenario
		for _, mythos := range mythosCards {
			found := false
			for _, id := range mythos.ScenarioIDs {
				if id == scenarioID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("mythos card %s returned for scenario %s but does not list it", mythos.ID, scenarioID)
			}
		}
	}
}

func TestMythosForScenario_EachScenarioHas10Plus(t *testing.T) {
	scenarios := []string{
		"eldersign.museum.nightwatch",
		"eldersign.azathoth.madness",
		"eldersign.yig.serpent",
		"eldersign.cthulhu.depths",
		"eldersign.hastur.king",
	}

	for _, scenarioID := range scenarios {
		mythosCards := MythosForScenario(scenarioID)

		if len(mythosCards) < 10 {
			t.Errorf("scenario %s has only %d mythos cards (expected at least 10)", scenarioID, len(mythosCards))
		}
	}
}

func TestDefaultMythosCards_DoomImpactRanges(t *testing.T) {
	mythosCards := DefaultMythosCards()

	for _, mythos := range mythosCards {
		if mythos.DoomImpact < 0 || mythos.DoomImpact > 5 {
			t.Errorf("mythos card %s has doom impact %d (expected 0-5)", mythos.ID, mythos.DoomImpact)
		}
	}
}

func TestDefaultMythosCards_OmensHaveHighDoom(t *testing.T) {
	mythosCards := DefaultMythosCards()

	for _, mythos := range mythosCards {
		if mythos.Type == "omen" && mythos.DoomImpact < 2 {
			t.Errorf("omen %s has low doom impact %d (expected >=2 for omens)", mythos.ID, mythos.DoomImpact)
		}
	}
}
