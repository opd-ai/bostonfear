package model

// Action is an Elder Sign family action identifier.
type Action string

const (
	ActionAcquireItem Action = "acquireItem"
	ActionResolveTask Action = "resolveTask"
	ActionUseSpell    Action = "useSpell"
)

// InvestigatorState is the Elder Sign-specific player state envelope.
type InvestigatorState struct {
	PlayerID         string
	Stamina          int
	Sanity           int
	ElderSigns       int
	ActionsRemaining int
}

// IsZero reports whether the state has not been initialized.
func (s InvestigatorState) IsZero() bool {
	return s.PlayerID == ""
}

// ElderSignGameState extends the base game state with Elder Sign-specific mechanics:
// adventure deck, dice tower state, and museum doom tracker.
type ElderSignGameState struct {
	// CurrentPlayer is the player ID whose turn it is
	CurrentPlayer string

	// Doom tracks Ancient One awakening progress (0-12)
	Doom int

	// AdventureDeck contains active adventure cards available for investigation
	AdventureDeck []AdventureCard

	// DiscardedAdventures contains completed or failed adventure cards
	DiscardedAdventures []AdventureCard

	// DicePool tracks the current dice state for the active investigator
	DicePool *DicePool

	// ElderSignTokensAwarded counts total Elder Sign tokens claimed by investigators
	ElderSignTokensAwarded int

	// GamePhase indicates current phase: "setup", "playing", "victory", "defeat"
	GamePhase string

	// Investigators maps player IDs to their Elder Sign state
	Investigators map[string]InvestigatorState
}

// AdventureCard represents a museum location with investigation tasks.
// Investigators attempt adventures by rolling dice to satisfy task requirements.
type AdventureCard struct {
	// ID uniquely identifies this adventure card
	ID string

	// Name is the display name of the adventure location
	Name string

	// Tasks are the ordered list of dice requirements to complete this adventure.
	// Each task specifies required die results (e.g., "2 red, 1 lore").
	Tasks []AdventureTask

	// Rewards are granted when all tasks are completed successfully.
	// May include Elder Sign tokens, items, sanity/stamina restoration.
	Rewards []Reward

	// Penalties are applied when the adventure fails or is abandoned.
	// May include doom increments, resource loss, monster spawns.
	Penalties []Penalty
}

// AdventureTask represents a single dice requirement within an adventure.
type AdventureTask struct {
	// RequiredResults specifies die results needed to pass this task.
	// Map keys are die result types (red, green, yellow, lore).
	// Map values are the count required.
	RequiredResults map[string]int

	// Description is optional flavor text for the task
	Description string
}

// Reward represents a benefit gained from completing an adventure.
type Reward struct {
	// Type specifies the reward category: "elderSign", "item", "stamina", "sanity"
	Type string

	// Value is the numeric value for stamina/sanity rewards
	Value int

	// ItemID is the specific item awarded (if Type == "item")
	ItemID string
}

// Penalty represents a consequence of failing or abandoning an adventure.
type Penalty struct {
	// Type specifies the penalty category: "doom", "stamina", "sanity", "monster"
	Type string

	// Value is the numeric value for doom/stamina/sanity penalties
	Value int

	// MonsterID is the specific monster spawned (if Type == "monster")
	MonsterID string
}

// DicePool tracks locked and unlocked dice during adventure resolution.
// Investigators roll dice, lock favorable results, and reroll unlocked dice
// until they satisfy all adventure tasks or run out of attempts.
type DicePool struct {
	// LockedResults stores dice that have been secured by the player.
	// Locked dice cannot be rerolled and persist across task attempts.
	LockedResults []string

	// ActiveResults stores the current roll outcome for unlocked dice.
	// These dice may be rerolled or locked on the player's turn.
	ActiveResults []string

	// AvailableDice is the total number of dice in the pool (typically 6).
	// May be modified by items or investigator abilities.
	AvailableDice int

	// RollsRemaining counts attempts left before the adventure fails.
	// Standard limit is 3 rolls per adventure.
	RollsRemaining int
}

// IsComplete returns true if the dice pool has satisfied all requirements.
// This is determined by the adventure card's task requirements.
func (d *DicePool) IsComplete() bool {
	// Placeholder: actual completion logic requires adventure card context
	return false
}

// Reset clears all dice state for a new adventure attempt.
func (d *DicePool) Reset() {
	d.LockedResults = nil
	d.ActiveResults = nil
	d.RollsRemaining = 3
}
