package rules

import "testing"

func TestNewMysteryDeck(t *testing.T) {
	md := NewMysteryDeck(3)
	if md == nil {
		t.Fatal("expected mystery deck to be created")
	}
	if md.MysteriesToSolve != 3 {
		t.Errorf("expected 3 mysteries to solve, got %d", md.MysteriesToSolve)
	}
	if md.ActiveMystery != nil {
		t.Error("expected no active mystery initially")
	}
	if len(md.CompletedMysteries) != 0 {
		t.Error("expected no completed mysteries initially")
	}
}

func TestActivateMystery(t *testing.T) {
	md := NewMysteryDeck(3)
	mystery := Mystery{
		ID:          "test-mystery",
		Name:        "Test Mystery",
		Description: "A test mystery",
		Stages: []MysteryStage{
			{StageNumber: 1, Description: "Stage 1", Completed: false},
		},
	}

	err := md.ActivateMystery(mystery)
	if err != nil {
		t.Errorf("unexpected error activating mystery: %v", err)
	}
	if md.ActiveMystery == nil {
		t.Fatal("expected active mystery to be set")
	}
	if md.ActiveMystery.ID != "test-mystery" {
		t.Errorf("expected mystery ID 'test-mystery', got %s", md.ActiveMystery.ID)
	}

	// Test activating second mystery fails
	err = md.ActivateMystery(mystery)
	if err == nil {
		t.Error("expected error when activating second mystery")
	}
}

func TestGetCurrentStage(t *testing.T) {
	md := NewMysteryDeck(3)
	mystery := Mystery{
		ID:   "test-mystery",
		Name: "Test Mystery",
		Stages: []MysteryStage{
			{StageNumber: 1, Description: "Stage 1", Completed: false},
			{StageNumber: 2, Description: "Stage 2", Completed: false},
			{StageNumber: 3, Description: "Stage 3", Completed: false},
		},
	}
	md.ActivateMystery(mystery)

	// Test getting first stage
	stage := md.GetCurrentStage()
	if stage == nil {
		t.Fatal("expected current stage")
	}
	if stage.StageNumber != 1 {
		t.Errorf("expected stage 1, got stage %d", stage.StageNumber)
	}

	// Complete first stage
	stage.Completed = true

	// Test getting second stage
	stage = md.GetCurrentStage()
	if stage == nil {
		t.Fatal("expected current stage after completing first")
	}
	if stage.StageNumber != 2 {
		t.Errorf("expected stage 2, got stage %d", stage.StageNumber)
	}
}

func TestCompleteCurrentStage(t *testing.T) {
	md := NewMysteryDeck(3)
	mystery := Mystery{
		ID:   "test-mystery",
		Name: "Test Mystery",
		Stages: []MysteryStage{
			{StageNumber: 1, Description: "Stage 1", Completed: false},
		},
	}
	md.ActivateMystery(mystery)

	err := md.CompleteCurrentStage()
	if err != nil {
		t.Errorf("unexpected error completing stage: %v", err)
	}

	stage := md.GetCurrentStage()
	if stage != nil {
		t.Error("expected no current stage after completing only stage")
	}
}

func TestIsMysteryComplete(t *testing.T) {
	md := NewMysteryDeck(3)
	mystery := Mystery{
		ID:   "test-mystery",
		Name: "Test Mystery",
		Stages: []MysteryStage{
			{StageNumber: 1, Description: "Stage 1", Completed: false},
			{StageNumber: 2, Description: "Stage 2", Completed: false},
		},
	}
	md.ActivateMystery(mystery)

	if md.IsMysteryComplete() {
		t.Error("expected mystery not to be complete")
	}

	// Complete all stages
	md.ActiveMystery.Stages[0].Completed = true
	md.ActiveMystery.Stages[1].Completed = true

	if !md.IsMysteryComplete() {
		t.Error("expected mystery to be complete")
	}
}

func TestFinalizeMystery(t *testing.T) {
	md := NewMysteryDeck(3)
	mystery := Mystery{
		ID:   "test-mystery",
		Name: "Test Mystery",
		Stages: []MysteryStage{
			{StageNumber: 1, Description: "Stage 1", Completed: true},
		},
	}
	md.ActivateMystery(mystery)

	completed, err := md.FinalizeMystery()
	if err != nil {
		t.Errorf("unexpected error finalizing mystery: %v", err)
	}
	if completed == nil {
		t.Fatal("expected completed mystery")
	}
	if completed.ID != "test-mystery" {
		t.Errorf("expected mystery ID 'test-mystery', got %s", completed.ID)
	}
	if md.ActiveMystery != nil {
		t.Error("expected active mystery to be cleared")
	}
	if len(md.CompletedMysteries) != 1 {
		t.Errorf("expected 1 completed mystery, got %d", len(md.CompletedMysteries))
	}
}

func TestIsVictoryConditionMet(t *testing.T) {
	md := NewMysteryDeck(3)

	if md.IsVictoryConditionMet() {
		t.Error("expected victory condition not met initially")
	}

	// Add completed mysteries
	for i := 0; i < 3; i++ {
		md.CompletedMysteries = append(md.CompletedMysteries, Mystery{
			ID: string(rune('A' + i)),
		})
	}

	if !md.IsVictoryConditionMet() {
		t.Error("expected victory condition met after 3 mysteries")
	}
}

func TestGetProgress(t *testing.T) {
	md := NewMysteryDeck(3)

	solved, required := md.GetProgress()
	if solved != 0 || required != 3 {
		t.Errorf("expected 0/3, got %d/%d", solved, required)
	}

	md.CompletedMysteries = append(md.CompletedMysteries, Mystery{ID: "1"})
	solved, required = md.GetProgress()
	if solved != 1 || required != 3 {
		t.Errorf("expected 1/3, got %d/%d", solved, required)
	}
}

func TestCanSpendClues(t *testing.T) {
	tests := []struct {
		name           string
		requirements   MysteryRequirements
		location       City
		cluesAvailable int
		expected       bool
	}{
		{
			name: "enough clues, any location",
			requirements: MysteryRequirements{
				CluesRequired:    2,
				LocationRequired: "",
			},
			location:       CityArkham,
			cluesAvailable: 3,
			expected:       true,
		},
		{
			name: "not enough clues",
			requirements: MysteryRequirements{
				CluesRequired:    3,
				LocationRequired: "",
			},
			location:       CityArkham,
			cluesAvailable: 2,
			expected:       false,
		},
		{
			name: "wrong location",
			requirements: MysteryRequirements{
				CluesRequired:    2,
				LocationRequired: CityLondon,
			},
			location:       CityArkham,
			cluesAvailable: 3,
			expected:       false,
		},
		{
			name: "correct location",
			requirements: MysteryRequirements{
				CluesRequired:    2,
				LocationRequired: CityLondon,
			},
			location:       CityLondon,
			cluesAvailable: 3,
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.requirements.CanSpendClues(tt.location, tt.cluesAvailable)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMysteryString(t *testing.T) {
	mystery := Mystery{
		ID:   "test",
		Name: "The Ancient Threat",
		Stages: []MysteryStage{
			{StageNumber: 1},
			{StageNumber: 2},
			{StageNumber: 3},
		},
	}

	str := mystery.String()
	if str != "Mystery: The Ancient Threat (3 stages)" {
		t.Errorf("unexpected string: %s", str)
	}
}
