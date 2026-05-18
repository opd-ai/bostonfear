package rules

import (
	"testing"
)

func TestNewDiceMechanics(t *testing.T) {
	dm := NewDiceMechanics()
	if dm.AvailableDice != 6 {
		t.Errorf("NewDiceMechanics() AvailableDice = %d, want 6", dm.AvailableDice)
	}
	if len(dm.LockedResults) != 0 {
		t.Errorf("NewDiceMechanics() LockedResults length = %d, want 0", len(dm.LockedResults))
	}
	if len(dm.ActiveResults) != 0 {
		t.Errorf("NewDiceMechanics() ActiveResults length = %d, want 0", len(dm.ActiveResults))
	}
}

func TestLockDie(t *testing.T) {
	dm := NewDiceMechanics()
	dm.ActiveResults = []DiceResult{DiceResultRed, DiceResultGreen, DiceResultLore}

	// Lock first die
	err := dm.LockDie(0)
	if err != nil {
		t.Errorf("LockDie(0) returned error: %v", err)
	}
	if len(dm.LockedResults) != 1 {
		t.Errorf("LockDie(0) LockedResults length = %d, want 1", len(dm.LockedResults))
	}
	if dm.LockedResults[0] != DiceResultRed {
		t.Errorf("LockDie(0) locked result = %v, want %v", dm.LockedResults[0], DiceResultRed)
	}
	if len(dm.ActiveResults) != 2 {
		t.Errorf("LockDie(0) ActiveResults length = %d, want 2", len(dm.ActiveResults))
	}
}

func TestLockDie_InvalidIndex(t *testing.T) {
	dm := NewDiceMechanics()
	dm.ActiveResults = []DiceResult{DiceResultRed}

	tests := []struct {
		name  string
		index int
	}{
		{"negative index", -1},
		{"index too large", 5},
		{"index equal to length", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := dm.LockDie(tt.index)
			if err != ErrInvalidDieIndex {
				t.Errorf("LockDie(%d) error = %v, want %v", tt.index, err, ErrInvalidDieIndex)
			}
		})
	}
}

func TestRollActiveDice(t *testing.T) {
	dm := NewDiceMechanics()

	// First roll should roll all 6 dice
	results, terrorCount := dm.RollActiveDice()
	if len(results) != 6 {
		t.Errorf("RollActiveDice() returned %d results, want 6", len(results))
	}
	if terrorCount < 0 {
		t.Errorf("RollActiveDice() terrorCount = %d, want >= 0", terrorCount)
	}

	// Lock one die and roll again
	dm.LockDie(0)
	results, _ = dm.RollActiveDice()
	if len(results) != 5 {
		t.Errorf("RollActiveDice() after lock returned %d results, want 5", len(results))
	}
}

func TestCountResultType(t *testing.T) {
	dm := NewDiceMechanics()
	dm.LockedResults = []DiceResult{DiceResultRed, DiceResultGreen, DiceResultRed}
	dm.ActiveResults = []DiceResult{DiceResultRed, DiceResultLore}

	count := dm.CountResultType(DiceResultRed)
	if count != 3 {
		t.Errorf("CountResultType(Red) = %d, want 3", count)
	}

	count = dm.CountResultType(DiceResultGreen)
	if count != 1 {
		t.Errorf("CountResultType(Green) = %d, want 1", count)
	}

	count = dm.CountResultType(DiceResultLore)
	if count != 1 {
		t.Errorf("CountResultType(Lore) = %d, want 1", count)
	}

	count = dm.CountResultType(DiceResultTerror)
	if count != 0 {
		t.Errorf("CountResultType(Terror) = %d, want 0", count)
	}
}

func TestReset(t *testing.T) {
	dm := NewDiceMechanics()
	dm.LockedResults = []DiceResult{DiceResultRed, DiceResultGreen}
	dm.ActiveResults = []DiceResult{DiceResultLore}

	dm.Reset()

	if len(dm.LockedResults) != 0 {
		t.Errorf("Reset() LockedResults length = %d, want 0", len(dm.LockedResults))
	}
	if len(dm.ActiveResults) != 0 {
		t.Errorf("Reset() ActiveResults length = %d, want 0", len(dm.ActiveResults))
	}
}

func TestDiceResultConstants(t *testing.T) {
	tests := []struct {
		name   string
		result DiceResult
		want   string
	}{
		{"Terror", DiceResultTerror, "terror"},
		{"Peril", DiceResultPeril, "peril"},
		{"Lore", DiceResultLore, "lore"},
		{"Red", DiceResultRed, "red"},
		{"Green", DiceResultGreen, "green"},
		{"Yellow", DiceResultYellow, "yellow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.result) != tt.want {
				t.Errorf("%s constant = %v, want %v", tt.name, string(tt.result), tt.want)
			}
		})
	}
}
