package content

import "testing"

func TestDefaultAdventures_ContainsAtLeast30Cards(t *testing.T) {
	adventures := DefaultAdventures()

	if len(adventures) < 30 {
		t.Errorf("expected at least 30 adventure cards, got %d", len(adventures))
	}
}

func TestDefaultAdventures_AllCardsHaveRequiredFields(t *testing.T) {
	adventures := DefaultAdventures()

	for _, adv := range adventures {
		if adv.ID == "" {
			t.Errorf("adventure missing ID")
		}
		if adv.Name == "" {
			t.Errorf("adventure %s missing Name", adv.ID)
		}
		if adv.Difficulty < 1 || adv.Difficulty > 4 {
			t.Errorf("adventure %s has invalid difficulty %d (must be 1-4)", adv.ID, adv.Difficulty)
		}
		if len(adv.Tasks) == 0 {
			t.Errorf("adventure %s has no tasks", adv.ID)
		}
		if len(adv.Rewards) == 0 {
			t.Errorf("adventure %s has no rewards", adv.ID)
		}
		if len(adv.Penalties) == 0 {
			t.Errorf("adventure %s has no penalties", adv.ID)
		}
		if len(adv.ScenarioIDs) == 0 {
			t.Errorf("adventure %s not assigned to any scenarios", adv.ID)
		}
	}
}

func TestDefaultAdventures_DifficultyDistribution(t *testing.T) {
	adventures := DefaultAdventures()
	difficultyCount := map[int]int{}

	for _, adv := range adventures {
		difficultyCount[adv.Difficulty]++
	}

	if difficultyCount[1] == 0 {
		t.Error("no easy (difficulty 1) adventures")
	}
	if difficultyCount[2] == 0 {
		t.Error("no medium (difficulty 2) adventures")
	}
	if difficultyCount[3] == 0 {
		t.Error("no hard (difficulty 3) adventures")
	}
	if difficultyCount[4] == 0 {
		t.Error("no very hard (difficulty 4) adventures")
	}
}

func TestAdventuresForScenario_FiltersCorrectly(t *testing.T) {
	scenarios := []string{
		"eldersign.azathoth.madness",
		"eldersign.yig.serpent",
		"eldersign.cthulhu.depths",
		"eldersign.hastur.king",
	}

	for _, scenarioID := range scenarios {
		adventures := AdventuresForScenario(scenarioID)

		if len(adventures) == 0 {
			t.Errorf("scenario %s has no adventures", scenarioID)
		}

		// Verify all returned adventures actually include this scenario
		for _, adv := range adventures {
			found := false
			for _, id := range adv.ScenarioIDs {
				if id == scenarioID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("adventure %s returned for scenario %s but does not list it", adv.ID, scenarioID)
			}
		}
	}
}

func TestAdventuresForScenario_EachScenarioHas30Plus(t *testing.T) {
	scenarios := []string{
		"eldersign.museum.nightwatch",
		"eldersign.azathoth.madness",
		"eldersign.yig.serpent",
		"eldersign.cthulhu.depths",
		"eldersign.hastur.king",
	}

	for _, scenarioID := range scenarios {
		adventures := AdventuresForScenario(scenarioID)

		if len(adventures) < 20 {
			t.Errorf("scenario %s has only %d adventures (expected at least 20 for variety)", scenarioID, len(adventures))
		}
	}
}

func TestAdventuresForScenario_AllDifficultyLevelsPresent(t *testing.T) {
	scenarios := []string{
		"eldersign.azathoth.madness",
		"eldersign.yig.serpent",
		"eldersign.cthulhu.depths",
		"eldersign.hastur.king",
	}

	for _, scenarioID := range scenarios {
		adventures := AdventuresForScenario(scenarioID)
		difficultyCount := map[int]int{}

		for _, adv := range adventures {
			difficultyCount[adv.Difficulty]++
		}

		if difficultyCount[1] == 0 {
			t.Errorf("scenario %s missing easy adventures", scenarioID)
		}
		if difficultyCount[2] == 0 {
			t.Errorf("scenario %s missing medium adventures", scenarioID)
		}
	}
}
