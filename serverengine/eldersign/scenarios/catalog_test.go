package scenarios

import "testing"

func TestDefaultCatalog_ContainsRequiredScenarios(t *testing.T) {
	catalog := DefaultCatalog()

	// Verify we have at least 5 scenarios (original + 4 Ancient Ones)
	if len(catalog) < 5 {
		t.Errorf("expected at least 5 scenarios, got %d", len(catalog))
	}

	// Verify all required Ancient One scenarios are present
	requiredIDs := []string{
		"eldersign.azathoth.madness",
		"eldersign.yig.serpent",
		"eldersign.cthulhu.depths",
		"eldersign.hastur.king",
	}

	for _, reqID := range requiredIDs {
		found := false
		for _, scenario := range catalog {
			if scenario.ID == reqID {
				found = true
				if !scenario.Enabled {
					t.Errorf("scenario %s should be enabled", reqID)
				}
				if scenario.Name == "" {
					t.Errorf("scenario %s missing Name", reqID)
				}
				if scenario.WinGoal == "" {
					t.Errorf("scenario %s missing WinGoal", reqID)
				}
				if scenario.LossGoal == "" {
					t.Errorf("scenario %s missing LossGoal", reqID)
				}
				break
			}
		}
		if !found {
			t.Errorf("required scenario %s not found in catalog", reqID)
		}
	}
}

func TestResolveDefault_ReturnsFirstEnabledScenario(t *testing.T) {
	catalog := DefaultCatalog()
	scenario, err := ResolveDefault(catalog)
	if err != nil {
		t.Fatalf("ResolveDefault failed: %v", err)
	}

	if scenario.ID == "" {
		t.Error("resolved scenario has empty ID")
	}

	if !scenario.Enabled {
		t.Error("resolved scenario should be enabled")
	}
}

func TestResolveDefault_ErrorsWhenNoEnabledScenarios(t *testing.T) {
	catalog := []Template{
		{ID: "test.disabled", Name: "Test", Enabled: false, WinGoal: "win", LossGoal: "lose"},
	}

	_, err := ResolveDefault(catalog)
	if err == nil {
		t.Error("expected error when no enabled scenarios, got nil")
	}
}
