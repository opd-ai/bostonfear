package rules

import "fmt"

// AncientOne represents a Great Old One threatening the world.
// Each Ancient One has unique mechanics, awakening conditions, and abilities.
type AncientOne struct {
	ID          string
	Name        string
	Description string
	DoomTrack   int  // Number of doom tokens before awakening (usually 12)
	IsAwakened  bool // Whether the Ancient One has awakened

	// Combat stats (used when awakened)
	CombatRating int // Difficulty of combat tests
	Horror       int // Sanity loss when encountering
	Damage       int // Health loss from attacks

	// Special abilities
	Abilities []AncientOneAbility
}

// AncientOneAbility represents a special power of an Ancient One.
type AncientOneAbility struct {
	Name        string
	Description string
	Trigger     AbilityTrigger
	Effect      string
}

// AbilityTrigger defines when an Ancient One ability activates.
type AbilityTrigger string

const (
	// TriggerOnAwaken fires when the Ancient One awakens.
	TriggerOnAwaken AbilityTrigger = "onAwaken"

	// TriggerEachMythos fires during every Mythos phase.
	TriggerEachMythos AbilityTrigger = "eachMythos"

	// TriggerWhenGateOpens fires whenever a new gate opens.
	TriggerWhenGateOpens AbilityTrigger = "whenGateOpens"

	// TriggerInCombat fires during combat with the Ancient One.
	TriggerInCombat AbilityTrigger = "inCombat"

	// TriggerPassive indicates a continuous effect.
	TriggerPassive AbilityTrigger = "passive"
)

// AwakeningCondition represents conditions that trigger Ancient One awakening.
type AwakeningCondition struct {
	// DoomReachesMax triggers when doom track fills (typically 12).
	DoomReachesMax bool

	// TooManyGates triggers if gate count exceeds threshold.
	TooManyGates int

	// TooFewInvestigators triggers if investigator count drops below minimum.
	TooFewInvestigators int

	// CustomCondition allows scenario-specific awakening triggers.
	CustomCondition string
}

// AncientOneState tracks the current state of the Ancient One.
type AncientOneState struct {
	Current           AncientOne
	CurrentDoom       int
	AwakeningProgress int // Some Ancient Ones have staged awakenings
}

// NewAncientOneState creates initial state for an Ancient One.
func NewAncientOneState(ancientOne AncientOne) *AncientOneState {
	return &AncientOneState{
		Current:           ancientOne,
		CurrentDoom:       0,
		AwakeningProgress: 0,
	}
}

// AddDoom increments the doom counter.
// Returns true if Ancient One should awaken.
func (aos *AncientOneState) AddDoom(amount int) bool {
	aos.CurrentDoom += amount
	if aos.CurrentDoom >= aos.Current.DoomTrack {
		return true
	}
	return false
}

// Awaken triggers the Ancient One awakening.
func (aos *AncientOneState) Awaken() error {
	if aos.Current.IsAwakened {
		return fmt.Errorf("ancient One is already awakened")
	}
	aos.Current.IsAwakened = true
	return nil
}

// IsAwakened checks if the Ancient One has awakened.
func (aos *AncientOneState) IsAwakened() bool {
	return aos.Current.IsAwakened
}

// GetDoomProgress returns current doom and maximum doom.
func (aos *AncientOneState) GetDoomProgress() (current, max int) {
	return aos.CurrentDoom, aos.Current.DoomTrack
}

// CheckAwakeningCondition evaluates if awakening should occur.
func (aos *AncientOneState) CheckAwakeningCondition(condition AwakeningCondition, gateCount, investigatorCount int) bool {
	// Check doom threshold
	if condition.DoomReachesMax && aos.CurrentDoom >= aos.Current.DoomTrack {
		return true
	}

	// Check gate threshold
	if condition.TooManyGates > 0 && gateCount >= condition.TooManyGates {
		return true
	}

	// Check investigator threshold
	if condition.TooFewInvestigators > 0 && investigatorCount <= condition.TooFewInvestigators {
		return true
	}

	return false
}

// GetAbilities returns all abilities for the current Ancient One.
func (aos *AncientOneState) GetAbilities() []AncientOneAbility {
	return aos.Current.Abilities
}

// GetAbilitiesForTrigger returns abilities that match the specified trigger.
func (aos *AncientOneState) GetAbilitiesForTrigger(trigger AbilityTrigger) []AncientOneAbility {
	var matching []AncientOneAbility
	for _, ability := range aos.Current.Abilities {
		if ability.Trigger == trigger {
			matching = append(matching, ability)
		}
	}
	return matching
}

// String implements Stringer for AncientOne.
func (ao AncientOne) String() string {
	status := "Dormant"
	if ao.IsAwakened {
		status = "Awakened"
	}
	return fmt.Sprintf("Ancient One: %s [%s] (Doom: %d)", ao.Name, status, ao.DoomTrack)
}

// PredefinedAncientOnes returns example Ancient Ones for testing.
// In a full implementation, these would be loaded from content files.
func PredefinedAncientOnes() []AncientOne {
	return []AncientOne{
		{
			ID:           "azathoth",
			Name:         "Azathoth",
			Description:  "The Daemon Sultan, Blind Idiot God at the center of the universe",
			DoomTrack:    12,
			IsAwakened:   false,
			CombatRating: 5,
			Horror:       3,
			Damage:       3,
			Abilities: []AncientOneAbility{
				{
					Name:        "Endless Night",
					Description: "Investigators cannot regain Sanity",
					Trigger:     TriggerPassive,
					Effect:      "sanity_regen_disabled",
				},
			},
		},
		{
			ID:           "cthulhu",
			Name:         "Cthulhu",
			Description:  "The Great Dreamer in R'lyeh",
			DoomTrack:    12,
			IsAwakened:   false,
			CombatRating: 4,
			Horror:       3,
			Damage:       2,
			Abilities: []AncientOneAbility{
				{
					Name:        "Dreams of Madness",
					Description: "All investigators lose 1 Sanity during Mythos phase",
					Trigger:     TriggerEachMythos,
					Effect:      "all_lose_sanity_1",
				},
			},
		},
		{
			ID:           "shub-niggurath",
			Name:         "Shub-Niggurath",
			Description:  "The Black Goat of the Woods with a Thousand Young",
			DoomTrack:    12,
			IsAwakened:   false,
			CombatRating: 4,
			Horror:       2,
			Damage:       3,
			Abilities: []AncientOneAbility{
				{
					Name:        "Spawn of Corruption",
					Description: "When a gate opens, place an additional monster",
					Trigger:     TriggerWhenGateOpens,
					Effect:      "spawn_extra_monster",
				},
			},
		},
		{
			ID:           "yog-sothoth",
			Name:         "Yog-Sothoth",
			Description:  "The Gate and the Key",
			DoomTrack:    12,
			IsAwakened:   false,
			CombatRating: 5,
			Horror:       2,
			Damage:       3,
			Abilities: []AncientOneAbility{
				{
					Name:        "Beyond Time",
					Description: "Gates cannot be closed except by spending Elder Signs",
					Trigger:     TriggerPassive,
					Effect:      "gates_require_elder_signs",
				},
			},
		},
		{
			ID:           "nyarlathotep",
			Name:         "Nyarlathotep",
			Description:  "The Crawling Chaos, Messenger of the Outer Gods",
			DoomTrack:    10,
			IsAwakened:   false,
			CombatRating: 5,
			Horror:       3,
			Damage:       2,
			Abilities: []AncientOneAbility{
				{
					Name:        "Masks of Chaos",
					Description: "Investigators must pass a Horror check during each Mythos phase",
					Trigger:     TriggerEachMythos,
					Effect:      "all_horror_check",
				},
				{
					Name:        "Shapeshifter",
					Description: "Gains +1 combat difficulty for each defeated investigator",
					Trigger:     TriggerInCombat,
					Effect:      "combat_scaling",
				},
			},
		},
	}
}
