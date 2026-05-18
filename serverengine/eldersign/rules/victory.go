package rules

// VictoryCondition defines Elder Sign win conditions.
// Unlike Arkham Horror's clue-gathering objective, Elder Sign requires
// sealing museum gates with Elder Sign tokens before the Ancient One awakens.
type VictoryCondition struct {
	// RequiredElderSigns is the number of Elder Sign tokens needed to win.
	// Standard game requires sealing 6 gates (11 gates for harder Ancient Ones).
	RequiredElderSigns int

	// GatesSealed tracks the current number of sealed gates.
	GatesSealed int
}

// DefaultVictoryCondition returns the standard Elder Sign win requirement.
func DefaultVictoryCondition() VictoryCondition {
	return VictoryCondition{
		RequiredElderSigns: 6,
		GatesSealed:        0,
	}
}

// IsVictorious checks if investigators have sealed enough gates to win.
func (v VictoryCondition) IsVictorious() bool {
	return v.GatesSealed >= v.RequiredElderSigns
}

// DefeatCondition defines Elder Sign loss conditions.
// Unlike Arkham Horror where doom reaches 12, Elder Sign has multiple defeat triggers.
type DefeatCondition struct {
	// DoomLevel is the current doom counter (0-12).
	DoomLevel int

	// MaxDoom is the threshold where the Ancient One awakens (typically 12).
	MaxDoom int

	// InvestigatorsDefeated tracks the count of eliminated investigators.
	InvestigatorsDefeated int

	// TotalInvestigators is the starting investigator count.
	TotalInvestigators int
}

// DefaultDefeatCondition returns the standard Elder Sign loss thresholds.
func DefaultDefeatCondition() DefeatCondition {
	return DefeatCondition{
		DoomLevel:             0,
		MaxDoom:               12,
		InvestigatorsDefeated: 0,
		TotalInvestigators:    1, // Will be updated when players join
	}
}

// IsDefeated checks if the game has reached a loss condition.
// Elder Sign loses if doom reaches 12 OR all investigators are eliminated.
func (d DefeatCondition) IsDefeated() bool {
	// Loss condition 1: Ancient One awakens (doom at maximum)
	if d.DoomLevel >= d.MaxDoom {
		return true
	}

	// Loss condition 2: All investigators defeated
	if d.TotalInvestigators > 0 && d.InvestigatorsDefeated >= d.TotalInvestigators {
		return true
	}

	return false
}

// IncrementDoom raises the doom counter by the specified amount.
// Returns the new doom level after increment.
func (d *DefeatCondition) IncrementDoom(amount int) int {
	d.DoomLevel += amount
	if d.DoomLevel > d.MaxDoom {
		d.DoomLevel = d.MaxDoom
	}
	return d.DoomLevel
}

// DecrementDoom lowers the doom counter by the specified amount.
// Used when investigators successfully cast wards or seal gates.
// Returns the new doom level after decrement.
func (d *DefeatCondition) DecrementDoom(amount int) int {
	d.DoomLevel -= amount
	if d.DoomLevel < 0 {
		d.DoomLevel = 0
	}
	return d.DoomLevel
}
