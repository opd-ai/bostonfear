package rules

import (
	"testing"
)

func TestDefaultVictoryCondition(t *testing.T) {
	vc := DefaultVictoryCondition()
	if vc.RequiredElderSigns != 6 {
		t.Errorf("DefaultVictoryCondition() RequiredElderSigns = %d, want 6", vc.RequiredElderSigns)
	}
	if vc.GatesSealed != 0 {
		t.Errorf("DefaultVictoryCondition() GatesSealed = %d, want 0", vc.GatesSealed)
	}
}

func TestVictoryConditionIsVictorious(t *testing.T) {
	tests := []struct {
		name        string
		gatesSealed int
		required    int
		want        bool
	}{
		{"no gates sealed", 0, 6, false},
		{"some gates sealed", 3, 6, false},
		{"exactly enough gates", 6, 6, true},
		{"more than required", 8, 6, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc := VictoryCondition{
				RequiredElderSigns: tt.required,
				GatesSealed:        tt.gatesSealed,
			}
			got := vc.IsVictorious()
			if got != tt.want {
				t.Errorf("IsVictorious() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultDefeatCondition(t *testing.T) {
	dc := DefaultDefeatCondition()
	if dc.DoomLevel != 0 {
		t.Errorf("DefaultDefeatCondition() DoomLevel = %d, want 0", dc.DoomLevel)
	}
	if dc.MaxDoom != 12 {
		t.Errorf("DefaultDefeatCondition() MaxDoom = %d, want 12", dc.MaxDoom)
	}
}

func TestDefeatConditionIsDefeated(t *testing.T) {
	tests := []struct {
		name                  string
		doomLevel             int
		maxDoom               int
		investigatorsDefeated int
		totalInvestigators    int
		want                  bool
	}{
		{"no doom, no defeats", 0, 12, 0, 4, false},
		{"some doom", 5, 12, 0, 4, false},
		{"doom at maximum", 12, 12, 0, 4, true},
		{"doom above maximum", 15, 12, 0, 4, true},
		{"some investigators defeated", 5, 12, 2, 4, false},
		{"all investigators defeated", 5, 12, 4, 4, true},
		{"doom high and some defeated", 11, 12, 3, 4, false},
		{"doom at max and all defeated", 12, 12, 4, 4, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := DefeatCondition{
				DoomLevel:             tt.doomLevel,
				MaxDoom:               tt.maxDoom,
				InvestigatorsDefeated: tt.investigatorsDefeated,
				TotalInvestigators:    tt.totalInvestigators,
			}
			got := dc.IsDefeated()
			if got != tt.want {
				t.Errorf("IsDefeated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIncrementDoom(t *testing.T) {
	dc := DefeatCondition{
		DoomLevel: 5,
		MaxDoom:   12,
	}

	// Normal increment
	newLevel := dc.IncrementDoom(2)
	if newLevel != 7 {
		t.Errorf("IncrementDoom(2) returned %d, want 7", newLevel)
	}
	if dc.DoomLevel != 7 {
		t.Errorf("DoomLevel = %d, want 7", dc.DoomLevel)
	}

	// Increment beyond maximum
	newLevel = dc.IncrementDoom(10)
	if newLevel != 12 {
		t.Errorf("IncrementDoom(10) returned %d, want 12 (capped)", newLevel)
	}
	if dc.DoomLevel != 12 {
		t.Errorf("DoomLevel = %d, want 12 (capped)", dc.DoomLevel)
	}
}

func TestDecrementDoom(t *testing.T) {
	dc := DefeatCondition{
		DoomLevel: 5,
		MaxDoom:   12,
	}

	// Normal decrement
	newLevel := dc.DecrementDoom(2)
	if newLevel != 3 {
		t.Errorf("DecrementDoom(2) returned %d, want 3", newLevel)
	}
	if dc.DoomLevel != 3 {
		t.Errorf("DoomLevel = %d, want 3", dc.DoomLevel)
	}

	// Decrement below zero
	newLevel = dc.DecrementDoom(10)
	if newLevel != 0 {
		t.Errorf("DecrementDoom(10) returned %d, want 0 (capped)", newLevel)
	}
	if dc.DoomLevel != 0 {
		t.Errorf("DoomLevel = %d, want 0 (capped)", dc.DoomLevel)
	}
}
