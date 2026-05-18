package rules

import (
	"testing"
)

func TestTaskIsSatisfied(t *testing.T) {
	tests := []struct {
		name string
		task Task
		dice *DiceMechanics
		want bool
	}{
		{
			name: "satisfied investigation task",
			task: Task{
				Type:          TaskTypeInvestigation,
				RequiredRed:   2,
				RequiredGreen: 1,
			},
			dice: &DiceMechanics{
				LockedResults: []DiceResult{DiceResultRed, DiceResultRed, DiceResultGreen},
			},
			want: true,
		},
		{
			name: "unsatisfied investigation task",
			task: Task{
				Type:          TaskTypeInvestigation,
				RequiredRed:   3,
				RequiredGreen: 1,
			},
			dice: &DiceMechanics{
				LockedResults: []DiceResult{DiceResultRed, DiceResultGreen},
			},
			want: false,
		},
		{
			name: "satisfied lore task",
			task: Task{
				Type:         TaskTypeLore,
				RequiredLore: 2,
			},
			dice: &DiceMechanics{
				LockedResults: []DiceResult{DiceResultLore, DiceResultLore},
			},
			want: true,
		},
		{
			name: "mixed locked and active results",
			task: Task{
				Type:           TaskTypeInvestigation,
				RequiredRed:    2,
				RequiredYellow: 1,
			},
			dice: &DiceMechanics{
				LockedResults: []DiceResult{DiceResultRed},
				ActiveResults: []DiceResult{DiceResultRed, DiceResultYellow},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.task.IsSatisfied(tt.dice)
			if got != tt.want {
				t.Errorf("Task.IsSatisfied() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdventureCardIsComplete(t *testing.T) {
	adventure := AdventureCard{
		ID: "test",
		Tasks: []Task{
			{RequiredRed: 2},
			{RequiredGreen: 1},
		},
	}

	tests := []struct {
		name string
		dice *DiceMechanics
		want bool
	}{
		{
			name: "all tasks satisfied",
			dice: &DiceMechanics{
				LockedResults: []DiceResult{
					DiceResultRed, DiceResultRed, DiceResultGreen,
				},
			},
			want: true,
		},
		{
			name: "first task unsatisfied",
			dice: &DiceMechanics{
				LockedResults: []DiceResult{DiceResultRed, DiceResultGreen},
			},
			want: false,
		},
		{
			name: "second task unsatisfied",
			dice: &DiceMechanics{
				LockedResults: []DiceResult{DiceResultRed, DiceResultRed},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adventure.IsComplete(tt.dice)
			if got != tt.want {
				t.Errorf("AdventureCard.IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdventureCardNextUnsatisfiedTask(t *testing.T) {
	adventure := AdventureCard{
		ID: "test",
		Tasks: []Task{
			{RequiredRed: 2},
			{RequiredGreen: 1},
			{RequiredLore: 1},
		},
	}

	tests := []struct {
		name      string
		dice      *DiceMechanics
		wantIndex int // -1 means nil (all satisfied)
	}{
		{
			name: "first task unsatisfied",
			dice: &DiceMechanics{
				LockedResults: []DiceResult{DiceResultRed},
			},
			wantIndex: 0,
		},
		{
			name: "second task unsatisfied",
			dice: &DiceMechanics{
				LockedResults: []DiceResult{DiceResultRed, DiceResultRed},
			},
			wantIndex: 1,
		},
		{
			name: "third task unsatisfied",
			dice: &DiceMechanics{
				LockedResults: []DiceResult{DiceResultRed, DiceResultRed, DiceResultGreen},
			},
			wantIndex: 2,
		},
		{
			name: "all tasks satisfied",
			dice: &DiceMechanics{
				LockedResults: []DiceResult{
					DiceResultRed, DiceResultRed, DiceResultGreen, DiceResultLore,
				},
			},
			wantIndex: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adventure.NextUnsatisfiedTask(tt.dice)
			if tt.wantIndex == -1 {
				if got != nil {
					t.Errorf("NextUnsatisfiedTask() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("NextUnsatisfiedTask() = nil, want task at index %d", tt.wantIndex)
				} else if got != &adventure.Tasks[tt.wantIndex] {
					t.Errorf("NextUnsatisfiedTask() returned wrong task")
				}
			}
		})
	}
}

func TestDefaultAdventureCard(t *testing.T) {
	adventure := DefaultAdventureCard()

	if adventure.ID != "adv.nightwatch.foyer" {
		t.Errorf("DefaultAdventureCard() ID = %s, want adv.nightwatch.foyer", adventure.ID)
	}
	if len(adventure.Tasks) != 2 {
		t.Errorf("DefaultAdventureCard() has %d tasks, want 2", len(adventure.Tasks))
	}
	if adventure.Difficulty != 2 {
		t.Errorf("DefaultAdventureCard() Difficulty = %d, want 2", adventure.Difficulty)
	}
}

func TestTaskTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		taskType TaskType
		want     string
	}{
		{"Investigation", TaskTypeInvestigation, "investigation"},
		{"Lore", TaskTypeLore, "lore"},
		{"Penal", TaskTypePenal, "penal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.taskType) != tt.want {
				t.Errorf("%s constant = %v, want %v", tt.name, string(tt.taskType), tt.want)
			}
		})
	}
}
