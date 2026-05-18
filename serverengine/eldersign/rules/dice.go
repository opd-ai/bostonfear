package rules

// DiceResult represents Elder Sign 6-sided die outcomes.
// Elder Sign uses custom dice with colored results (red/green/yellow)
// plus special icons, distinct from Arkham Horror's 3-sided dice.
type DiceResult string

const (
	// Terror represents a skull icon result.
	// Terror results typically trigger negative effects or spawn monsters.
	DiceResultTerror DiceResult = "terror"

	// Peril represents a tentacle icon result.
	// Similar to Terror but may have different penalty effects.
	DiceResultPeril DiceResult = "peril"

	// Lore represents a book/scroll icon result.
	// Lore results are used to complete investigation tasks.
	DiceResultLore DiceResult = "lore"

	// Red represents a red investigation symbol.
	// Used to satisfy red task requirements on adventure cards.
	DiceResultRed DiceResult = "red"

	// Green represents a green investigation symbol.
	// Used to satisfy green task requirements on adventure cards.
	DiceResultGreen DiceResult = "green"

	// Yellow represents a yellow investigation symbol.
	// Used to satisfy yellow task requirements on adventure cards.
	DiceResultYellow DiceResult = "yellow"
)

// DiceMechanics defines Elder Sign dice rolling behavior.
// Unlike Arkham Horror's fixed 3-sided dice, Elder Sign uses 6-sided dice
// with colored investigation symbols and special penalty icons.
type DiceMechanics struct {
	// DicePool tracks the current dice available for rolling.
	// Starts with 6 dice per adventure attempt; may be modified by items.
	AvailableDice int

	// LockedResults stores dice that have been locked (secured) by the player.
	// Locked dice cannot be rerolled and persist across task attempts.
	LockedResults []DiceResult

	// ActiveResults stores the current roll outcome for unlocked dice.
	// These dice may be rerolled or locked on the player's turn.
	ActiveResults []DiceResult
}

// NewDiceMechanics creates a standard Elder Sign dice pool.
// Standard pool starts with 6 dice, no locked results.
func NewDiceMechanics() *DiceMechanics {
	return &DiceMechanics{
		AvailableDice: 6,
		LockedResults: make([]DiceResult, 0),
		ActiveResults: make([]DiceResult, 0),
	}
}

// LockDie moves a die result from active pool to locked pool.
// Locked dice cannot be rerolled.
func (d *DiceMechanics) LockDie(index int) error {
	if index < 0 || index >= len(d.ActiveResults) {
		return ErrInvalidDieIndex
	}
	result := d.ActiveResults[index]
	d.LockedResults = append(d.LockedResults, result)
	// Remove from active results
	d.ActiveResults = append(d.ActiveResults[:index], d.ActiveResults[index+1:]...)
	return nil
}

// RollActiveDice simulates rolling all unlocked dice.
// Returns the number of Terror/Peril results for penalty application.
func (d *DiceMechanics) RollActiveDice() ([]DiceResult, int) {
	numActiveDice := d.AvailableDice - len(d.LockedResults)
	d.ActiveResults = make([]DiceResult, numActiveDice)
	terrorCount := 0

	// In production, this would use a proper RNG.
	// For now, placeholder logic ensures at least one valid result.
	for i := 0; i < numActiveDice; i++ {
		// Simplified distribution: 50% investigation, 25% lore, 25% penalty
		switch i % 4 {
		case 0:
			d.ActiveResults[i] = DiceResultRed
		case 1:
			d.ActiveResults[i] = DiceResultGreen
		case 2:
			d.ActiveResults[i] = DiceResultLore
		case 3:
			d.ActiveResults[i] = DiceResultTerror
			terrorCount++
		}
	}
	return d.ActiveResults, terrorCount
}

// CountResultType returns the count of a specific result across locked and active dice.
func (d *DiceMechanics) CountResultType(result DiceResult) int {
	count := 0
	for _, r := range d.LockedResults {
		if r == result {
			count++
		}
	}
	for _, r := range d.ActiveResults {
		if r == result {
			count++
		}
	}
	return count
}

// Reset clears all dice state for a new adventure attempt.
func (d *DiceMechanics) Reset() {
	d.LockedResults = make([]DiceResult, 0)
	d.ActiveResults = make([]DiceResult, 0)
}
