package model

import "github.com/opd-ai/bostonfear/serverengine/eldritchhorror/rules"

// Action is an Eldritch Horror family action identifier.
type Action string

const (
	ActionTravel    Action = "travel"
	ActionLocal     Action = "localAction"
	ActionComponent Action = "componentAction"
	ActionRest      Action = "rest"
	ActionTrade     Action = "trade"
	ActionPrepare   Action = "prepare"
	ActionResearch  Action = "research"
)

// InvestigatorState is the Eldritch Horror-specific player state envelope.
type InvestigatorState struct {
	PlayerID         string
	Location         rules.City
	Health           int
	Sanity           int
	Focus            int
	Clues            int
	Money            int
	ActionsRemaining int
	ShipTickets      int
	TrainTickets     int
}

// IsZero reports whether the state has not been initialized.
func (s InvestigatorState) IsZero() bool {
	return s.PlayerID == ""
}

// EldritchGameState extends base game state with Eldritch Horror-specific elements.
// This includes the global map, active mysteries, gates, monsters, and Ancient One status.
type EldritchGameState struct {
	// Core game identifiers
	GameID        string
	CurrentPlayer string
	TurnNumber    int
	PhaseNumber   int
	GamePhase     string

	// Investigators and their positions
	Investigators map[string]InvestigatorState

	// Global map state
	GlobalMap GlobalMapState

	// Mystery progression
	MysteryDeck MysteryDeckState

	// Ancient One
	AncientOne AncientOneState

	// Doom and gates
	Doom     int
	Gates    []GateState
	Monsters []MonsterState

	// Win/lose conditions
	GameOver   bool
	GameResult string
}

// GlobalMapState represents the current state of the world map.
type GlobalMapState struct {
	// Cities with investigators present
	InvestigatorLocations map[string]rules.City

	// Cities with gates
	GateLocations []rules.City

	// Cities with active encounters
	EncounterLocations []rules.City
}

// MysteryDeckState represents mystery progression.
type MysteryDeckState struct {
	ActiveMystery      *MysteryState
	CompletedMysteries []string
	MysteriesToSolve   int
}

// MysteryState represents the current state of an active mystery.
type MysteryState struct {
	ID               string
	Name             string
	Description      string
	CurrentStage     int
	TotalStages      int
	StageProgress    map[int]bool
	CluesContributed int
}

// AncientOneState represents the Ancient One's current status.
type AncientOneState struct {
	ID          string
	Name        string
	Description string
	DoomTrack   int
	CurrentDoom int
	IsAwakened  bool

	// Combat stats (relevant when awakened)
	CombatRating int
	Horror       int
	Damage       int

	// Active abilities
	ActiveAbilities []string
}

// GateState represents an open gate at a location.
type GateState struct {
	Location        rules.City
	Stability       int
	MonstersSpawned int
}

// MonsterState represents a monster on the board.
type MonsterState struct {
	ID          string
	Type        string
	Location    rules.City
	Toughness   int
	Horror      int
	Damage      int
	IsEngaged   bool
	EngagedWith string
}
