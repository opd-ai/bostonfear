package rules

import "testing"

func TestNewResources(t *testing.T) {
	r := NewResources(10, 10)
	if r.Health != 10 || r.MaxHealth != 10 {
		t.Errorf("expected health 10/10, got %d/%d", r.Health, r.MaxHealth)
	}
	if r.Sanity != 10 || r.MaxSanity != 10 {
		t.Errorf("expected sanity 10/10, got %d/%d", r.Sanity, r.MaxSanity)
	}
	if r.Clues != 0 {
		t.Errorf("expected 0 clues, got %d", r.Clues)
	}
}

func TestRestoreHealth(t *testing.T) {
	r := NewResources(10, 10)
	r.Health = 5

	restored := r.RestoreHealth(3)
	if restored != 3 {
		t.Errorf("expected 3 health restored, got %d", restored)
	}
	if r.Health != 8 {
		t.Errorf("expected health 8, got %d", r.Health)
	}

	// Test capping at maximum
	restored = r.RestoreHealth(5)
	if restored != 2 {
		t.Errorf("expected 2 health restored (capped), got %d", restored)
	}
	if r.Health != 10 {
		t.Errorf("expected health 10, got %d", r.Health)
	}
}

func TestLoseHealth(t *testing.T) {
	r := NewResources(10, 10)

	eliminated := r.LoseHealth(5)
	if eliminated {
		t.Error("should not be eliminated at 5 health")
	}
	if r.Health != 5 {
		t.Errorf("expected health 5, got %d", r.Health)
	}

	eliminated = r.LoseHealth(5)
	if !eliminated {
		t.Error("should be eliminated at 0 health")
	}
	if r.Health != 0 {
		t.Errorf("expected health 0, got %d", r.Health)
	}

	// Test cannot go below 0
	eliminated = r.LoseHealth(5)
	if r.Health != 0 {
		t.Errorf("expected health to stay at 0, got %d", r.Health)
	}
}

func TestRestoreSanity(t *testing.T) {
	r := NewResources(10, 10)
	r.Sanity = 3

	restored := r.RestoreSanity(4)
	if restored != 4 {
		t.Errorf("expected 4 sanity restored, got %d", restored)
	}
	if r.Sanity != 7 {
		t.Errorf("expected sanity 7, got %d", r.Sanity)
	}
}

func TestLoseSanity(t *testing.T) {
	r := NewResources(10, 10)

	eliminated := r.LoseSanity(10)
	if !eliminated {
		t.Error("should be eliminated at 0 sanity")
	}
	if r.Sanity != 0 {
		t.Errorf("expected sanity 0, got %d", r.Sanity)
	}
}

func TestGainSpendClues(t *testing.T) {
	r := NewResources(10, 10)

	r.GainClues(3)
	if r.Clues != 3 {
		t.Errorf("expected 3 clues, got %d", r.Clues)
	}

	err := r.SpendClues(2)
	if err != nil {
		t.Errorf("unexpected error spending clues: %v", err)
	}
	if r.Clues != 1 {
		t.Errorf("expected 1 clue, got %d", r.Clues)
	}

	err = r.SpendClues(2)
	if err == nil {
		t.Error("expected error spending more clues than available")
	}
}

func TestGainSpendMoney(t *testing.T) {
	r := NewResources(10, 10)

	r.GainMoney(5)
	if r.Money != 5 {
		t.Errorf("expected 5 money, got %d", r.Money)
	}

	err := r.SpendMoney(3)
	if err != nil {
		t.Errorf("unexpected error spending money: %v", err)
	}
	if r.Money != 2 {
		t.Errorf("expected 2 money, got %d", r.Money)
	}

	err = r.SpendMoney(3)
	if err == nil {
		t.Error("expected error spending more money than available")
	}
}

func TestGainSpendTickets(t *testing.T) {
	r := NewResources(10, 10)

	r.GainTickets(2)
	if r.Tickets != 2 {
		t.Errorf("expected 2 tickets, got %d", r.Tickets)
	}

	err := r.SpendTicket()
	if err != nil {
		t.Errorf("unexpected error spending ticket: %v", err)
	}
	if r.Tickets != 1 {
		t.Errorf("expected 1 ticket, got %d", r.Tickets)
	}

	r.SpendTicket()
	err = r.SpendTicket()
	if err == nil {
		t.Error("expected error spending ticket when none available")
	}
}

func TestGainSpendElderSign(t *testing.T) {
	r := NewResources(10, 10)

	r.GainElderSign()
	if r.ElderSigns != 1 {
		t.Errorf("expected 1 elder sign, got %d", r.ElderSigns)
	}

	err := r.SpendElderSign()
	if err != nil {
		t.Errorf("unexpected error spending elder sign: %v", err)
	}
	if r.ElderSigns != 0 {
		t.Errorf("expected 0 elder signs, got %d", r.ElderSigns)
	}

	err = r.SpendElderSign()
	if err == nil {
		t.Error("expected error spending elder sign when none available")
	}
}

func TestIsEliminated(t *testing.T) {
	r := NewResources(10, 10)

	if r.IsEliminated() {
		t.Error("should not be eliminated at full health/sanity")
	}

	r.Health = 0
	if !r.IsEliminated() {
		t.Error("should be eliminated at 0 health")
	}

	r.Health = 10
	r.Sanity = 0
	if !r.IsEliminated() {
		t.Error("should be eliminated at 0 sanity")
	}
}

func TestGetHealthRatio(t *testing.T) {
	r := NewResources(10, 10)
	r.Health = 5

	ratio := r.GetHealthRatio()
	if ratio != 0.5 {
		t.Errorf("expected health ratio 0.5, got %f", ratio)
	}
}

func TestGetSanityRatio(t *testing.T) {
	r := NewResources(10, 10)
	r.Sanity = 7

	ratio := r.GetSanityRatio()
	if ratio != 0.7 {
		t.Errorf("expected sanity ratio 0.7, got %f", ratio)
	}
}

func TestDefaultRestAction(t *testing.T) {
	action := DefaultRestAction()
	if action.HealthRestore <= 0 || action.SanityRestore <= 0 {
		t.Error("expected positive restore values")
	}
}

func TestCityRestAction(t *testing.T) {
	tests := []struct {
		city       City
		minRestore int
	}{
		{CityArkham, 3},      // Major city
		{CityCairo, 2},       // Moderate city
		{CityAntarctica, 1},  // Remote location
		{CityBuenosAires, 1}, // Default
	}

	for _, tt := range tests {
		action := CityRestAction(tt.city)
		if action.HealthRestore < tt.minRestore {
			t.Errorf("city %s health restore %d below expected minimum %d",
				tt.city, action.HealthRestore, tt.minRestore)
		}
		if action.SanityRestore < tt.minRestore {
			t.Errorf("city %s sanity restore %d below expected minimum %d",
				tt.city, action.SanityRestore, tt.minRestore)
		}
	}
}

func TestApplyRestAction(t *testing.T) {
	r := NewResources(10, 10)
	r.Health = 5
	r.Sanity = 6

	action := RestAction{
		HealthRestore: 3,
		SanityRestore: 2,
	}

	r.ApplyRestAction(action)
	if r.Health != 8 {
		t.Errorf("expected health 8 after rest, got %d", r.Health)
	}
	if r.Sanity != 8 {
		t.Errorf("expected sanity 8 after rest, got %d", r.Sanity)
	}
}

func TestResourcesString(t *testing.T) {
	r := NewResources(10, 10)
	r.Clues = 3
	r.Money = 5

	str := r.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
	// Just verify it doesn't panic and returns something
}
