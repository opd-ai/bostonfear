// Package model defines the investigator and resource rules for Arkham Horror.
// S4 Migration: Investigator and resource model rules - arkhamhorror module ownership.
package model

// ResourceBounds defines the valid ranges for each resource type
type ResourceBounds struct {
	HealthMax  int
	SanityMax  int
	CluesMax   int
	MoneyMax   int
	FocusMax   int
	RemnantMax int
}

// ArkhamHorrorResourceBounds defines the official AH3e resource limits
var ArkhamHorrorResourceBounds = ResourceBounds{
	HealthMax:  8,
	SanityMax:  8,
	CluesMax:   5,
	MoneyMax:   10,
	FocusMax:   12,
	RemnantMax: 6,
}

// ClampResourceToBounds ensures a resource value stays within valid range
func ClampResourceToBounds(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// ClampHealth clamps a health value to [0, MaxHealth]
func ClampHealth(value int) int {
	return ClampResourceToBounds(value, 0, ArkhamHorrorResourceBounds.HealthMax)
}

// ClampSanity clamps a sanity value to [0, MaxSanity]
func ClampSanity(value int) int {
	return ClampResourceToBounds(value, 0, ArkhamHorrorResourceBounds.SanityMax)
}

// ClampClues clamps a clues value to [0, MaxClues]
func ClampClues(value int) int {
	return ClampResourceToBounds(value, 0, ArkhamHorrorResourceBounds.CluesMax)
}

// ClampMoney clamps a money value to [0, MaxMoney]
func ClampMoney(value int) int {
	return ClampResourceToBounds(value, 0, ArkhamHorrorResourceBounds.MoneyMax)
}

// ClampFocus clamps a focus value to [0, MaxFocus]
func ClampFocus(value int) int {
	return ClampResourceToBounds(value, 0, ArkhamHorrorResourceBounds.FocusMax)
}

// ClampRemnants clamps a remnants value to [0, MaxRemnants]
func ClampRemnants(value int) int {
	return ClampResourceToBounds(value, 0, ArkhamHorrorResourceBounds.RemnantMax)
}

// InvestigatorDefeated checks if an investigator is defeated
// (Health or Sanity at 0 requires them to be moved to Lost in Time and Space)
func InvestigatorDefeated(health, sanity int) bool {
	return health <= 0 || sanity <= 0
}

// S4: Investigator ability constants defined as module-owned data
type InvestigatorAbility struct {
	Name          string
	HealthCost    int
	SanityCost    int
	ClueGain      int
	HealthGain    int
	SanityGain    int
	FocusGain     int
	DoomReduct    int
	DrawEncounter bool
}
