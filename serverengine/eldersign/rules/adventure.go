package rules

// TaskType defines the types of tasks on adventure cards.
type TaskType string

const (
	// TaskTypeInvestigation requires specific colored dice results.
	TaskTypeInvestigation TaskType = "investigation"

	// TaskTypeLore requires Lore dice results.
	TaskTypeLore TaskType = "lore"

	// TaskTypePenal applies penalties (Terror or Peril effects).
	TaskTypePenal TaskType = "penal"
)

// Task represents a single requirement on an adventure card.
// Tasks must be completed in sequence to claim the adventure.
type Task struct {
	// Type identifies the task category (investigation, lore, penal).
	Type TaskType

	// RequiredRed is the count of red investigation dice needed.
	RequiredRed int

	// RequiredGreen is the count of green investigation dice needed.
	RequiredGreen int

	// RequiredYellow is the count of yellow investigation dice needed.
	RequiredYellow int

	// RequiredLore is the count of lore dice needed.
	RequiredLore int

	// Penalty describes the effect if this task fails (stamina/sanity loss, monster spawn, etc.).
	Penalty string
}

// IsSatisfied checks if the dice pool satisfies this task's requirements.
func (t Task) IsSatisfied(dm *DiceMechanics) bool {
	redCount := dm.CountResultType(DiceResultRed)
	greenCount := dm.CountResultType(DiceResultGreen)
	yellowCount := dm.CountResultType(DiceResultYellow)
	loreCount := dm.CountResultType(DiceResultLore)

	return redCount >= t.RequiredRed &&
		greenCount >= t.RequiredGreen &&
		yellowCount >= t.RequiredYellow &&
		loreCount >= t.RequiredLore
}

// AdventureCard represents a location-based challenge in Elder Sign.
// Each adventure has multiple tasks that must be completed sequentially.
type AdventureCard struct {
	// ID is the unique adventure identifier.
	ID string

	// Name is the adventure display title.
	Name string

	// Tasks are the sequential requirements to complete this adventure.
	Tasks []Task

	// Reward describes what the investigator gains on success.
	// Examples: "1 Elder Sign token", "2 Clue tokens + 1 item", "3 Stamina"
	Reward string

	// FailurePenalty describes the consequence if the adventure is abandoned.
	FailurePenalty string

	// Difficulty estimates the challenge level (1-5 scale, informational).
	Difficulty int
}

// IsComplete checks if all tasks have been satisfied.
func (a AdventureCard) IsComplete(dm *DiceMechanics) bool {
	for _, task := range a.Tasks {
		if !task.IsSatisfied(dm) {
			return false
		}
	}
	return true
}

// NextUnsatisfiedTask returns the first task that is not yet completed.
// Returns nil if all tasks are satisfied.
func (a AdventureCard) NextUnsatisfiedTask(dm *DiceMechanics) *Task {
	for i := range a.Tasks {
		if !a.Tasks[i].IsSatisfied(dm) {
			return &a.Tasks[i]
		}
	}
	return nil
}

// DefaultAdventureCard returns a starter adventure for integration testing.
func DefaultAdventureCard() AdventureCard {
	return AdventureCard{
		ID:   "adv.nightwatch.foyer",
		Name: "Museum Foyer Investigation",
		Tasks: []Task{
			{
				Type:          TaskTypeInvestigation,
				RequiredRed:   2,
				RequiredGreen: 1,
				Penalty:       "Lose 1 Stamina",
			},
			{
				Type:         TaskTypeLore,
				RequiredLore: 1,
				Penalty:      "Lose 1 Sanity",
			},
		},
		Reward:         "1 Elder Sign token",
		FailurePenalty: "Advance Doom by 1",
		Difficulty:     2,
	}
}
